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
	"runtime"

	"github.com/homeport/gonvenience/pkg/v1/bunt"
	"github.com/spf13/cobra"
)

var havenerVersion string
var kubeVersion string
var helmVersion string

// versionCmd represents the top command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Shows the version",
	Long:  `Shows version details`,
	Run: func(cmd *cobra.Command, args []string) {
		for _, ptr := range []*string{&havenerVersion, &kubeVersion, &helmVersion} {
			if len(*ptr) == 0 {
				*ptr = "develop"
			}
		}

		bunt.Printf("*havener* version MintCream{_%s_}, *Go* version MintCream{_%s %s/%s_}, *Kubernetes API* version MintCream{_%s_}, *Helm* version MintCream{_%s_}\n",
			havenerVersion,
			runtime.Version(),
			runtime.GOOS,
			runtime.GOARCH,
			kubeVersion,
			helmVersion,
		)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
