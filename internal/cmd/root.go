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
	"bytes"
	"fmt"
	"net/url"
	"os"
	"runtime/debug"
	"strings"

	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/neat"
	"github.com/gonvenience/term"
	"github.com/gonvenience/wrap"
)

type wrappedError interface {
	Error() string
	Unwrap() error
}

func title(w wrappedError) string {
	var parts = strings.SplitN(w.Error(), ":", 2)
	return parts[0]
}

func cause(w wrappedError) string {
	return w.Unwrap().Error()
}

var kubeConfig string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "havener",
	Short: "Convenience wrapper around some kubectl commands",
	Long: `Convenience wrapper around some kubectl commands.

Think of it as a swiss army knife for Kubernetes tasks. Possible use cases are
for example executing a command on multiple pods at the same time, or
retrieving usage details.

See the individual commands to get the complete overview.

`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	defer func() {
		const panicTitle = "Well, uhm, that is something we did not cover ..."
		if r := recover(); r != nil {
			switch r := r.(type) {
			case error:
				exitWithErrorAndIssue(panicTitle, r)

			default:
				exitWithErrorAndIssue(panicTitle, fmt.Errorf("%v", r))
			}
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		var headline, content string

		switch err := err.(type) {
		case wrap.ContextError:
			headline = bunt.Sprintf("*Error:* _%s_", err.Context())
			content = err.Cause().Error()

		case wrappedError:
			headline = title(err)
			content = cause(err)

		default:
			headline = "Error occurred"
			content = err.Error()
		}

		neat.Box(os.Stderr,
			headline, strings.NewReader(content),
			neat.HeadlineColor(bunt.Coral),
			neat.ContentColor(bunt.DimGray),
			neat.NoLineWrap(),
		)

		os.Exit(1)
	}
}

func init() {
	kubeConfigDefault, err := havener.KubeConfig()
	if err != nil {
		panic(err)
	}

	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().SortFlags = false

	rootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", kubeConfigDefault, "Kubernetes configuration")

	rootCmd.PersistentFlags().Int("terminal-width", -1, "disable autodetection and specify an explicit terminal width")
	rootCmd.PersistentFlags().Int("terminal-height", -1, "disable autodetection and specify an explicit terminal height")

	rootCmd.PersistentFlags().Bool("fatal", false, "fatal output - level 1")
	rootCmd.PersistentFlags().Bool("error", false, "error output - level 2")
	rootCmd.PersistentFlags().Bool("warn", false, "warn output - level 3")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output - level 4")
	rootCmd.PersistentFlags().Bool("debug", false, "debug output - level 5")
	rootCmd.PersistentFlags().Bool("trace", false, "trace output - level 6")

	// Bind environment variables to CLI flags
	_ = viper.BindPFlag("TERMINAL_WIDTH", rootCmd.PersistentFlags().Lookup("terminal-width"))
	_ = viper.BindPFlag("TERMINAL_HEIGHT", rootCmd.PersistentFlags().Lookup("terminal-height"))

	_ = viper.BindPFlag("fatal", rootCmd.PersistentFlags().Lookup("fatal"))
	_ = viper.BindPFlag("error", rootCmd.PersistentFlags().Lookup("error"))
	_ = viper.BindPFlag("warn", rootCmd.PersistentFlags().Lookup("warn"))
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindPFlag("trace", rootCmd.PersistentFlags().Lookup("trace"))

	term.FixedTerminalWidth, term.FixedTerminalHeight = viper.GetInt("TERMINAL_WIDTH"), viper.GetInt("TERMINAL_HEIGHT")

	// Issue https://github.com/homeport/havener/issues/26:
	// Enforce fixed terminal width override if executed inside a Garden container
	if term.FixedTerminalWidth < 0 && term.IsGardenContainer() {
		term.FixedTerminalHeight = 128
	}
}

// exitWithErrorAndIssue leaves the tool with the provided error message and a
// link that can be used to open a GitHub issue
func exitWithErrorAndIssue(msg string, err error) {
	neat.Box(os.Stderr,
		msg, strings.NewReader(err.Error()),
		neat.HeadlineColor(bunt.Coral),
		neat.ContentColor(bunt.DimGray),
	)

	var buf bytes.Buffer
	buf.WriteString(err.Error())
	buf.WriteString("\n\nStacktrace:\n```")
	buf.WriteString(string(debug.Stack()))
	buf.WriteString("```")

	bunt.Printf("\nIf you like to open an issue in GitHub:\nCornflowerBlue{~https://github.com/homeport/havener/issues/new?title=%s&body=%s~}\n\n",
		url.PathEscape("Report panic: "+err.Error()),
		url.PathEscape(buf.String()),
	)

	os.Exit(1)
}

// NewHvnrRootCmd returns the cobra base cmd
func NewHvnrRootCmd() *cobra.Command {
	return rootCmd
}
