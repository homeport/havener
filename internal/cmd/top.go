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

	"github.com/HeavyWombat/dyff/pkg/v1/bunt"

	"github.com/spf13/cobra"
	"github.ibm.com/hatch/havener/pkg/havener"

	"k8s.io/client-go/kubernetes"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	Short: "TBD",
	Long:  `TBD`,
	Run: func(cmd *cobra.Command, args []string) {
		clientSet, err := havener.OutOfClusterAuthentication()
		if err != nil {
			havener.ExitWithError("unable to get access to cluster", err)
		}

		usageData, err := GetUsageData(clientSet)
		if err != nil {
			havener.ExitWithError("unable to get cluster usage data", err)
		}

		for _, nodeName := range sortedKeyList(usageData) {
			usage := usageData[nodeName]

			fmt.Printf("%s %-16s  %s %s  %s %s\n",
				bunt.Style("Node", bunt.Bold),
				nodeName,

				bunt.Style("CPU", bunt.Bold),
				displayProcessorUsage(usage.CPU),

				bunt.Style("Memory", bunt.Bold),
				displayMemoryUsage(usage.Memory))
		}
	},
}

func init() {
	rootCmd.AddCommand(topCmd)
}

type RawHeapsterMetrics struct {
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

type UsageEntry struct {
	Used int64
	Max  int64
}

type UsageData struct {
	CPU    UsageEntry
	Memory UsageEntry
}

// GetUsageData ...
// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/resource-metrics-api.md
func GetUsageData(clientSet *kubernetes.Clientset) (map[string]UsageData, error) {
	result := map[string]UsageData{}
	api := clientSet.CoreV1()

	// ---

	currentCPUValues := map[string]int64{}
	currentMemValues := map[string]int64{}

	// TODO Refactor into one or two lines to make it more readable
	prefix := "/apis"
	metricsGv := schema.GroupVersion{Group: "metrics", Version: "v1alpha1"}
	groupVersion := fmt.Sprintf("%s/%s", metricsGv.Group, metricsGv.Version)
	metricsRoot := fmt.Sprintf("%s/%s", prefix, groupVersion)

	path := fmt.Sprintf("%s/nodes/", metricsRoot)
	params := map[string]string{}

	data, err := api.
		Services(heapsterNamespace).
		ProxyGet(heapsterScheme, heapsterService, heapsterPort, path, params).
		DoRaw()
	if err != nil {
		return nil, err
	}

	var metrics RawHeapsterMetrics
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		return nil, err
	}

	for _, node := range metrics.Items {
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

func displayProcessorUsage(input UsageEntry) string {
	textLength := 64

	if input.Used > input.Max {
		input.Used = input.Max
	}

	usage := float64(input.Used) / float64(input.Max)

	var buf bytes.Buffer
	buf.WriteString("[")

	if input.Used > 0 {
		infoText := fmt.Sprintf("%5.1f%%", usage*100.0)

		buf.WriteString(printProgressBar(textLength-len(infoText), usage))
		buf.WriteString(infoText)

	} else {
		buf.WriteString(centerText("no data points", textLength))
	}

	buf.WriteString("]")

	return buf.String()
}

func displayMemoryUsage(input UsageEntry) string {
	textLength := 64

	if input.Used > input.Max {
		input.Used = input.Max
	}

	usage := float64(input.Used) / float64(input.Max)

	var buf bytes.Buffer
	buf.WriteString("[")

	if input.Used > 0 {
		infoText := fmt.Sprintf(" %s/%s",
			havener.HumanReadableSize(input.Used/1000),
			havener.HumanReadableSize(input.Max/1000))

		buf.WriteString(printProgressBar(textLength-len(infoText), usage))
		buf.WriteString(infoText)

	} else {
		buf.WriteString(centerText("no data points", textLength))
	}

	buf.WriteString("]")

	return buf.String()
}

func printProgressBar(width int, usage float64) string {
	symbol := "■"

	var buf bytes.Buffer

	marks := int(usage * float64(width))
	for i := 0; i < width; i++ {
		if i < marks {
			switch bunt.UseColors() {
			case true:
				buf.WriteString(bunt.Colorize(symbol, bunt.LimeGreen.BlendLab(bunt.OrangeRed, float64(i)/float64(width))))

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

	return buf.String()
}

func centerText(text string, length int) string {
	strLen := len(text)
	if strLen > length {
		return text
	}

	remainder := length - strLen
	left := int(math.Ceil(float64(remainder) / 2.0))
	right := int(math.Floor(float64(remainder) / 2.0))

	return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
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
