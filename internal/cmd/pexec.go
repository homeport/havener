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
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"

	"github.com/gonvenience/bunt"
	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	podExecDefaultCommand = "/bin/sh"
)

type target struct {
	namespace     string
	podName       string
	containerName string
}

var podExecCmdSettings struct {
	stdin        bool
	tty          bool
	notty        bool
	printAsBlock bool
}

// podExecCmd represents the pod-exec command
var podExecCmd = &cobra.Command{
	Use:     "pod-exec [flags] [[<namespace>/]<pod>[/container]] [<command>]",
	Aliases: []string{"pe"},
	Short:   "Execute command on Kubernetes pod",
	Long: bunt.Sprintf(`*Execute a command on a pod*

This is similar to the kubectl exec command with just a slightly different
syntax. In contrast to kubectl, you do not have to specify the namespace
of the pod.

If no namespace is given, *havener* will search all namespaces for a pod that
matches the name.

Also, you can omit the command which will result in the default command: %s.
For example _havener pod-exec api-0_ will search for a pod named _api-0_ in all
namespaces and open a shell if found.

In case no container name is given, *havener* will assume you want to execute the
command in the first container found in the pod.

If you run the 'pod-exec' without any additional arguments, it will print a
list of available pods.

For convenience, if the target pod name _all_ is used, *havener* will look up
all pods in all namespaces automatically.
`, podExecDefaultCommand),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check edge case for deprecated command-line flag
		if cmd.Flags().Changed("no-tty") {
			// Bail out if both the new and the old flag are used at the same time
			if cmd.Flags().Changed("tty") {
				return fmt.Errorf("cannot use --no-tty and --tty at the same time")
			}

			// If only --no-tty is used, continue to accept its input
			podExecCmdSettings.tty = !podExecCmdSettings.notty
		}

		hvnr, err := havener.NewHavener(havener.WithContext(cmd.Context()), havener.WithKubeConfigPath(kubeConfig))
		if err != nil {
			return fmt.Errorf("unable to get access to cluster: %w", err)
		}

		return execInClusterPods(hvnr, args)
	},
}

func init() {
	rootCmd.AddCommand(podExecCmd)

	podExecCmd.Flags().SortFlags = false
	podExecCmd.Flags().BoolVarP(&podExecCmdSettings.stdin, "stdin", "i", false, "Pass stdin to the container")
	podExecCmd.Flags().BoolVarP(&podExecCmdSettings.tty, "tty", "t", false, "Stdin is a TTY")
	podExecCmd.Flags().BoolVar(&podExecCmdSettings.printAsBlock, "block", false, "show distributed shell output as block for each pod")

	// Deprecated/old flags
	podExecCmd.Flags().BoolVar(&podExecCmdSettings.notty, "no-tty", false, "do not allocate pseudo-terminal for command execution")
	_ = podExecCmd.Flags().MarkDeprecated("no-tty", "use --tty flag instead")
}

func execInClusterPods(hvnr havener.Havener, args []string) error {
	var (
		podMap          map[*corev1.Pod][]string
		countContainers int
		input           string
		command         []string
		err             error
	)

	switch {
	case len(args) >= 2: // pod and command is given
		input, command = args[0], args[1:]
		podMap, err = lookupPodsByName(hvnr, input)
		if err != nil {
			return err
		}

	case len(args) == 1: // only pod is given
		input, command = args[0], []string{podExecDefaultCommand}
		podMap, err = lookupPodsByName(hvnr, input)
		if err != nil {
			return err
		}

	default:
		return availablePodsError(hvnr, "no pod name specified")
	}

	// Count number of containers from all pods
	// We only get all containers per pod, when "all" is specified
	for _, containers := range podMap {
		for range containers {
			countContainers++
		}
	}

	if !isStdinTerminal() {
		podExecCmdSettings.tty = false
	}

	var in io.Reader
	if podExecCmdSettings.stdin {
		in = os.Stdin
	}

	// Single pod mode, use default streams and run pod execute function
	if len(podMap) == 1 {
		for pod, containers := range podMap {
			for i := range containers {
				return hvnr.PodExec(
					pod, containers[i],
					havener.ExecConfig{
						Command: command,
						Stdin:   in,
						Stdout:  os.Stdout,
						Stderr:  os.Stderr,
						TTY:     podExecCmdSettings.tty,
					},
				)
			}
		}
	}

	// In distributed shell mode, TTY is forced to be disabled
	podExecCmdSettings.stdin = false
	podExecCmdSettings.tty = false

	var (
		wg      = &sync.WaitGroup{}
		output  = make(chan OutputMsg)
		errors  = make(chan error, countContainers)
		printer = make(chan bool, 1)
		counter = 0
	)

	for pod, containers := range podMap {
		for i := range containers {
			wg.Add(1)
			go func(pod *corev1.Pod, container string) {
				defer wg.Done()
				origin := fmt.Sprintf("%s/%s", pod.Name, container)
				errors <- hvnr.PodExec(
					pod, container,
					havener.ExecConfig{
						Command: command,
						Stdin:   nil, // Disabled for now until reliable input duplication works
						Stdout:  chanWriter("StdOut", origin, output),
						Stderr:  chanWriter("StdErr", origin, output),
						TTY:     podExecCmdSettings.tty,
					},
				)
			}(pod, containers[i])
			counter++
		}
	}

	// Start the respective output printer in a separate Go routine
	go func() {
		if podExecCmdSettings.printAsBlock {
			PrintOutputMessageAsBlock(output)

		} else {
			PrintOutputMessage(output)
		}

		printer <- true
	}()

	wg.Wait()
	close(errors)
	close(output)

	if viper.GetBool("verbose") {
		if err := combineErrorsFromChannel("pod command execution failed", errors); err != nil {
			return err
		}
	}

	<-printer
	return nil
}

func containerNames(pod *corev1.Pod) []string {
	var result []string
	for _, container := range pod.Spec.Containers {
		result = append(result, container.Name)
	}

	return result
}

func (t target) String() string {
	return fmt.Sprintf("%s/%s/%s",
		t.namespace,
		t.podName,
		t.containerName,
	)
}

func lookupPodsByName(h havener.Havener, input string) (map[*corev1.Pod][]string, error) {
	var targets = map[*corev1.Pod][]string{}

	// In case special term `all` is used, immediately return the full list of all pod containers
	if input == "all" {
		list, err := h.ListPods()
		if err != nil {
			return nil, err
		}

		for _, pod := range list {
			targets[pod] = containerNames(pod)
		}

		return targets, nil
	}

	var keys, candidates []target
	var lookUp = map[target]*corev1.Pod{}

	for _, str := range strings.Split(input, ",") {
		var splited = strings.Split(str, "/")
		switch len(splited) {
		case 1: // only the pod name is given
			namespace, podName, containerName := "*", splited[0], "*"
			candidates = append(candidates, target{namespace, podName, containerName})
			if err := updateLookUps(h, &keys, lookUp, namespace); err != nil {
				return nil, err
			}

		case 2: // namespace, and pod name is given
			namespace, podName, containerName := splited[0], splited[1], "*"
			candidates = append(candidates, target{namespace, podName, containerName})
			if err := updateLookUps(h, &keys, lookUp, namespace); err != nil {
				return nil, err
			}

		case 3: // namespace, pod, and container name is given
			namespace, podName, containerName := splited[0], splited[1], splited[2]
			candidates = append(candidates, target{namespace, podName, containerName})
			if err := updateLookUps(h, &keys, lookUp, namespace); err != nil {
				return nil, err
			}

		default:
			return nil, fmt.Errorf("unsupported naming schema, it needs to be [namespace/]pod[/container]")
		}
	}

	for _, candidate := range candidates {
		for _, key := range keys {
			match, err := filepath.Match(candidate.String(), key.String())
			if err != nil {
				return nil, err
			}

			if match {
				pod := lookUp[key]
				targets[pod] = append(targets[pod], key.containerName)
			}
		}
	}

	return targets, nil
}

func updateLookUps(h havener.Havener, keys *[]target, lookUp map[target]*corev1.Pod, namespace string) error {
	var namespaces []string
	if namespace != "*" {
		namespaces = append(namespaces, namespace)
	}

	list, err := h.ListPods(namespaces...)
	if err != nil {
		return err
	}

	for i, pod := range list {
		for _, containerName := range containerNames(pod) {
			key := target{pod.Namespace, pod.Name, containerName}
			*keys = append(*keys, key)
			lookUp[key] = list[i]
		}
	}

	return nil
}

func availablePodsError(h havener.Havener, format string, a ...any) error {
	pods, err := h.ListPods()
	if err != nil {
		return fmt.Errorf("failed to list all pods in cluster: %w", err)
	}

	var targets []string
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			target := target{pod.Namespace, pod.Name, container.Name}
			targets = append(targets, target.String())
		}
	}

	return fmt.Errorf("%s: %w",
		fmt.Sprintf(format, a...),
		fmt.Errorf("List of available pods:\n%s",
			strings.Join(targets, "\n"),
		),
	)
}
