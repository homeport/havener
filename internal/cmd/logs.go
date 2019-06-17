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
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/gonvenience/wait"

	"github.com/homeport/havener/pkg/havener"
)

var includeConfigFiles bool
var downloadLocation string
var totalDownloadTimeout int

// logsCmd represents the top command
var logsCmd = &cobra.Command{
	Use:           "logs",
	Short:         "Retrieve log files from pods",
	Long:          `Retrieve log files from pods`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return retrieveClusterLogs()
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.PersistentFlags().BoolVar(&includeConfigFiles, "config-files", false, "include configuration files in download package")
	logsCmd.PersistentFlags().StringVar(&downloadLocation, "target", os.TempDir(), "desired target download location for retrieved files")
	logsCmd.PersistentFlags().IntVar(&totalDownloadTimeout, "timeout", 5*60, "allowed time in seconds before the download is aborted")
}

func retrieveClusterLogs() error {
	clientSet, restconfig, err := havener.OutOfClusterAuthentication("")
	if err != nil {
		return &ErrorWithMsg{"unable to get access to cluster", err}
	}

	var commonText string
	if includeConfigFiles {
		commonText = "log and configuration files"
	} else {
		commonText = "log files"
	}

	timeout := time.Duration(totalDownloadTimeout) * time.Second

	pi := wait.NewProgressIndicator("Downloading " + commonText + " ...")
	pi.SetTimeout(timeout)
	pi.Start()

	resultChan := make(chan error, 1)
	go func() {
		resultChan <- havener.RetrieveLogs(clientSet, restconfig, downloadLocation, includeConfigFiles)
	}()

	select {
	case err := <-resultChan:
		if err != nil {
			pi.Stop()
			return &ErrorWithMsg{"unable to retrieve logs from pods", err}
		}

	case <-time.After(timeout):
		pi.Stop()
		return &ErrorWithMsg{"unable to retrieve logs from pods", fmt.Errorf("download did not finish within configured timeout")}
	}

	pi.Done("Done downloading " + commonText + ": " + filepath.Join(downloadLocation, havener.LogDirName))
	return nil
}
