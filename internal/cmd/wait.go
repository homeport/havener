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
	"context"
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

// WaitCmdSettings contains all possible settings of the wait command
type WaitCmdSettings struct {
	Quiet        bool
	Namespace    string
	PodStartWith string
	Interval     time.Duration
	Timeout      time.Duration
}

var waitCmdSettings WaitCmdSettings

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
		case len(waitCmdSettings.PodStartWith) == 0:
			return wrap.Errorf(
				bunt.Errorf("There was no pod name prefix defined to watch for.\n\n%s", cmd.UsageString()),
				"Mandatory condition is missing",
			)
		}

		hvnr, err := havener.NewHavener()
		if err != nil {
			return wrap.Error(err, "unable to get access to cluster")
		}

		return WaitCmdFunc(hvnr, waitCmdSettings)
	},
}

func init() {
	rootCmd.AddCommand(waitCmd)

	waitCmd.PersistentFlags().BoolVar(&waitCmdSettings.Quiet, "quiet", false, "be quiet and do not output status updates")
	waitCmd.PersistentFlags().StringVar(&waitCmdSettings.Namespace, "namespace", "default", "namespace to watch for")
	waitCmd.PersistentFlags().StringVar(&waitCmdSettings.PodStartWith, "pod-starts-with", "", "name prefix of pods to wait for")
	waitCmd.PersistentFlags().DurationVar(&waitCmdSettings.Timeout, "timeout", time.Duration(1*time.Minute), "timeout until giving up waiting on condition")
	waitCmd.PersistentFlags().DurationVar(&waitCmdSettings.Interval, "interval", time.Duration(10*time.Second), "interval to check for updates")
}

// WaitCmdFunc blocks until either the configured condition is reached or the
// timeout occurred.
func WaitCmdFunc(hvnr *havener.Hvnr, settings WaitCmdSettings) error {
	var (
		pi *wait.ProgressIndicator

		waitString = func(pods ...corev1.Pod) string {
			switch len(pods) {
			case 0:
				return bunt.Sprintf("Waiting for pods starting with *%s* to become ready in namespace _%s_ ...",
					settings.PodStartWith,
					settings.Namespace,
				)

			default:
				names := make([]string, len(pods))
				for i, pod := range pods {
					names[i] = bunt.Sprintf("*%s*", pod.Name)
				}

				return fmt.Sprintf("Waiting for %s in namespace _%s_: %s",
					text.Plural(len(pods), "unready pod"),
					settings.Namespace,
					text.List(names),
				)
			}
		}

		listUnreadyPods = func() ([]corev1.Pod, int, error) {
			list, err := hvnr.Client().CoreV1().Pods(settings.Namespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return nil, 0, wrap.Errorf(err, "failed to get a list of pods in namespace %s", settings.Namespace)
			}

			var (
				unreadyPods    = []corev1.Pod{}
				podsWithPrefix = 0
			)

			for i := range list.Items {
				pod := list.Items[i]

				if strings.HasPrefix(pod.Name, settings.PodStartWith) {
					podsWithPrefix++

					var ready int
					for _, containerStatus := range pod.Status.ContainerStatuses {
						if containerStatus.Ready {
							ready++
						}
					}

					if ready != len(pod.Status.ContainerStatuses) {
						unreadyPods = append(unreadyPods, pod)
					}
				}
			}

			return unreadyPods, podsWithPrefix, nil
		}

		checkFunc = func() (bool, error) {
			unreadyPods, podsWithPrefix, err := listUnreadyPods()
			if err != nil {
				return false, err
			}

			if len(unreadyPods) == 0 && podsWithPrefix > 0 {
				return true, nil
			}

			if pi != nil {
				pi.SetText(waitString(unreadyPods...))
			}

			return false, nil
		}
	)

	if !settings.Quiet {
		pi = wait.NewProgressIndicator(waitString())
		pi.SetTimeout(settings.Timeout)
		pi.Start()
		defer pi.Stop()
	}

	ticker := time.NewTicker(settings.Interval)
	timeout := time.After(settings.Timeout)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			list, _, err := listUnreadyPods()
			if err != nil {
				return err
			}

			if len(list) == 0 {
				return wrap.Errorf(
					bunt.Errorf("There are no pods that start with *%s* in namespace _%s_",
						settings.PodStartWith,
						settings.Namespace,
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
}
