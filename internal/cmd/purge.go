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
	"io/ioutil"

	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/helm/pkg/helm"
)

// purgeCmd represents the purge command
var purgeCmd = &cobra.Command{
	Use:   "purge <helm-release> [<helm-release>] [...]",
	Args:  cobra.MinimumNArgs(1),
	Short: "Deletes Helm Releases",
	Long: `Deletes all specified Helm Releases as quickly as possible.

It first deletes all stateful sets and deployments at the same time. Afterwards
the deletion of the namespace associated with the Helm Release will be triggered
in parallel to the deletion of the Helm Release itself.

If multiple Helm Releases are specified, then they will deleted concurrently.
`,
	Run: func(cmd *cobra.Command, args []string) {
		client, _, err := havener.OutOfClusterAuthentication()
		if err != nil {
			havener.ExitWithError("unable to get access to cluster", err)
		}

		if err := havener.PurgeHelmReleases(client, getConfiguredHelmClient(), args...); err != nil {
			havener.ExitWithError("failed to purge helm releases", err)
		}
	},
}

// TODO Make this a hevener package function?
func getConfiguredHelmClient() *helm.Client {
	cfg, err := ioutil.ReadFile(viper.GetString("kubeconfig"))
	if err != nil {
		havener.ExitWithError("unable to read the kube config file", err)
	}

	helmClient, err := havener.GetHelmClient(cfg)
	if err != nil {
		havener.ExitWithError("failed to get helm client", err)
	}

	return helmClient
}

func init() {
	rootCmd.AddCommand(purgeCmd)

	purgeCmd.PersistentFlags().BoolVar(&havener.NoUserPrompt, "non-interactive", false, "delete without asking for confirmation")
}
