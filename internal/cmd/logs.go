// Copyright © 2021 The Homeport Team
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
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/gonvenience/wait"
	"github.com/gonvenience/wrap"

	"github.com/homeport/havener/pkg/havener"
)

var (
	excludeConfigFiles   bool
	parallelDownloads    int
	totalDownloadTimeout int
	downloadLocation     string
)

// logsCmd represents the top command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Retrieve log files from all pods",
	Long: `Loops over all pods and all namespaces to download log and configuration
files from some well-known hard-coded locations to a local directory. Use this
to quickly scan through multiple files from multiple locations in case you have
to debug an issue where it is not clear yet where to look.

The download includes all deployment YAMLs of the pods and the describe output.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return retrieveClusterLogs()
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.PersistentFlags().BoolVar(&excludeConfigFiles, "no-config-files", false, "exclude configuration files in download package")
	logsCmd.PersistentFlags().StringVar(&downloadLocation, "target", os.TempDir(), "desired target download location for retrieved files")
	logsCmd.PersistentFlags().IntVar(&totalDownloadTimeout, "timeout", 5*60, "allowed time in seconds before the download is aborted")
	logsCmd.PersistentFlags().IntVar(&parallelDownloads, "parallel", 64, "number of parallel download jobs")
}

func retrieveClusterLogs() error {
	hvnr, err := havener.NewHavener()
	if err != nil {
		return wrap.Error(err, "unable to get access to cluster")
	}

	var commonText string
	if excludeConfigFiles {
		commonText = "log files"
	} else {
		commonText = "log and configuration files"
	}

	timeout := time.Duration(totalDownloadTimeout) * time.Second

	pi := wait.NewProgressIndicator("Downloading %s to _%s_ ...", commonText, downloadLocation)
	pi.SetTimeout(timeout)
	setCurrentProgressIndicator(pi)
	defer setCurrentProgressIndicator(nil)
	pi.Start()

	resultChan := make(chan error, 1)
	go func() {
		resultChan <- hvnr.RetrieveLogs(
			parallelDownloads,
			downloadLocation,
			!excludeConfigFiles,
		)
	}()

	select {
	case err := <-resultChan:
		if err != nil {
			pi.Stop()
			return wrap.Error(err, "unable to retrieve logs from pods")
		}

	case <-time.After(timeout):
		pi.Stop()
		return wrap.Error(
			fmt.Errorf("download did not finish within configured timeout"),
			"unable to retrieve logs from pods",
		)
	}

	pi.Done("Finished downloading %s to %s",
		commonText,
		filepath.Join(downloadLocation, havener.LogDirName),
	)

	return nil
}
