// Copyright © 2018 The Havener
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

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/HeavyWombat/dyff/pkg/v1/bunt"
	colorful "github.com/lucasb-eyer/go-colorful"

	"github.com/spf13/cobra"
	"github.ibm.com/hatch/havener/pkg/havener"

	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	heapsterNamespace = "kube-system"
	heapsterService   = "heapster"
	heapsterScheme    = "http"
	heapsterPort      = ""
)

// topCmd represents the top command
var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Shows CPU and Memory usage",
	Long:  `TBD`,
	Run: func(cmd *cobra.Command, args []string) {
		const nodeCaption = "Node "
		const processorCaption = "CPU "
		const memoryCaption = "Memory "
		const delimiter = "  "

		clientSet, _, err := havener.OutOfClusterAuthentication()
		if err != nil {
			havener.ExitWithError("unable to get access to cluster", err)
		}

		// --- --- ---

		usageData, err := GetNodeUsageData(clientSet)
		if err != nil {
			havener.ExitWithError("unable to get node usage metrics", err)
		}

		maxLength := 0
		for nodeName := range usageData {
			if length := len(nodeName); length > maxLength {
				maxLength = length
			}
		}

		barLength := (havener.GetTerminalWidth() -
			len(nodeCaption) -
			maxLength -
			len(delimiter) -
			len(processorCaption) -
			len(delimiter) -
			len(memoryCaption)) / 2

		fmt.Print(bunt.Style("CPU and Memory usage by Node\n", bunt.Bold, bunt.Italic))
		for _, nodeName := range sortedKeyList(usageData) {
			usage := usageData[nodeName]

			fmt.Print(bunt.Style(nodeCaption, bunt.Bold))
			fmt.Print(padRight(nodeName, maxLength))
			fmt.Print(delimiter)

			fmt.Print(bunt.Style(processorCaption, bunt.Bold))
			fmt.Print(progresBar(barLength, usage.CPU, func(used, max int64) string {
				return fmt.Sprintf(" %5.1f%%", float64(used)/float64(max)*100.0)

			}))
			fmt.Print(delimiter)

			fmt.Print(bunt.Style(memoryCaption, bunt.Bold))
			fmt.Print(progresBar(barLength, usage.Memory, func(used, max int64) string {
				return fmt.Sprintf(" %s/%s",
					havener.HumanReadableSize(used/1000),
					havener.HumanReadableSize(max/1000))
			}))
			fmt.Print("\n")
		}

		// --- --- ---

		podUsage, err := GetPodUsageData(clientSet)
		if err != nil {
			havener.ExitWithError("unable to get pod usage metrics", err)
		}

		splitKey := func(key string) (string, string, string) {
			split := strings.Split(key, "/")
			return split[0], split[1], split[2]
		}

		usedCPUOfNamespace := map[string]int64{}
		usedMemOfNamespace := map[string]int64{}
		for key, value := range podUsage {
			namespace, _, _ := splitKey(key)
			if _, ok := usedCPUOfNamespace[namespace]; !ok {
				usedCPUOfNamespace[namespace] = 0
				usedMemOfNamespace[namespace] = 0
			}

			usedCPUOfNamespace[namespace] += value.CPU.Used
			usedMemOfNamespace[namespace] += value.Memory.Used
		}

		names := []string{}
		for key := range usedCPUOfNamespace {
			names = append(names, key)
		}
		sort.Strings(names)

		fmt.Print("\n")
		fmt.Print(bunt.Style("CPU and Memory usage by Namespace\n", bunt.Bold, bunt.Italic))
		fmt.Print(usageChart(names, usedCPUOfNamespace, usedMemOfNamespace))
	},
}

func init() {
	rootCmd.AddCommand(topCmd)
}

type NodeMetrics struct {
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

type PodMetrics struct {
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

type UsageEntry struct {
	Used int64
	Max  int64
}

type UsageData struct {
	CPU    UsageEntry
	Memory UsageEntry
}

// GetNodeUsageData ...
// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/resource-metrics-api.md
func GetNodeUsageData(clientSet *kubernetes.Clientset) (map[string]UsageData, error) {
	result := map[string]UsageData{}
	api := clientSet.CoreV1()

	// ---

	currentCPUValues := map[string]int64{}
	currentMemValues := map[string]int64{}

	nodeMetrics, err := getNodeMetrics(api)
	if err != nil {
		return nil, err
	}

	for _, node := range nodeMetrics.Items {
		nodeName := node.Metadata.Name
		currentCPUValues[nodeName] = parseQuantity(node.Usage.CPU)
		currentMemValues[nodeName] = parseQuantity(node.Usage.Memory)
	}

	// ---

	nodeList, err := api.Nodes().List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, node := range nodeList.Items {
		nodeName := node.Name

		result[nodeName] = UsageData{
			CPU: UsageEntry{
				Used: lookupValue(currentCPUValues, nodeName),
				Max:  int64(node.Status.Capacity.Cpu().MilliValue()),
			},
			Memory: UsageEntry{
				Used: lookupValue(currentMemValues, nodeName),
				Max:  int64(node.Status.Capacity.Memory().MilliValue()),
			},
		}
	}

	return result, nil
}

func GetPodUsageData(clientSet *kubernetes.Clientset) (map[string]UsageData, error) {
	result := map[string]UsageData{}

	podmetrics, err := getPodMetrics(clientSet.CoreV1())
	if err != nil {
		return nil, err
	}

	for _, podmetric := range podmetrics.Items {
		namespace := podmetric.Metadata.Namespace
		podname := podmetric.Metadata.Name

		for _, container := range podmetric.Containers {
			containerName := container.Name
			result[strings.Join([]string{namespace, podname, containerName}, "/")] = UsageData{
				CPU: UsageEntry{
					Used: parseQuantity(container.Usage.CPU),
				},
				Memory: UsageEntry{
					Used: parseQuantity(container.Usage.Memory),
				},
			}
		}
	}

	return result, nil
}

func getRawHeapsterMetrics(api corev1.CoreV1Interface, path string, params map[string]string) ([]byte, error) {
	return api.Services(heapsterNamespace).
		ProxyGet(heapsterScheme, heapsterService, heapsterPort, path, params).
		DoRaw()
}

func getNodeMetrics(api corev1.CoreV1Interface) (*NodeMetrics, error) {
	data, err := getRawHeapsterMetrics(api, "/apis/metrics/v1alpha1/nodes/", map[string]string{})
	if err != nil {
		return nil, err
	}

	var metrics NodeMetrics
	if err = json.Unmarshal(data, &metrics); err != nil {
		return nil, err
	}

	return &metrics, nil
}

func getPodMetrics(api corev1.CoreV1Interface) (*PodMetrics, error) {
	var result PodMetrics

	namespaceList, err := api.Namespaces().List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, namespace := range namespaceList.Items {
		data, err := getRawHeapsterMetrics(api, fmt.Sprintf("/apis/metrics/v1alpha1/namespaces/%s/pods", namespace.Name), map[string]string{})
		if err != nil {
			return nil, err
		}

		var metrics PodMetrics
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

func padRight(text string, length int) string {
	strLen := len(text)

	if strLen > length {
		return text
	}

	return text + strings.Repeat(" ", length-strLen)
}

func sortedKeyList(data map[string]UsageData) []string {
	result := make([]string, len(data), len(data))

	i := 0
	for key := range data {
		result[i] = key
		i++
	}

	sort.Strings(result)

	return result
}

func progresBar(length int, usageEntry UsageEntry, textDetails func(used, max int64) string) string {
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

func usageChart(names []string, cpu map[string]int64, memory map[string]int64) string {
	var cpuSum int64
	for _, used := range cpu {
		cpuSum += used
	}

	var memSum int64
	for _, used := range memory {
		memSum += used
	}

	colors := []colorful.Color{
		bunt.OrangeRed,
		bunt.Aqua,
		bunt.Moccasin,
		bunt.DeepPink,
		bunt.DarkSlateGray,
		bunt.PaleGreen,
		bunt.SeaGreen,
		bunt.Olive,
		bunt.PaleGreen,
		bunt.Purple,
	}

	const symbol = "■"
	var buf bytes.Buffer

	possibleRunes := havener.GetTerminalWidth() - 2

	chart := func(input map[string]int64, totalSum int64) {
		var chartBuf bytes.Buffer
		for idx, namespace := range names {
			if idx > len(colors) {
				panic("ran out of colors")
			}

			count := int(math.Floor(float64(input[namespace]) / float64(totalSum) * float64(possibleRunes)))
			chartBuf.WriteString(bunt.Colorize(strings.Repeat(symbol, count), colors[idx]))
		}

		buf.WriteString(centerText(chartBuf.String(), possibleRunes))
	}

	buf.WriteString("[")
	chart(cpu, cpuSum)
	buf.WriteString("]")
	buf.WriteString("\n")

	buf.WriteString("[")
	chart(memory, memSum)
	buf.WriteString("]")
	buf.WriteString("\n")

	for idx, namespace := range names {
		buf.WriteString(fmt.Sprintf("  %s %-16s  CPU %s, Memory %s\n",
			bunt.Colorize(symbol, colors[idx]),
			namespace,
			fmt.Sprintf("%.1f cores (%.1f%%)", float64(cpu[namespace])/1000.0, float64(cpu[namespace])/float64(cpuSum)*100.0),
			fmt.Sprintf("%s (%.1f%%)", havener.HumanReadableSize(memory[namespace]/1000), float64(memory[namespace])/float64(memSum)*100.0),
		))
	}

	return buf.String()
}
