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
			panic(err)
		}

		usageData, err := GetUsageData(clientSet)
		if err != nil {
			panic(err)
		}

		for nodeName, usage := range usageData {
			fmt.Printf("Node %-16s  CPU %s   Memory %s\n",
				nodeName,
				displayProcessorUsage(usage.CPU),
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

	maxCPUValues := map[string]int64{}
	maxMemValues := map[string]int64{}

	nodeList, err := api.Nodes().List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, node := range nodeList.Items {
		maxCPUValues[node.Name] = int64(node.Status.Capacity.Cpu().MilliValue())
		maxMemValues[node.Name] = int64(node.Status.Capacity.Memory().MilliValue())
	}

	// ---

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
		result[nodeName] = UsageData{
			CPU: UsageEntry{
				Used: parseQuantity(node.Usage.CPU),
				Max:  maxCPUValues[nodeName]},
			Memory: UsageEntry{
				Used: parseQuantity(node.Usage.Memory),
				Max:  maxMemValues[nodeName]},
		}
	}

	return result, nil
}

func parseQuantity(input string) int64 {
	quantity := resource.MustParse(input)
	return quantity.MilliValue()
}

func displayProcessorUsage(input UsageEntry) string {
	textLength := 64
	usage := float64(input.Used) / float64(input.Max)

	var buf bytes.Buffer
	buf.WriteString("[")

	buf.WriteString(printProgressBar(textLength, usage))

	buf.WriteString(fmt.Sprintf("%5.1f%%", usage*100.0))
	buf.WriteString("]")

	return buf.String()
}

func displayMemoryUsage(input UsageEntry) string {
	textLength := 64
	usage := float64(input.Used) / float64(input.Max)

	var buf bytes.Buffer
	buf.WriteString("[")

	buf.WriteString(printProgressBar(textLength, usage))

	buf.WriteString(fmt.Sprintf(" %s/%s", havener.HumanReadableSize(input.Used/1000), havener.HumanReadableSize(input.Max/1000)))
	buf.WriteString("]")

	return buf.String()
}

func printProgressBar(width int, usage float64) string {
	symbol := "■"

	var buf bytes.Buffer

	marks := int(usage * float64(width))
	for i := 0; i < width; i++ {
		if i <= marks {
			buf.WriteString(symbol)

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
