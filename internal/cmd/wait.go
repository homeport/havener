// Copyright © 2020 The Havener
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
	"strings"
	"time"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/text"
	"github.com/gonvenience/wait"
	"github.com/gonvenience/wrap"
	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var waitCmdSettings struct {
	quiet        bool
	namespace    string
	podStartWith string
	interval     time.Duration
	timeout      time.Duration
}

// waitCmd represents the top command
var waitCmd = &cobra.Command{
	Use:   "wait",
	Short: "Wait for pods to become ready",
	Long: `
Command that blocks and waits until the specified condition is reached. It will
constantly check the given namespace using the provided interval to check
whether all pods that match the configured prefix to become ready.`,

	SilenceUsage:  true,
	SilenceErrors: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		switch {
		case len(waitCmdSettings.podStartWith) == 0:
			return wrap.Errorf(
				bunt.Errorf("There was no pod name prefix defined to watch for.\n\n%s", cmd.UsageString()),
				"Mandatory condition is missing",
			)
		}

		hvnr, err := havener.NewHavener()
		if err != nil {
			return wrap.Error(err, "unable to get access to cluster")
		}

		var (
			pi *wait.ProgressIndicator

			waitString = func(pods ...corev1.Pod) string {
				switch len(pods) {
				case 0:
					return bunt.Sprintf("Waiting for pods starting with *%s* to become ready in namespace _%s_ ...",
						waitCmdSettings.podStartWith,
						waitCmdSettings.namespace,
					)

				default:
					names := make([]string, len(pods))
					for i, pod := range pods {
						names[i] = bunt.Sprintf("*%s*", pod.Name)
					}

					return fmt.Sprintf("Waiting for %s in namespace _%s_: %s",
						text.Plural(len(pods), "unready pod"),
						waitCmdSettings.namespace,
						text.List(names),
					)
				}
			}

			listUnreadyPods = func() ([]corev1.Pod, error) {
				list, err := hvnr.Client().CoreV1().Pods(waitCmdSettings.namespace).List(metav1.ListOptions{})
				if err != nil {
					return nil, wrap.Errorf(err, "failed to get a list of pods in namespace %s", waitCmdSettings.namespace)
				}

				var unreadyPods = []corev1.Pod{}
				for i := range list.Items {
					pod := list.Items[i]

					if !strings.HasPrefix(pod.Name, waitCmdSettings.podStartWith) {
						continue
					}

					var ready int
					for _, containerStatus := range pod.Status.ContainerStatuses {
						if containerStatus.Ready {
							ready++
						}
					}

					if ready == len(pod.Status.ContainerStatuses) {
						continue
					}

					unreadyPods = append(unreadyPods, pod)
				}

				return unreadyPods, nil
			}

			checkFunc = func() (bool, error) {
				list, err := listUnreadyPods()
				if err != nil {
					return false, err
				}

				if len(list) == 0 {
					return true, nil
				}

				if pi != nil {
					pi.SetText(waitString(list...))
				}

				return false, nil
			}
		)

		if !waitCmdSettings.quiet {
			pi = wait.NewProgressIndicator(waitString())
			pi.SetTimeout(waitCmdSettings.timeout)
			pi.Start()
			defer pi.Stop()
		}

		ticker := time.NewTicker(waitCmdSettings.interval)
		timeout := time.After(waitCmdSettings.timeout)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				list, err := listUnreadyPods()
				if err != nil {
					return err
				}

				if len(list) == 0 {
					return wrap.Errorf(
						bunt.Errorf("There are no pods that start with *%s* in namespace _%s_",
							waitCmdSettings.podStartWith,
							waitCmdSettings.namespace,
						),
						"Giving up waiting for pods to become ready",
					)
				}

				var buf bytes.Buffer
				for _, pod := range list {
					fmt.Fprintf(&buf, "- %s\n", pod.Name)
				}

				return wrap.Errorf(
					fmt.Errorf("Pods that are not ready:\n%s", buf.String()),
					"Giving up waiting for pods to become ready",
				)

			case <-ticker.C:
				complete, err := checkFunc()
				if err != nil {
					return err
				}

				if complete {
					return nil
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(waitCmd)

	waitCmd.PersistentFlags().BoolVar(&waitCmdSettings.quiet, "quiet", false, "be quiet and do not output status updates")
	waitCmd.PersistentFlags().StringVar(&waitCmdSettings.namespace, "namespace", "default", "namespace to watch for")
	waitCmd.PersistentFlags().StringVar(&waitCmdSettings.podStartWith, "pod-starts-with", "", "name prefix of pods to wait for")
	waitCmd.PersistentFlags().DurationVar(&waitCmdSettings.timeout, "timeout", time.Duration(1*time.Minute), "timeout until giving up waiting on condition")
	waitCmd.PersistentFlags().DurationVar(&waitCmdSettings.interval, "interval", time.Duration(10*time.Second), "interval to check for updates")
}
