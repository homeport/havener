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
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gonvenience/term"
	"github.com/gonvenience/wrap"
	"github.com/homeport/havener/pkg/havener"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
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
different syntax. In contrast to kubectl, you do not have to specify
the namespace of the pod.

If no namespace is given, havener will search all namespaces for
a pod that matches the name.

Also, you can omit the command which will result in the default
command: ` + podDefaultCommand + `. For example 'havener pod-exec api-0' will search
for a pod named 'api-0' in all namespaces and open a shell if found.

In case no container name is given, havener will assume you want to
execute the command in the first container found in the pod.

If you run the 'pod-exec' without any additional arguments, it will print a
list of available pods.

For convenience, if the target pod name _all_ is used, havener will look up
all pods in all namespaces automatically.
`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return execInClusterPods(args)
	},
}

func init() {
	rootCmd.AddCommand(podExecCmd)
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	podExecCmd.PersistentFlags().BoolVar(&podExecNoTty, "no-tty", false, "do not allocate pseudo-terminal for command execution")
	podExecCmd.PersistentFlags().BoolVar(&podExecBlock, "block", false, "show distributed shell output as block for each pod")
}

func execInClusterPods(args []string) error {
	hvnr, err := havener.NewHavener()
	if err != nil {
		return wrap.Error(err, "unable to get access to cluster")
	}

	var (
		podMap          map[*corev1.Pod][]string
		countContainers int
		input           string
		command         []string
	)

	switch {
	case len(args) >= 2: // pod and command is given
		input, command = args[0], args[1:]
		podMap, err = lookupPodsByName(hvnr.Client(), input)
		if err != nil {
			return err
		}

	case len(args) == 1: // only pod is given
		input, command = args[0], []string{podDefaultCommand}
		podMap, err = lookupPodsByName(hvnr.Client(), input)
		if err != nil {
			return err
		}

	default:
		return availablePodsError(hvnr.Client(), "no pod name specified")
	}

	// Count number of containers from all pods
	// We only get all containers per pod, when "all" is specified
	for _, containers := range podMap {
		for range containers {
			countContainers++
		}
	}

	// In case the current process does not run in a terminal, disable the
	// default TTY behavior.
	if !term.IsTerminal() {
		podExecNoTty = true
	}

	// Single pod mode, use default streams and run pod execute function
	if len(podMap) == 1 {
		for pod, containers := range podMap {
			for i := range containers {
				return hvnr.PodExec(
					pod,
					containers[i],
					command,
					os.Stdin,
					os.Stdout,
					os.Stderr,
					!podExecNoTty,
				)
			}

		}
	}

	// In distributed shell mode, TTY is forced to be disabled
	podExecNoTty = true

	var (
		wg      = &sync.WaitGroup{}
		readers = duplicateReader(os.Stdin, countContainers)
		output  = make(chan OutputMsg)
		errors  = make(chan error, countContainers)
		printer = make(chan bool, 1)
		counter = 0
	)
	// wg.Add(countContainers)
	for pod, containers := range podMap {
		for i := range containers {
			wg.Add(1)
			go func(pod *corev1.Pod, container string, reader io.Reader) {
				defer wg.Done()
				origin := fmt.Sprintf("%s/%s", pod.Name, container)
				errors <- hvnr.PodExec(
					pod,
					container,
					command,
					reader,
					chanWriter("StdOut", origin, output),
					chanWriter("StdErr", origin, output),
					!podExecNoTty,
				)
			}(pod, containers[i], readers[counter])
			counter++
		}
	}

	// Start the respective output printer in a separate Go routine
	go func() {
		if podExecBlock {
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

func lookupPodContainers(client kubernetes.Interface, p *corev1.Pod) (containerList []string, err error) {
	for _, c := range p.Spec.Containers {
		containerList = append(containerList, c.Name)
	}
	return containerList, err
}

func lookupAllPods(client kubernetes.Interface, namespaces []string) (map[*corev1.Pod][]string, error) {
	var podLists = make(map[*corev1.Pod][]string)
	for _, namespace := range namespaces {
		podsPerNs, err := client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for i := range podsPerNs.Items {
			listOfContainers, err := lookupPodContainers(client, &(podsPerNs.Items[i]))
			if err != nil {
				return nil, err
			}
			podLists[&(podsPerNs.Items[i])] = listOfContainers
		}
	}
	return podLists, nil
}

func lookupPodsByName(client kubernetes.Interface, input string) (map[*corev1.Pod][]string, error) {
	inputList := strings.Split(input, ",")

	podList := make(map[*corev1.Pod][]string, len(inputList))
	for _, podName := range inputList {
		splited := strings.Split(podName, "/")

		switch len(splited) {
		case 1: // only the pod name is given
			namespaces, err := havener.ListNamespaces(client)
			if err != nil {
				return nil, err
			}

			if input == "all" {
				return lookupAllPods(client, namespaces)
			}

			pods := []*corev1.Pod{}
			for _, namespace := range namespaces {
				if pod, err := client.CoreV1().Pods(namespace).Get(context.TODO(), input, metav1.GetOptions{}); err == nil {
					pods = append(pods, pod)
				}
			}

			switch {
			case len(pods) < 1:
				return nil, availablePodsError(client, fmt.Sprintf("unable to find a pod named %s", input))

			case len(pods) > 1:
				return nil, fmt.Errorf("more than one pod named %s found, please specify a namespace", input)
			}

			podList[pods[0]] = []string{pods[0].Spec.Containers[0].Name}

		case 2: // namespace, and pod name is given
			namespace, podName := splited[0], splited[1]
			pod, err := client.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
			if err != nil {
				return nil, availablePodsError(client, fmt.Sprintf("pod %s not found", input))
			}

			podList[pod] = []string{pod.Spec.Containers[0].Name}

		case 3: // namespace, pod, and container name is given
			namespace, podName, container := splited[0], splited[1], splited[2]
			pod, err := client.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
			if err != nil {
				return nil, availablePodsError(client, fmt.Sprintf("pod %s not found", input))
			}

			podList[pod] = []string{container}

		default:
			return nil, fmt.Errorf("unsupported naming schema, it needs to be [namespace/]pod[/container]")
		}
	}

	return podList, nil
}

func availablePodsError(client kubernetes.Interface, title string) error {
	pods, err := havener.ListPods(client)
	if err != nil {
		return wrap.Error(err, "failed to list all pods in cluster")
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

	return wrap.Error(
		fmt.Errorf("> Usage:\npod-exec [flags] <pod> <command>\n> List of available pods:\n%s",
			strings.Join(podList, "\n"),
		),
		title,
	)
}
