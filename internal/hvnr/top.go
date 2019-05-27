// Copyright © 2019 The Havener
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package hvnr

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/homeport/gonvenience/pkg/v1/bunt"
	"github.com/homeport/gonvenience/pkg/v1/neat"
	"github.com/homeport/gonvenience/pkg/v1/term"
)

const (
	nodeCaption      = "Node"
	processorCaption = "CPU"
	memoryCaption    = "Memory"
)

var (
	heapsterNamespace = "kube-system"
	heapsterService   = "heapster"
	heapsterScheme    = "http"
	heapsterPort      = ""

	topX = 20
)

type nodeMetrics struct {
	Items []struct {
		Metadata struct {
			Name              string    `json:"name"`
			CreationTimestamp time.Time `json:"creationTimestamp"`
		} `json:"metadata"`
		Timestamp time.Time `json:"timestamp"`
		Window    string    `json:"window"`
		Usage     struct {
			CPU    string `json:"cpu"`
			Memory string `json:"memory"`
		} `json:"usage"`
	} `json:"items"`
}

type podMetrics struct {
	Items []struct {
		Metadata struct {
			Name              string    `json:"name"`
			Namespace         string    `json:"namespace"`
			CreationTimestamp time.Time `json:"creationTimestamp"`
		} `json:"metadata"`
		Timestamp  time.Time `json:"timestamp"`
		Window     string    `json:"window"`
		Containers []struct {
			Name  string `json:"name"`
			Usage struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
			} `json:"usage"`
		} `json:"containers"`
	} `json:"items"`
}

type usageEntry struct {
	Used int64
	Max  int64
}

type usageData struct {
	CPU    usageEntry
	Memory usageEntry
}

// CompileNodeStats renders a string with node usage statistics
func CompileNodeStats(client kubernetes.Interface) (string, error) {
	var buf bytes.Buffer
	out := bufio.NewWriter(&buf)

	usageData, err := getNodeUsageData(client)
	if err != nil {
		return "", err
	}

	maxLength := 0
	for nodeName := range usageData {
		if length := len(nodeName); length > maxLength {
			maxLength = length
		}
	}

	barLength := 0.5 * float64(term.GetTerminalWidth()-
		len(nodeCaption)-1- // node caption (and space)
		maxLength- // longest node name
		3- // delimiter
		len(processorCaption)-1- // processor caption (and space)
		3- // delimiter
		len(memoryCaption)-1) // memory caption (and space)

	firstBarLength, secondBarLength := int(math.Ceil(barLength)), int(math.Floor(barLength))

	data := [][]string{}
	for _, nodeName := range sortedKeyList(usageData) {
		usage := usageData[nodeName]

		data = append(data, []string{
			bunt.Sprintf("%s DimGray{%s}",
				nodeCaption,
				nodeName,
			),

			bunt.Sprintf("%s %s",
				processorCaption,
				progressBar(firstBarLength, usage.CPU, func(used, max int64) string {
					return fmt.Sprintf(" %5.1f%%", float64(used)/float64(max)*100.0)
				}),
			),

			bunt.Sprintf("%s %s",
				memoryCaption,
				progressBar(secondBarLength, usage.Memory, func(used, max int64) string {
					return fmt.Sprintf(" %s/%s",
						humanReadableSize(used/1000),
						humanReadableSize(max/1000))
				}),
			),
		})
	}

	table, err := neat.Table(data, neat.CustomSeparator("  "))
	if err != nil {
		return "", err
	}

	bunt.Fprintf(out, "_*Usage by Node*_\n%s\n", table)

	out.Flush()
	return buf.String(), nil
}

// CompilePodStats renders a string with pod usage statistics
func CompilePodStats(client kubernetes.Interface) (string, error) {
	podUsage, err := getPodUsageData(client)
	if err != nil {
		return "", err
	}

	type consumer struct {
		name string
		cpu  int64
		mem  int64
	}

	splitKey := func(key string) (string, string, string) {
		split := strings.Split(key, "/")
		return split[0], split[1], split[2]
	}

	usageByNamespace := func() []consumer {
		totalcpu, totalmem := map[string]int64{}, map[string]int64{}
		for key, value := range podUsage {
			namespace, _, _ := splitKey(key)
			if _, ok := totalcpu[namespace]; !ok {
				totalcpu[namespace] = 0
				totalmem[namespace] = 0
			}

			totalcpu[namespace] += value.CPU.Used
			totalmem[namespace] += value.Memory.Used
		}

		result := []consumer{}
		for namespace, cpu := range totalcpu {
			result = append(result, consumer{
				name: namespace,
				cpu:  cpu,
				mem:  totalmem[namespace],
			})
		}

		sort.Slice(result, func(i, j int) bool {
			return result[i].mem > result[j].mem
		})

		return result
	}()

	usageByPod := func() []consumer {
		result := []consumer{}
		for key, value := range podUsage {
			result = append(result, consumer{
				name: key,
				cpu:  value.CPU.Used,
				mem:  value.Memory.Used,
			})
		}

		sort.Slice(result, func(i, j int) bool {
			return result[i].mem > result[j].mem
		})

		if len(result) > topX {
			result = result[:topX]
		}

		return result
	}()

	usageToData := func(list []consumer) [][]string {
		result := [][]string{}
		for _, consumer := range list {
			result = append(result, []string{
				consumer.name,
				fmt.Sprintf("%.1f cores", float64(consumer.cpu)/1000.0),
				humanReadableSize(consumer.mem / 1000),
			})
		}

		return result
	}

	var buf bytes.Buffer
	out := bufio.NewWriter(&buf)

	namespaceUsageStats, err := neat.Table(usageToData(usageByNamespace),
		neat.CustomSeparator(bunt.Sprintf("DimGray{ │ }")),
		neat.AlignRight(1, 2),
	)
	if err != nil {
		return "", err
	}

	podUsageStats, err := neat.Table(usageToData(usageByPod),
		neat.CustomSeparator(bunt.Sprintf("DimGray{ │ }")),
		neat.AlignRight(1, 2),
	)
	if err != nil {
		return "", err
	}

	namespaceOutput := bunt.Sprintf("_*Usage by Namespace*_\n%s\n", namespaceUsageStats)
	podOutput := bunt.Sprintf("_*Usage by Pod*_\n%s\n", podUsageStats)

	if term.GetTerminalWidth() > getTextWidth(namespaceOutput)+getTextWidth(podOutput) {
		leftLines, rightLines := strings.Split(podOutput, "\n"), strings.Split(namespaceOutput, "\n")
		maxLines := int(math.Max(float64(len(leftLines)), float64(len(rightLines))))

		data := [][]string{}
		for i := 0; i < maxLines; i++ {
			var (
				left  string
				right string
			)

			if i < len(leftLines) {
				left = leftLines[i]
			}

			if i < len(rightLines) {
				right = rightLines[i]
			}

			data = append(data, []string{left, right})
		}

		output, err := neat.Table(data, neat.CustomSeparator("    "))
		if err != nil {
			return "", err
		}

		out.WriteString(output)

	} else {
		out.WriteString(namespaceOutput)
		out.WriteString(podOutput)
	}

	out.Flush()
	return buf.String(), nil
}

func getNodeUsageData(client kubernetes.Interface) (map[string]usageData, error) {
	// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/resource-metrics-api.md
	result := map[string]usageData{}

	// ---

	currentCPUValues := map[string]int64{}
	currentMemValues := map[string]int64{}

	nodeMetrics, err := getNodeMetrics(client)
	if err != nil {
		return nil, err
	}

	for _, node := range nodeMetrics.Items {
		nodeName := node.Metadata.Name
		currentCPUValues[nodeName] = parseQuantity(node.Usage.CPU)
		currentMemValues[nodeName] = parseQuantity(node.Usage.Memory)
	}

	// ---

	nodeList, err := client.CoreV1().Nodes().List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, node := range nodeList.Items {
		nodeName := node.Name

		result[nodeName] = usageData{
			CPU: usageEntry{
				Used: lookupValue(currentCPUValues, nodeName),
				Max:  int64(node.Status.Capacity.Cpu().MilliValue()),
			},
			Memory: usageEntry{
				Used: lookupValue(currentMemValues, nodeName),
				Max:  int64(node.Status.Capacity.Memory().MilliValue()),
			},
		}
	}

	return result, nil
}

func getPodUsageData(client kubernetes.Interface) (map[string]usageData, error) {
	result := map[string]usageData{}

	podmetrics, err := getPodMetrics(client)
	if err != nil {
		return nil, err
	}

	for _, podmetric := range podmetrics.Items {
		namespace := podmetric.Metadata.Namespace
		podname := podmetric.Metadata.Name

		for _, container := range podmetric.Containers {
			containerName := container.Name
			result[strings.Join([]string{namespace, podname, containerName}, "/")] = usageData{
				CPU: usageEntry{
					Used: parseQuantity(container.Usage.CPU),
				},
				Memory: usageEntry{
					Used: parseQuantity(container.Usage.Memory),
				},
			}
		}
	}

	return result, nil
}

func getRawHeapsterMetrics(client kubernetes.Interface, path string, params map[string]string) ([]byte, error) {
	return client.CoreV1().Services(heapsterNamespace).
		ProxyGet(heapsterScheme, heapsterService, heapsterPort, path, params).
		DoRaw()
}

func getNodeMetrics(client kubernetes.Interface) (*nodeMetrics, error) {
	data, err := getRawHeapsterMetrics(client, "/apis/metrics/v1alpha1/nodes/", map[string]string{})
	if err != nil {
		return nil, err
	}

	var metrics nodeMetrics
	if err = json.Unmarshal(data, &metrics); err != nil {
		return nil, err
	}

	return &metrics, nil
}

func getPodMetrics(client kubernetes.Interface) (*podMetrics, error) {
	var result podMetrics

	namespaceList, err := client.CoreV1().Namespaces().List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, namespace := range namespaceList.Items {
		data, err := getRawHeapsterMetrics(client, fmt.Sprintf("/apis/metrics/v1alpha1/namespaces/%s/pods", namespace.Name), map[string]string{})
		if err != nil {
			return nil, err
		}

		var metrics podMetrics
		if err = json.Unmarshal(data, &metrics); err != nil {
			return nil, err
		}

		result.Items = append(result.Items, metrics.Items...)
	}

	return &result, nil
}

func lookupValue(data map[string]int64, key string) int64 {
	if value, ok := data[key]; ok {
		return value
	}

	return -1
}

func parseQuantity(input string) int64 {
	quantity := resource.MustParse(input)
	return quantity.MilliValue()
}

func plainTextLength(text string) int {
	return utf8.RuneCountInString(bunt.RemoveAllEscapeSequences(text))
}

func centerText(text string, length int) string {
	strLen := plainTextLength(text)
	if strLen > length {
		return text
	}

	remainder := length - strLen
	left := int(math.Floor(float64(remainder) / 2.0))
	right := int(math.Ceil(float64(remainder) / 2.0))

	return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
}

func sortedKeyList(data map[string]usageData) []string {
	result := make([]string, len(data))

	i := 0
	for key := range data {
		result[i] = key
		i++
	}

	sort.Strings(result)

	return result
}

func progressBar(length int, usageEntry usageEntry, textDetails func(used, max int64) string) string {
	if usageEntry.Used < 0 {
		return "[" + centerText("no data points", length-2) + "]"
	}

	const symbol = "■"
	var buf bytes.Buffer

	details := textDetails(usageEntry.Used, usageEntry.Max)
	width := length - 2 - len(details)
	usage := float64(usageEntry.Used) / float64(usageEntry.Max)
	marks := int(usage * float64(width))

	buf.WriteString("[")
	for i := 0; i < width; i++ {
		if i < marks {
			switch bunt.UseColors() {
			case true:
				// Use smooth curved gradient:
				// http://fooplot.com/?lang=en#W3sidHlwZSI6MCwiZXEiOiIoMS1jb3MoeF4yKjMuMTQxNSkpLzIiLCJjb2xvciI6IiMwMDAwMDAifSx7InR5cGUiOjEwMDAsIndpbmRvdyI6WyIwIiwiMSIsIjAiLCIxIl19XQ--
				blendFactor := 0.5 * (1.0 - math.Cos(math.Pow(float64(i)/float64(length), 2)*math.Pi))
				buf.WriteString(bunt.Colorize(symbol, bunt.LimeGreen.BlendLab(bunt.Red, blendFactor)))

			default:
				buf.WriteString(symbol)
			}

		} else {
			switch bunt.UseColors() {
			case true:
				buf.WriteString(bunt.Colorize(symbol, bunt.DimGray))

			default:
				buf.WriteString(" ")
			}
		}
	}

	if len(details) > 0 {
		buf.WriteString(bunt.Colorize(details, bunt.Gray))
	}

	buf.WriteString("]")

	return buf.String()
}

func getTextWidth(text string) int {
	var max int
	for _, line := range strings.Split(text, "\n") {
		if len := plainTextLength(line); len > max {
			max = len
		}
	}

	return max
}

func humanReadableSize(bytes int64) string {
	var mods = []string{"Byte", "KiB", "MiB", "GiB", "TiB"}

	value := float64(bytes)
	i := 0
	for value > 1023.99999 {
		value /= 1024.0
		i++
	}

	return fmt.Sprintf("%.1f %s", value, mods[i])
}
