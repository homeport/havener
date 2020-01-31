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
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/text"
	"github.com/gonvenience/wait"
	"github.com/gonvenience/wrap"
	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
)

// purgeCmd represents the purge command
var purgeCmd = &cobra.Command{
	Use:   "purge <helm-release> [<helm-release>] [...]",
	Short: "Deletes Helm Releases",
	Long: `Deletes all specified Helm Releases as quickly as possible.

It first deletes all stateful sets and deployments at the same time. Afterwards
the deletion of the namespace associated with the Helm Release will be triggered
in parallel to the deletion of the Helm Release itself.

If multiple Helm Releases are specified, then they will deleted concurrently.
`,

	SilenceUsage:  true,
	SilenceErrors: true,

	Args: func() cobra.PositionalArgs {
		return func(cmd *cobra.Command, args []string) error {
			hvnr, err := havener.NewHavener()
			if err != nil {
				return err
			}

			if len(args) < 1 {
				var message string = "At least one Helm Release has to be specified."
				if list, err := hvnr.ListHelmReleases(); err == nil {
					names := make([]string, len(list))
					for i, entry := range list {
						names[i] = entry.Name
					}

					message = fmt.Sprintf("%s\n\nList of Helm Releases:\n%s",
						message,
						strings.Join(names, "\n"),
					)
				}

				return wrap.Errorf(errors.New(message), "missing argument")
			}

			return nil
		}
	}(),

	RunE: func(cmd *cobra.Command, args []string) error {
		hvnr, err := havener.NewHavener()
		if err != nil {
			return err
		}

		if err := PurgeHelmReleases(hvnr, args...); err != nil {
			return wrap.Error(err, "failed to purge helm releases")
		}

		return nil
	},
}

// PurgeHelmReleases delete releases via helm
func PurgeHelmReleases(hvnr havener.Havener, helmReleaseNames ...string) error {
	list, err := hvnr.ListHelmReleases()
	if err != nil {
		return err
	}

	isExistingHelmRelease := func(list []havener.HelmRelease, name string) bool {
		for _, helmRelease := range list {
			if helmRelease.Name == name {
				return true
			}
		}

		return false
	}

	// Go through the list of actual helm releases to filter our non-existing releases.
	toBeDeleted := []string{}
	for _, helmReleaseName := range helmReleaseNames {
		if isExistingHelmRelease(list, helmReleaseName) {
			toBeDeleted = append(toBeDeleted, helmReleaseName)
		}
	}

	// Ask for confirmation about the releases to be deleted.
	if ok := PromptUser("Are you sure you want to delete the Helm Releases " + strings.Join(toBeDeleted, ", ") + "? (yes/no): "); !ok {
		return nil
	}

	outputMsg := bunt.Sprintf("*Deleting %s*: %s",
		text.Plural(len(toBeDeleted), "Helm Release"),
		strings.Join(toBeDeleted, ", "),
	)

	// Show a wait indicator ...
	pi := wait.NewProgressIndicator(outputMsg)
	setCurrentProgressIndicator(pi)
	pi.Start()
	defer setCurrentProgressIndicator(nil)
	defer pi.Stop()

	var wg sync.WaitGroup
	wg.Add(len(toBeDeleted))
	errors := make(chan error, len(toBeDeleted))

	// Start to purge the helm releaes in parallel
	for _, name := range toBeDeleted {
		releaseMetaData, err := hvnr.GetReleaseByName(name)
		if err != nil {
			return err
		}

		go func(helmRelease string) {
			errors <- hvnr.PurgeHelmRelease(releaseMetaData, helmRelease)
			wg.Done()
		}(name)
	}

	wg.Wait()
	close(errors)

	return combineErrorsFromChannel("foobar", errors)
}

func init() {
	rootCmd.AddCommand(purgeCmd)

	purgeCmd.PersistentFlags().BoolVar(&NoUserPrompt, "non-interactive", false, "delete without asking for confirmation")
}
