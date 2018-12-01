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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/homeport/gonvenience/pkg/v1/term"
	"github.com/mitchellh/go-homedir"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "havener",
	Short: "Convenience tool to handle tasks around Containerized CF workloads on a Kubernetes cluster",
	Long: `A convenience tool to handle tasks around Containerized CF workloads on a Kubernetes cluster, for example:
- Deploy a new series of Helm Charts
- Remove all Helm Releases
- Retrieve log and configuration files from all pods

See the individual commands to get the complete overview.
`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	home, err := homedir.Dir()
	if err != nil {
		exitWithError("unable to get home directory", err)
	}

	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().SortFlags = false

	rootCmd.PersistentFlags().String("kubeconfig", filepath.Join(home, ".kube", "config"), "kubeconfig file (default is $HOME/.kube/config)")
	rootCmd.PersistentFlags().Int("terminal-width", -1, "disable autodetection and specify an explicit terminal width")
	rootCmd.PersistentFlags().Int("terminal-height", -1, "disable autodetection and specify an explicit terminal height")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

	// Bind environment variables to CLI flags
	viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("TERMINAL_WIDTH", rootCmd.PersistentFlags().Lookup("terminal-width"))
	viper.BindPFlag("TERMINAL_HEIGHT", rootCmd.PersistentFlags().Lookup("terminal-height"))

	term.FixedTerminalWidth, term.FixedTerminalHeight = viper.GetInt("TERMINAL_WIDTH"), viper.GetInt("TERMINAL_HEIGHT")

	// Issue https://github.com/homeport/havener/issues/26:
	// Enforce fixed terminal width override if executed inside a Garden container
	if term.FixedTerminalWidth < 0 && term.IsGardenContainer() {
		term.FixedTerminalHeight = 128
	}
}
