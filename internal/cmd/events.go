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
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/gonvenience/bunt"
	"github.com/homeport/havener/pkg/havener"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var namespaceFilter string

type note struct {
	time      time.Time
	noteType  string
	namespace string
	resource  string
	reason    string
	message   string
}

// eventsCmd represents the top command
var eventsCmd = &cobra.Command{
	Use:           "events",
	Short:         "Show Kubernetes cluster events",
	Long:          `Show Kubernetes cluster events of all resources in all namespaces`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		hvnr, err := havener.NewHavener(havener.WithContext(cmd.Context()), havener.WithKubeConfigPath(kubeConfig))
		if err != nil {
			return fmt.Errorf("unable to get access to cluster: %w", err)
		}

		return retrieveClusterEvents(hvnr)
	},
}

func init() {
	rootCmd.AddCommand(eventsCmd)

	eventsCmd.PersistentFlags().StringVarP(&namespaceFilter, "namespace", "n", "", "Filter events for specific namespace")
}

func retrieveClusterEvents(hvnr havener.Havener) error {
	namespaces, err := hvnr.ListNamespaces()
	if err != nil {
		return fmt.Errorf("failed to get a list of namespaces: %w", err)
	}

	notes := make(chan note)

	// Start one Go routine per namespace to watch for events
	for i := range namespaces {
		namespace := namespaces[i]

		if namespaceFilter != "" && namespace != namespaceFilter {
			continue
		}

		watcher, err := hvnr.Client().CoreV1().Events(namespace).Watch(hvnr.Context(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to setup event watcher: %w", err)
		}

		go func() {
			for event := range watcher.ResultChan() {
				switch event.Type {
				case watch.Added, watch.Modified:
					switch data := event.Object.(type) {
					case *corev1.Event:
						resourceName := data.Name
						if strings.Contains(resourceName, ".") {
							parts := strings.Split(resourceName, ".")
							resourceName = strings.Join(parts[:len(parts)-1], ".")
						}

						notes <- note{
							namespace: namespace,
							time:      data.FirstTimestamp.Time,
							noteType:  data.Type,
							resource:  resourceName,
							reason:    data.Reason,
							message:   strings.TrimSuffix(data.Message, "\n"),
						}
					}
				}
			}
		}()
	}

	// Show the generated notes until the user stops the application
	for note := range notes {
		var noteColor = bunt.LightSteelBlue
		if note.noteType == "Warning" {
			noteColor = bunt.FireBrick
		}

		bunt.Printf("DimGray{%s} %-7s _%s_/%s  *%s*  AntiqueWhite{%s}\n",
			note.time,
			bunt.Style(note.noteType, bunt.Foreground(noteColor)),
			note.namespace,
			note.resource,
			note.reason,
			note.message,
		)
	}
	return nil
}
