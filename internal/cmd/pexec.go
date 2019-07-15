// Copyright © 2019 The Havener
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
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"

	"github.com/gonvenience/wrap"
	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
)

const podDefaultCommand = "/bin/sh"

var (
	podExecNoTty bool
	podExecBlock bool
)

// podExecCmd represents the pod-exec command
var podExecCmd = &cobra.Command{
	Use:     "pod-exec [flags] [[<namespace>/]<pod>[/container]] [<command>]",
	Aliases: []string{"pe"},
	Short:   "Execute command on Kubernetes pod",
	Long: `Execute a shell command on a pod.

This is similar to the kubectl exec command with just a slightly
different syntax. In constrast to kubectl, you do not have to specify
the namespace of the pod.

If no namespace is given, havener will search all namespaces for
a pod that matches the name.

Also, you can omit the command which will result in the default
command: ` + podDefaultCommand + `. For example 'havener pod-exec api-0' will search
for a pod named 'api-0' in all namespaces and open a shell if found.

In case no container name is given, havener will assume you want to
execute the command in the first container found in the pod.

`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return execInClusterPods(args)
	},
}

func init() {
	rootCmd.AddCommand(podExecCmd)

	podExecCmd.PersistentFlags().BoolVar(&podExecNoTty, "no-tty", false, "do not allocate pseudo-terminal for command execution")
	podExecCmd.PersistentFlags().BoolVar(&podExecBlock, "block", false, "show distributed shell output as block for each pod")
}

func execInClusterPods(args []string) error {
	client, restconfig, err := havener.OutOfClusterAuthentication("")
	if err != nil {
		return &ErrorWithMsg{"failed to connect to Kubernetes cluster", err}
	}

	var (
		podMap  map[*corev1.Pod]string
		input   string
		command []string
	)

	switch {
	case len(args) >= 2: // pod and command is given
		input, command = args[0], args[1:]
		podMap, err = lookupPodsByName(client, input)
		if err != nil {
			return err
		}

	case len(args) == 1: // only pod is given
		input, command = args[0], []string{podDefaultCommand}
		podMap, err = lookupPodsByName(client, input)
		if err != nil {
			return err
		}

	default:
		return availablePodsError(client, "no pod name specified")
	}

	// Single pod mode, use default streams and run pod execute function
	if len(podMap) == 1 {
		for pod, container := range podMap {
			return havener.PodExec(
				client,
				restconfig,
				pod,
				container,
				command,
				os.Stdin,
				os.Stdout,
				os.Stderr,
				!podExecNoTty,
			)
		}
	}

	// In distributed shell mode, TTY is forced to be disabled
	podExecNoTty = true

	var (
		wg      = &sync.WaitGroup{}
		readers = duplicateReader(os.Stdin, len(podMap))
		output  = make(chan OutputMsg)
		errors  = make(chan error, len(podMap))
		printer = make(chan bool, 1)
		counter = 0
	)

	wg.Add(len(podMap))
	for pod, container := range podMap {
		go func(pod *corev1.Pod, container string, reader io.Reader) {
			defer func() {
				wg.Done()
			}()

			origin := fmt.Sprintf("%s/%s", pod.Name, container)
			errors <- havener.PodExec(
				client,
				restconfig,
				pod,
				container,
				command,
				reader,
				chanWriter("StdOut", origin, output),
				chanWriter("StdErr", origin, output),
				!podExecNoTty,
			)
		}(pod, container, readers[counter])

		counter++
	}

	// Start the respective output printer in a separate Go routine
	go func() {
		if podExecBlock {
			PrintOutputMessageAsBlock(output, len(podMap))

		} else {
			PrintOutputMessage(output, len(podMap))
		}

		printer <- true
	}()

	wg.Wait()
	close(errors)
	close(output)

	errList := []error{}
	for err := range errors {
		if err != nil {
			errList = append(errList, err)
		}
	}

	if len(errList) != 0 {
		return wrap.Errors(errList, "pod command execution failed")
	}

	<-printer
	return nil
}

func lookupPodsByName(client kubernetes.Interface, input string) (map[*corev1.Pod]string, error) {
	inputList := strings.Split(input, ",")

	podList := make(map[*corev1.Pod]string, len(inputList))
	for _, podName := range inputList {
		splited := strings.Split(podName, "/")

		switch len(splited) {
		case 1: // only the pod name is given
			namespaces, err := havener.ListNamespaces(client)
			if err != nil {
				return nil, err
			}

			pods := []*corev1.Pod{}
			for _, namespace := range namespaces {
				if pod, err := client.CoreV1().Pods(namespace).Get(input, metav1.GetOptions{}); err == nil {
					pods = append(pods, pod)
				}
			}

			switch {
			case len(pods) < 1:
				return nil, availablePodsError(client, fmt.Sprintf("unable to find a pod named %s", input))

			case len(pods) > 1:
				return nil, fmt.Errorf("more than one pod named %s found, please specify a namespace", input)
			}

			podList[pods[0]] = pods[0].Spec.Containers[0].Name

		case 2: // namespace, and pod name is given
			namespace, podName := splited[0], splited[1]
			pod, err := client.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
			if err != nil {
				return nil, availablePodsError(client, fmt.Sprintf("pod %s not found", input))
			}

			podList[pod] = pod.Spec.Containers[0].Name

		case 3: // namespace, pod, and container name is given
			namespace, podName, container := splited[0], splited[1], splited[2]
			pod, err := client.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
			if err != nil {
				return nil, availablePodsError(client, fmt.Sprintf("pod %s not found", input))
			}

			podList[pod] = container

		default:
			return nil, fmt.Errorf("unsupported naming schema, it needs to be [namespace/]pod[/container]")
		}
	}

	return podList, nil
}

func availablePodsError(client kubernetes.Interface, title string) error {
	pods, err := havener.ListPods(client)
	if err != nil {
		return &ErrorWithMsg{"failed to list all pods in cluster", err}
	}
	podList := []string{}
	for _, pod := range pods {
		for i := range pod.Spec.Containers {
			podList = append(podList, fmt.Sprintf("%s/%s/%s",
				pod.ObjectMeta.Namespace,
				pod.Name,
				pod.Spec.Containers[i].Name,
			))
		}
	}

	return &ErrorWithMsg{title,
		fmt.Errorf("> Usage:\npod-exec [flags] <pod> <command>\n> List of available pods:\n%s",
			strings.Join(podList, "\n"),
		)}

}
