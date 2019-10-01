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
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/neat"
	"github.com/gonvenience/term"
	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

var watchCmdSettings struct {
	namespaces []string
}

// watchCmd represents the top command
var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch status of all pods in all namespaces",
	Long:  `Continuesly creates a list of all pods in all namespaces.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		hvnr, err := havener.NewHavener()
		if err != nil {
			return err
		}

		term.HideCursor()
		defer term.ShowCursor()

		// Make sure to start with a print
		if err := printWatchList(hvnr); err != nil {
			return err
		}

		var ticker = time.NewTicker(time.Duration(interval) * time.Second)
		for {
			select {
			case <-ticker.C:
				if err := printWatchList(hvnr); err != nil {
					return err
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.PersistentFlags().StringSliceVarP(&watchCmdSettings.namespaces, "namespace", "n", []string{}, "comma separated list of namespaces to filter (default is to use all namespaces")
}

func printWatchList(hvnr havener.Havener) error {
	pods, err := hvnr.ListPods(watchCmdSettings.namespaces...)
	if err != nil {
		return err
	}

	sort.Slice(pods, func(i, j int) bool {
		// sort by system namespace (user namespaces before system namespaces)
		if categoryI, categoryJ := humanReadableNamespaceCategory(*pods[i]), humanReadableNamespaceCategory(*pods[j]); categoryI != categoryJ {
			return categoryI > categoryJ
		}

		// sort by namespace
		if pods[i].Namespace != pods[j].Namespace {
			return pods[i].Namespace < pods[j].Namespace
		}

		// sort by finish status (active before done)
		if statusI, statusJ := humanReadablePodRunningStatus(*pods[i]), humanReadablePodRunningStatus(*pods[j]); statusI != statusJ {
			return statusI < statusJ
		}

		// sort by name
		return pods[i].Name < pods[j].Name
	})

	var (
		counter  = 0
		maxLines = term.GetTerminalHeight() - 5
		table    = [][]string{}
	)

	for _, pod := range pods {
		status := humanReadablePodStatus(*pod)

		age := humanReadableDuration(
			time.Now().Sub(
				pod.CreationTimestamp.Time,
			),
		)

		readyContainer, totalContainer := func() (int, int) {
			var counter int
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.Ready {
					counter++
				}
			}

			return counter, len(pod.Status.ContainerStatuses)
		}()

		ready := fmt.Sprintf("%d/%d", readyContainer, totalContainer)

		styleOptions := []bunt.StyleOption{}
		switch {
		case status == "Succeeded":
			styleOptions = append(styleOptions, bunt.Foreground(bunt.DimGray))

		case status == "Terminating":
			styleOptions = append(styleOptions, bunt.Foreground(bunt.PeachPuff))

		case status == "CrashLoopBackOff":
			styleOptions = append(styleOptions, bunt.Foreground(bunt.LightCoral))

		case status == "PodInitializing":
			styleOptions = append(styleOptions, bunt.Foreground(bunt.LightCyan))

		case status == "Pending":
			styleOptions = append(styleOptions, bunt.Foreground(bunt.Bisque))

		case readyContainer != totalContainer:
			styleOptions = append(styleOptions, bunt.Foreground(bunt.Gold))

		case humanReadableNamespaceCategory(*pod) == "system namespace":
			styleOptions = append(styleOptions, bunt.Foreground(bunt.LightSlateGray), bunt.Italic())
		}

		table = append(table, []string{
			bunt.Style(pod.Namespace, styleOptions...),
			bunt.Style(pod.Name, styleOptions...),
			bunt.Style(ready, styleOptions...),
			bunt.Style(status, styleOptions...),
			bunt.Style(age, styleOptions...),
		})

		if counter++; counter > maxLines {
			table = append(table, []string{"[...]", "", "", "", ""})
			break
		}
	}

	out, err := renderBoxWithTable(
		bunt.Sprintf("Pods running in cluster _%s_", hvnr.ClusterName()),
		[]string{"Namespace", "Pod", "Ready", "Status", "Age"},
		table,
		neat.CustomSeparator("  "),
	)

	if err != nil {
		return err
	}

	print("\x1b[H", "\x1b[2J", out)
	return nil
}

func humanReadableNamespaceCategory(pod corev1.Pod) string {
	switch {
	case strings.HasSuffix(pod.Namespace, "-system"):
		return "system namespace"

	default:
		return "user namespace"
	}
}

func humanReadablePodRunningStatus(pod corev1.Pod) string {
	if humanReadablePodStatus(pod) == "Succeeded" {
		return "Done"
	}

	return "Active"
}

func humanReadablePodStatus(pod corev1.Pod) string {
	if pod.DeletionTimestamp != nil {
		return "Terminating"
	}

	switch pod.Status.Phase {
	case corev1.PodPending:
		for _, containerStatus := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
			if containerStatus.State.Waiting != nil {
				if len(containerStatus.State.Waiting.Reason) != 0 {
					return containerStatus.State.Waiting.Reason
				}
			}
		}
	}

	return string(pod.Status.Phase)
}

func humanReadableDuration(duration time.Duration) string {
	if duration < time.Second {
		return "less than a second"
	}

	var (
		seconds = int(duration.Seconds())
		minutes = 0
		hours   = 0
		days    = 0
	)

	if seconds >= 60 {
		minutes = seconds / 60
		seconds = seconds % 60

		if minutes >= 60 {
			hours = minutes / 60
			minutes = minutes % 60

			if hours >= 24 {
				days = hours / 24
				hours = hours % 24
			}
		}
	}

	parts := []string{}

	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d d", days))
	}

	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d h", hours))
	}

	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d min", minutes))
	}

	if seconds > 0 {
		parts = append(parts, fmt.Sprintf("%d sec", seconds))
	}

	switch len(parts) {
	case 1:
		return parts[0]

	default:
		return strings.Join(parts[:2], " ")
	}
}
