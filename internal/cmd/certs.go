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
	"github.com/gonvenience/neat"
	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
)

// certCmd represents the cert command
var certsCmd = &cobra.Command{
	Use:   "certs",
	Short: "Check certificates",
	Long:  `Verify certificates from all secrets in all namespaces`,
	RunE: func(cmd *cobra.Command, args []string) error {
		details, err := havener.VerifyCertExpirations()
		if err != nil {
			return err
		}

		return printBoxWithTable(
			"Overview of certificates found in Cluster secrets",
			[]string{"namespace", "secret", "name/key", "status"},
			details,
			neat.VertialBarSeparator(),
		)
	},
}

func init() {
	rootCmd.AddCommand(certsCmd)
}
