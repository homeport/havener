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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/homeport/gonvenience/pkg/v1/wait"
	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm"
)

/* TODO Currently, purge will ignore all non-existing helm releases that were
   provided by the user. Think about making the behaviour configurable: For
   example by introducing a flag like `--ignore-non-existent` or similar. */

/* TODO Should we make getConfiguredHelmClient a havener package function? */

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
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		havener.VerboseMessage("Accessing cluster...")

		client, _, err := havener.OutOfClusterAuthentication("")
		if err != nil {
			return &ErrorWithMsg{"unable to get access to cluster", err}
		}

		if err := PurgeHelmReleases(client, args...); err != nil {
			return &ErrorWithMsg{"failed to purge helm releases", err}
		}
		return nil
	},
}

func getConfiguredHelmClient() (*helm.Client, error) {

	havener.VerboseMessage("Reading kube config file...")

	cfg := viper.GetString("kubeconfig")

	havener.VerboseMessage("Getting helm client...")

	helmClient, err := havener.GetHelmClient(cfg)
	if err != nil {
		return nil, err
	}

	return helmClient, nil
}

// PurgeHelmReleases delete releases via helm
func PurgeHelmReleases(kubeClient kubernetes.Interface, helmReleaseNames ...string) error {
	// Get a struct with existing releases, and access it by name
	releasesList := havener.HelmReleases{}

	stdOutput, err := havener.RunHelmBinary("list", "-a", "--output", "json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(stdOutput, &releasesList)
	if err != nil {
		return err
	}

	// Go through the list of actual helm releases to filter our non-existing releases.
	toBeDeleted := []string{}
	for _, helmReleaseName := range helmReleaseNames {
		if havener.ReleaseExist(releasesList, helmReleaseName) {
			toBeDeleted = append(toBeDeleted, helmReleaseName)
		}
	}

	// Ask for confirmation about the releases to be deleted.
	if ok := PromptUser("Are you sure you want to delete the Helm Releases " + strings.Join(toBeDeleted, ", ") + "? (yes/no): "); !ok {
		return nil
	}

	// Show a wait indicator ...
	pi := wait.NewProgressIndicator(fmt.Sprintf("Deleting Helm Releases: " + strings.Join(toBeDeleted, ",")))
	pi.Start()
	defer pi.Stop()

	// Start to purge the helm releaes in parallel
	errors := make(chan error, len(toBeDeleted))
	for _, name := range toBeDeleted {
		releaseMetaData, err := havener.GetReleaseByName(name)
		if err != nil {
			return err
		}
		go func(helmRelease string) {
			errors <- havener.PurgeHelmRelease(kubeClient, releaseMetaData, helmRelease)
		}(name)
	}

	// Wait for the go-routines to finish before leaving this function
	for i := 0; i < len(toBeDeleted); i++ {
		if err := <-errors; err != nil {
			return err
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(purgeCmd)

	purgeCmd.PersistentFlags().BoolVar(&NoUserPrompt, "non-interactive", false, "delete without asking for confirmation")
}
