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
	"strings"
	"time"

	"github.com/homeport/gonvenience/pkg/v1/term"
	"github.com/homeport/havener/internal/hvnr"
	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
)

var (
	cycles   = -1
	interval = 2
)

// topCmd represents the top command
var topCmd = &cobra.Command{
	Use:           "top",
	Short:         "Shows CPU and Memory usage",
	Long:          `Shows more detailed CPU and Memory usage details`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return retrieveClusterStats()
	},
}

func init() {
	rootCmd.AddCommand(topCmd)

	topCmd.PersistentFlags().IntVarP(&cycles, "cycles", "c", -1, "number of cycles to run, negative numbers means infinite cycles")
	topCmd.PersistentFlags().IntVarP(&interval, "interval", "i", 2, "interval between measurements in seconds")

	topCmd.Flags().SortFlags = false
	topCmd.PersistentFlags().SortFlags = false
}

func retrieveClusterStats() error {
	// Fall back to default interval if unsupported interval was specified
	if interval <= 0 {
		interval = 2
	}

	// Restrict the output to one single measurement in case of a dumb terminal or used inside Concourse
	if term.IsDumbTerminal() || term.IsGardenContainer() {
		cycles = 1
	}

	clientSet, _, err := havener.OutOfClusterAuthentication("")
	if err != nil {
		return &ErrorWithMsg{"unable to get access to cluster", err}
	}

	term.HideCursor()
	defer term.ShowCursor()

	iterations := 0
	for range time.Tick(time.Duration(interval) * time.Second) {
		// TODO Get stats for nodes and pods at the same time
		nodeStats, err := hvnr.CompileNodeStats(clientSet)
		if err != nil {
			return &ErrorWithMsg{"failed to compile node usage stats", err}
		}

		podStats, err := hvnr.CompilePodStats(clientSet)
		if err != nil {
			return &ErrorWithMsg{"failed to compile pod usage stats", err}
		}

		podLineLimit := term.GetTerminalHeight() - len(strings.Split(nodeStats, "\n")) - 1
		if lines := strings.Split(podStats, "\n"); len(lines) > podLineLimit {
			podStats = strings.Join(lines[:podLineLimit], "\n")
		}

		fmt.Print("\x1b[H")
		fmt.Print("\x1b[2J")
		fmt.Print(nodeStats)
		fmt.Print(podStats)

		if iterations++; cycles > 0 && iterations > cycles {
			break
		}
	}
	return nil
}
