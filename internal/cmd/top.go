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
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/neat"
	"github.com/gonvenience/term"
	"github.com/gonvenience/text"
	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
)

var (
	cycles                 = -1
	interval               = 2
	maxContainerNameLength = 64
)

// topCmd represents the top command
var topCmd = &cobra.Command{
	Use:           "top",
	Short:         "Shows CPU and Memory usage",
	Long:          `Shows more detailed CPU and Memory usage details`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Fall back to default interval if unsupported interval was specified
		if interval <= 0 {
			interval = 2
		}

		// Restrict the output to one single measurement in case of a dumb
		// terminal or used inside Concourse
		if term.IsDumbTerminal() || term.IsGardenContainer() {
			cycles = 1
		}

		hvnr, err := havener.NewHavener()
		if err != nil {
			return err
		}

		term.HideCursor()
		defer term.ShowCursor()

		f := func() error {
			top, err := hvnr.TopDetails()
			if err != nil {
				return err
			}

			nodeDetails := renderNodeDetails(top)
			namespaceDetails := renderNamespaceDetails(top)
			availableLines := term.GetTerminalHeight() - lines(nodeDetails) - lines(namespaceDetails)
			topContainers := renderTopContainers(top, availableLines-4)

			fmt.Print(
				"\x1b[H",
				"\x1b[2J",
				nodeDetails,
				namespaceDetails,
				topContainers,
			)

			return nil
		}

		// Make sure to start with a print
		if err := f(); err != nil {
			return err
		}

		var ticker = time.NewTicker(time.Duration(interval) * time.Second)
		var timeout = make(<-chan time.Time)
		if cycles > 0 {
			timeout = time.After(time.Duration(interval*cycles) * time.Second)
		}

		for {
			select {
			case <-ticker.C:
				if err := f(); err != nil {
					return err
				}

			case <-timeout:
				return nil
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(topCmd)

	topCmd.PersistentFlags().IntVarP(&cycles, "cycles", "c", -1, "number of cycles to run, negative numbers means infinite cycles")
	topCmd.PersistentFlags().IntVarP(&interval, "interval", "i", 2, "interval between measurements in seconds")

	topCmd.Flags().SortFlags = false
	topCmd.PersistentFlags().SortFlags = false
}

func renderNodeDetails(topDetails *havener.TopDetails) string {
	progressBarWidth, maxLength := func() (int, int) {
		var maxLength int
		for name := range topDetails.Nodes {
			if length := len(name); length > maxLength {
				maxLength = length
			}
		}

		// Subtract 6, 2 for the prefix, and 4 more for spaces
		return (term.GetTerminalWidth() - maxLength - 6) / 2, maxLength
	}()

	var buf bytes.Buffer
	for _, name := range sortedNodeList(topDetails) {
		stats := topDetails.Nodes[name]

		cpuUsage := fmt.Sprintf("%.1f%%", float64(stats.UsedCPU)/float64(stats.TotalCPU)*100.0)
		memUsage := fmt.Sprintf("%s/%s", humanReadableSize(stats.UsedMemory), humanReadableSize(stats.TotalMemory))
		bunt.Fprintf(&buf, "%s  %s  %s\n",
			fill(name, maxLength),
			renderProgressBar(stats.UsedCPU, stats.TotalCPU, "CPU", cpuUsage, progressBarWidth),
			renderProgressBar(stats.UsedMemory, stats.TotalMemory, "Memory", memUsage, progressBarWidth),
		)
	}

	return neat.ContentBox(
		"CPU and Memory usage by Node",
		buf.String(),
		neat.HeadlineColor(bunt.DimGray),
	)
}

func renderNamespaceDetails(topDetails *havener.TopDetails) string {
	type sum struct {
		name string
		cpu  int64
		mem  int64
	}

	sumsPerNamespace := func() []sum {
		result := []sum{}
		for namespace, podMap := range topDetails.Containers {
			var cpu, mem int64
			for _, containerMap := range podMap {
				for _, usage := range containerMap {
					cpu += usage.UsedCPU
					mem += usage.UsedMemory
				}
			}

			result = append(result, sum{
				name: namespace,
				cpu:  cpu,
				mem:  mem,
			})
		}

		sort.Slice(result, func(i, j int) bool {
			return result[i].mem > result[j].mem
		})

		return result
	}()

	var totalCPU, totalMem int64
	for _, stats := range topDetails.Nodes {
		totalCPU += stats.TotalCPU
		totalMem += stats.TotalMemory
	}

	progressBarWidth, maxLength := func() (int, int) {
		var maxLength int
		for namespace := range topDetails.Containers {
			if length := len(namespace); length > maxLength {
				maxLength = length
			}
		}

		// Subtract 6, 2 for the prefix, and 4 more for spaces
		return (term.GetTerminalWidth() - maxLength - 6) / 2, maxLength
	}()

	var buf bytes.Buffer
	for _, sums := range sumsPerNamespace {
		cpuUsage := fmt.Sprintf("%.1f%%", float64(sums.cpu)/float64(totalCPU)*100.0)
		memUsage := fmt.Sprintf("%s/%s", humanReadableSize(sums.mem), humanReadableSize(totalMem))
		bunt.Fprintf(&buf, "%s  %s  %s\n",
			fill(sums.name, maxLength),
			renderProgressBar(sums.cpu, totalCPU, "CPU", cpuUsage, progressBarWidth),
			renderProgressBar(sums.mem, totalMem, "Memory", memUsage, progressBarWidth),
		)
	}

	return neat.ContentBox(
		"CPU and Memory usage by Namespace",
		buf.String(),
		neat.HeadlineColor(bunt.DimGray),
	)
}

func renderTopContainers(topDetails *havener.TopDetails, x int) string {
	type entry struct {
		nodename  string
		namespace string
		pod       string
		container string
		cpu       int64
		mem       int64
	}

	topContainers, topContainersPerNode := func() ([]entry, map[string][]entry) {
		perNode := map[string][]entry{}
		for node := range topDetails.Nodes {
			perNode[node] = []entry{}
		}

		result := []entry{}
		for namespace, podMap := range topDetails.Containers {
			for pod, containerMap := range podMap {
				for container, usage := range containerMap {
					nodename := usage.Nodename

					e := entry{
						nodename:  nodename,
						namespace: namespace,
						pod:       pod,
						container: container,
						cpu:       usage.UsedCPU,
						mem:       usage.UsedMemory,
					}

					result = append(result, e)
					perNode[nodename] = append(perNode[nodename], e)
				}
			}
		}

		sort.Slice(result, func(i, j int) bool {
			return result[i].mem > result[j].mem
		})

		for i := range perNode {
			list := perNode[i]
			sort.Slice(list, func(i, j int) bool {
				return list[i].mem > list[j].mem
			})
		}

		return result, perNode
	}()

	topPodsInCluster := func() string {
		table := [][]string{
			{
				bunt.Sprintf("LightSteelBlue{*Namespace/Pod/Container*}"),
				bunt.Sprintf("LightSteelBlue{*Cores*}"),
				bunt.Sprintf("LightSteelBlue{*Memory*}"),
			},
		}

		x = func() int {
			if x < len(topContainers) {
				return x
			}

			return len(topContainers) - 1
		}()

		for _, entry := range topContainers[:x] {
			table = append(table, []string{
				renderContainerName(entry.namespace, entry.pod, entry.container),
				fmt.Sprintf("%.2f", float64(entry.cpu)/1000),
				humanReadableSize(entry.mem),
			})
		}

		out, err := neat.Table(table, neat.AlignRight(1, 2), neat.CustomSeparator("  "))
		if err != nil {
			return err.Error()
		}

		return out
	}()

	topPodsPerNode := func() string {
		table := [][]string{
			{
				bunt.Sprintf("LightSteelBlue{*Node*}"),
				bunt.Sprintf("LightSteelBlue{*Namespace/Pod/Container*}"),
				bunt.Sprintf("LightSteelBlue{*Cores*}"),
				bunt.Sprintf("LightSteelBlue{*Memory*}"),
			},
		}

		for _, node := range sortedNodeList(topDetails) {
			list := topContainersPerNode[node]
			j := func() int {
				endIdx := x / len(topContainersPerNode)
				if endIdx < len(topContainersPerNode) {
					return endIdx
				}

				return len(topContainersPerNode) - 1
			}()

			for i := 0; i < j; i++ {
				var nodename string
				if i == 0 {
					nodename = node
				}

				table = append(table, []string{
					nodename,
					renderContainerName(list[i].namespace, list[i].pod, list[i].container),
					fmt.Sprintf("%.2f", float64(list[i].cpu)/1000),
					humanReadableSize(list[i].mem),
				})
			}
		}

		out, err := neat.Table(table, neat.AlignRight(2, 3), neat.CustomSeparator("  "))
		if err != nil {
			return err.Error()
		}

		return out
	}()

	return sideBySide(
		neat.ContentBox(
			"Top Pods in Cluster",
			topPodsInCluster,
			neat.HeadlineColor(bunt.DimGray),
		),

		neat.ContentBox(
			"Top Pods per Node",
			topPodsPerNode,
			neat.HeadlineColor(bunt.DimGray),
		),
	)
}

func sortedNodeList(topDetails *havener.TopDetails) []string {
	type tmp struct {
		name  string
		value int64
	}

	list := []tmp{}
	for name, stats := range topDetails.Nodes {
		list = append(list, tmp{name, stats.UsedMemory})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].value > list[j].value
	})

	result := []string{}
	for _, entry := range list {
		result = append(result, entry.name)
	}

	return result
}

func fill(text string, length int) string {
	switch {
	case len(text) < length:
		return text + strings.Repeat(" ", length-len(text))

	default:
		return text
	}
}

func lines(text string) int {
	return len(strings.Split(text, "\n"))
}

func sideBySide(left string, right string) string {
	leftLines, rightLines := strings.Split(left, "\n"), strings.Split(right, "\n")

	tmp := [][]string{}
	max := func() int {
		if len(leftLines) > len(rightLines) {
			return len(leftLines)
		}

		return len(rightLines)
	}()

	for i := 0; i < max; i++ {
		var l string
		if i < len(leftLines) {
			l = leftLines[i]
		}

		var r string
		if i < len(rightLines) {
			r = rightLines[i]
		}

		tmp = append(tmp, []string{l, r})
	}

	out, err := neat.Table(tmp,
		neat.DesiredWidth(term.GetTerminalWidth()),
		neat.CustomSeparator("   "),
	)

	if err != nil {
		return err.Error()
	}

	return out
}

func renderContainerName(namespace string, pod string, container string) string {
	return text.FixedLength(
		bunt.Sprintf("%s/%s/%s",
			namespace,
			pod,
			container,
		),
		maxContainerNameLength,
	)
}

func renderProgressBar(value int64, max int64, caption string, text string, length int) string {
	const symbol = "■"

	if !strings.HasSuffix(caption, " ") {
		caption = caption + " "
	}

	if !strings.HasPrefix(text, " ") {
		text = " " + text
	}

	width := length - len(text) - len(caption)
	usage := float64(value) / float64(max)
	marks := int(usage * float64(width))

	var buf bytes.Buffer

	bunt.Fprintf(&buf, "LightSteelBlue{%s}", caption)
	for i := 0; i < width; i++ {
		if i < marks {
			switch bunt.UseColors() {
			case true:
				// Use smooth curved gradient:
				// http://fooplot.com/?lang=en#W3sidHlwZSI6MCwiZXEiOiIoMS1jb3MoeF4yKjMuMTQxNSkpLzIiLCJjb2xvciI6IiMwMDAwMDAifSx7InR5cGUiOjEwMDAsIndpbmRvdyI6WyIwIiwiMSIsIjAiLCIxIl19XQ--
				blendFactor := 0.5 * (1.0 - math.Cos(math.Pow(float64(i)/float64(length), 2)*math.Pi))
				buf.WriteString(
					bunt.Style(
						symbol,
						bunt.Foreground(bunt.LimeGreen.BlendLab(bunt.Red, blendFactor)),
					),
				)

			default:
				buf.WriteString(symbol)
			}

		} else {
			switch bunt.UseColors() {
			case true:
				buf.WriteString(
					bunt.Style(
						symbol,
						bunt.Foreground(bunt.DimGray),
					),
				)

			default:
				buf.WriteString(" ")
			}
		}
	}

	if len(text) > 0 {
		bunt.Fprintf(&buf, "Gray{%s}", text)
	}

	return buf.String()
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
