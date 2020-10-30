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
	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/spf13/cobra"
)

var topCmdSettings struct {
	cycles   int
	interval int
}

// topCmd represents the top command
var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Shows CPU and Memory usage",
	Long: `Shows more detailed CPU and Memory usage details

The top command shows Load, CPU, and Memory usage details for all nodes.

Based on the pod usage, aggregated usage details per namespace are generated
and displayed to show CPU and Memory usage.

Furthermore, the list of top pod consumers is displayed, both for the whole
cluster as well as a list per node.
`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Fall back to default interval if unsupported interval was specified
		if topCmdSettings.interval <= 0 {
			topCmdSettings.interval = 4
		}

		// Restrict the output to one single measurement in case of a dumb
		// terminal or used inside Concourse
		if term.IsDumbTerminal() || term.IsGardenContainer() {
			topCmdSettings.cycles = 1
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

			nodeDetails := RenderNodeDetails(top)
			namespaceDetails := renderNamespaceDetails(top)
			availableLines := term.GetTerminalHeight() - lines(nodeDetails) - lines(namespaceDetails)
			topContainers := renderTopContainers(top, max(0, availableLines-1))

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

		var ticker = time.NewTicker(time.Duration(topCmdSettings.interval) * time.Second)
		var timeout = make(<-chan time.Time)
		if topCmdSettings.cycles > 0 {
			timeout = time.After(time.Duration(topCmdSettings.interval*topCmdSettings.cycles) * time.Second)
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

	topCmd.PersistentFlags().IntVarP(&topCmdSettings.cycles, "cycles", "c", -1, "number of cycles to run, negative numbers means infinite cycles")
	topCmd.PersistentFlags().IntVarP(&topCmdSettings.interval, "interval", "i", 4, "interval between measurements in seconds")

	topCmd.Flags().SortFlags = false
	topCmd.PersistentFlags().SortFlags = false
}

// RenderNodeDetails renders a box with usage details per node
func RenderNodeDetails(topDetails *havener.TopDetails) string {
	maxNodeNameLength := func() int {
		var maxLength int
		for name := range topDetails.Nodes {
			if length := len(name); length > maxLength {
				maxLength = length
			}
		}

		return maxLength
	}()

	var loadAvgStrings = map[string]string{}
	for name, node := range topDetails.Nodes {
		loadAvgStrings[name] = renderLoadAvg("Load", node)
	}

	maxLoadAvgStringLength := func() int {
		var maxLength int
		for _, value := range loadAvgStrings {
			if length := bunt.PlainTextLength(value); length > maxLength {
				maxLength = length
			}
		}

		return maxLength
	}()

	// Based on the terminal width, subtract 2 for the prefix, 6 for the spaces
	// and the respective longest name and load string
	progressBarWidth := (term.GetTerminalWidth() - 2 - maxNodeNameLength - maxLoadAvgStringLength - 6) / 2

	var buf bytes.Buffer
	for _, name := range sortedNodeList(topDetails) {
		stats := topDetails.Nodes[name]

		cpuUsage := fmt.Sprintf("%.1f%%", float64(stats.UsedCPU)/float64(stats.TotalCPU)*100.0)
		memUsage := fmt.Sprintf("%s/%s", humanReadableSize(stats.UsedMemory), humanReadableSize(stats.TotalMemory))
		bunt.Fprintf(&buf, "%s  %s  %s  %s\n",
			fill(name, maxNodeNameLength),
			fill(loadAvgStrings[name], maxLoadAvgStringLength),
			renderProgressBar(stats.UsedCPU, stats.TotalCPU, "CPU", cpuUsage, progressBarWidth),
			renderProgressBar(stats.UsedMemory, stats.TotalMemory, "Memory", memUsage, progressBarWidth),
		)
	}

	return neat.ContentBox(
		"CPU and Memory usage by Node",
		buf.String(),
		neat.HeadlineColor(bunt.SkyBlue),
		neat.NoLineWrap(),
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
		neat.HeadlineColor(bunt.SkyBlue),
		neat.NoLineWrap(),
	)
}

func renderTopContainers(topDetails *havener.TopDetails, maxNumberOfLines int) string {
	type entry struct {
		nodename  string
		namespace string
		pod       string
		container string
		cpu       int64
		mem       int64
	}

	// In order to keep the output as compact as possible, set the maximum size
	// for the container display name to fraction of the display width
	maxContainerNameLength := term.GetTerminalWidth() / 5

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
		table := [][]string{}
		maxNumberOfLines = func() int {
			if maxNumberOfLines < len(topContainers) {
				return maxNumberOfLines
			}

			return len(topContainers) - 1
		}()

		for _, entry := range topContainers[:maxNumberOfLines] {
			table = append(table, []string{
				renderContainerName(entry.namespace, entry.pod, entry.container, maxContainerNameLength),
				fmt.Sprintf("%.2f", float64(entry.cpu)/1000),
				humanReadableSize(entry.mem),
			})
		}

		out, err := renderBoxWithTable(
			"Top Pods in Cluster",
			[]string{"Namespace/Pod/Container", "Cores", "Memory"},
			table,
			neat.AlignRight(1, 2),
			neat.CustomSeparator("  "),
		)

		if err != nil {
			return err.Error()
		}

		return out
	}()

	topPodsPerNode := func() string {
		table := [][]string{}
		for _, node := range sortedNodeList(topDetails) {
			list := topContainersPerNode[node]
			maxInnerLoop := func() int {
				if endIdx := maxNumberOfLines / len(topContainersPerNode); endIdx < len(list) {
					return endIdx
				}

				return len(list) - 1
			}()

			for i := 0; i < maxInnerLoop; i++ {
				var nodename string
				if i == 0 {
					nodename = node
				}

				table = append(table, []string{
					nodename,
					renderContainerName(list[i].namespace, list[i].pod, list[i].container, maxContainerNameLength),
					fmt.Sprintf("%.2f", float64(list[i].cpu)/1000),
					humanReadableSize(list[i].mem),
				})
			}
		}

		out, err := renderBoxWithTable(
			"Top Pods per Node",
			[]string{"Node", "Namespace/Pod/Container", "Cores", "Memory"},
			table,
			neat.AlignRight(2, 3),
			neat.CustomSeparator("  "),
		)

		if err != nil {
			return err.Error()
		}

		return out
	}()

	return sideBySide(topPodsInCluster, topPodsPerNode)
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
	var textLength = bunt.PlainTextLength(text)
	switch {
	case textLength < length:
		return text + strings.Repeat(" ", length-textLength)

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

	out, err := neat.Table(
		tmp,
		neat.DesiredWidth(term.GetTerminalWidth()),
		neat.CustomSeparator("   "),
		neat.OmitLinefeedAtTableEnd(),
	)

	if err != nil {
		return err.Error()
	}

	return out
}

func renderContainerName(namespace string, pod string, container string, maxContainerNameLength int) string {
	return text.FixedLength(
		bunt.Sprintf("%s/%s/%s",
			namespace,
			pod,
			container,
		),
		maxContainerNameLength,
		bunt.Sprint(" DimGray{[...]}"),
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

	bunt.Fprintf(&buf, "*%s*", caption)
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
						bunt.Foreground(bunt.DimGray.BlendLab(bunt.Black, 0.5)),
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

func renderLoadAvg(caption string, stats havener.NodeDetails) string {
	if len(stats.LoadAvg) == 0 {
		return bunt.Sprintf("*%s* DimGray{_(no data)_}", caption)
	}

	colorForLoad := func(value float64, max float64) colorful.Color {
		switch {
		case value > max:
			return bunt.Red

		default:
			return bunt.Gray.BlendLab(
				bunt.Red,
				0.5*(1.0-math.Cos(math.Pow(value/max, 2)*math.Pi)),
			)
		}
	}

	cores := float64(stats.TotalCPU / 1000)
	return bunt.Sprintf("*%s* %s %s %s",
		caption,
		bunt.Style(fmt.Sprintf("%.1f", stats.LoadAvg[0]), bunt.Foreground(colorForLoad(stats.LoadAvg[0], cores))),
		bunt.Style(fmt.Sprintf("%.1f", stats.LoadAvg[1]), bunt.Foreground(colorForLoad(stats.LoadAvg[1], cores))),
		bunt.Style(fmt.Sprintf("%.1f", stats.LoadAvg[2]), bunt.Foreground(colorForLoad(stats.LoadAvg[2], cores))),
	)
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

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}
