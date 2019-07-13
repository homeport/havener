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

package havener

// batchv1 "k8s.io/api/batch/v1"
// corev1 "k8s.io/api/core/v1"
// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

import (
	"bufio"
	"fmt"
	"io"
	"time"

	"github.com/gonvenience/term"
	"github.com/gonvenience/text"
	"golang.org/x/crypto/ssh/terminal"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/utils/exec"
)

/* TODO In general, more comments and explanations are required. */

// ExecMessage is a helper structure for assigning a date to a message string
type ExecMessage struct {
	Prefix string
	Text   string
	Date   time.Time
}

// ExecResponse is a helper structure for returning the results of the
// node-exec and pod-exec commands via channels.
type ExecResponse struct {
	Messages []*ExecMessage
	Error    error
}

// PodExecDistributed executes the provided command in the referenced pod's container and returns a
// slice of all messages from the output stream instead of printing them out.
func PodExecDistributed(client kubernetes.Interface, restconfig *rest.Config, pod *corev1.Pod, podName string, command string, tty bool) ([]*ExecMessage, error) {
	reader, writer := io.Pipe()
	errChan := make(chan error, 1)

	go func() {
		err := PodExec(
			client,
			restconfig,
			pod,
			podName,
			command,
			nil,
			writer,
			writer,
			tty,
		)
		errChan <- err
		writer.Close()
	}()

	messages := []*ExecMessage{}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		messages = append(messages, &ExecMessage{Text: scanner.Text(), Date: time.Now()})
	}

	err := <-errChan
	if err != nil {
		return messages, err
	}

	return messages, nil
}

// PodExec executes the provided command in the referenced pod's container.
func PodExec(client kubernetes.Interface, restconfig *rest.Config, pod *corev1.Pod, container string, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer, tty bool) error {
	logf(Verbose, "Executing command on pod...")

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{"/bin/sh", "-c", command},
			Stdin:     stdin != nil,
			Stdout:    stdout != nil,
			Stderr:    stderr != nil,
			TTY:       tty,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(restconfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to initialize remote executor: %v", err)
	}

	var tsq *terminalSizeQueue
	if tty && term.IsTerminal() {
		tsq = setupTerminalResizeWatcher()
		defer tsq.stop()
	}

	// Terminal needs to run in raw mode for the actual command execution when TTY is enabled.
	// The raw mode is the one where characters are not printed twice in the terminal. See
	// https://en.wikipedia.org/wiki/POSIX_terminal_interface#History for a bit more details.
	if tty {
		if stateToBeRestored, err := terminal.MakeRaw(0); err == nil {
			defer terminal.Restore(0, stateToBeRestored)
		}
	}

	if err = executor.Stream(remotecommand.StreamOptions{Stdin: stdin, Stdout: stdout, Stderr: stderr, Tty: tty, TerminalSizeQueue: tsq}); err != nil {
		switch err := err.(type) {
		case exec.CodeExitError:
			// In case this needs to be refactored in a way where the exit code of the remote command is interesting
			return fmt.Errorf("remote command failed with exit code %d", err.Code)

		default:
			return fmt.Errorf("could not execute: %v", err)
		}
	}

	logf(Verbose, "Successfully executed command.")

	return nil
}

// NodeExec executes the provided command on the given node.
func NodeExec(client kubernetes.Interface, restconfig *rest.Config, node *corev1.Node, containerImage string, timeoutSeconds int, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer, tty bool, distributed bool) ([]*ExecMessage, error) {
	logf(Verbose, "Executing command on node...")
	var err error

	namespace := "kube-system"
	podName := text.RandomStringWithPrefix("node-exec-", 15) // Create unique pod/container name
	trueThat := true

	// Pod confoguration
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			NodeSelector:  map[string]string{"kubernetes.io/hostname": node.Name}, // Deploy pod on specific node using label selector
			HostPID:       true,
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  podName,
					Image: containerImage,
					SecurityContext: &corev1.SecurityContext{
						Privileged: &trueThat,
					},
					Stdin: stdin != nil,
				},
			},
		},
	}

	logf(Verbose, "Creating pod...")

	// Create pod in given namespace based on configuration
	pod, err = client.CoreV1().Pods(namespace).Create(pod)
	if err != nil {
		return nil, err
	}
	// Stop pod and container after command execution
	defer func() {
		jobDeletionGracePeriod := int64(10)
		propagationPolicy := metav1.DeletePropagationForeground
		client.CoreV1().Pods(namespace).Delete(podName, &metav1.DeleteOptions{
			GracePeriodSeconds: &jobDeletionGracePeriod,
			PropagationPolicy:  &propagationPolicy,
		})
	}()

	logf(Verbose, "Waiting for pod to be started...")
	if err := waitForPodReadiness(client, namespace, pod, timeoutSeconds); err != nil {
		return nil, err
	}

	if distributed {
		messages, err := PodExecDistributed(
			client,
			restconfig,
			pod,
			podName,
			fmt.Sprintf("nsenter --target 1 --mount --uts --ipc --net --pid -- %s", command),
			tty,
		)
		return messages, err

	}

	// Execute command on pod and redirect output to users provided stdout and stderr
	return nil, PodExec(
		client,
		restconfig,
		pod,
		podName,
		fmt.Sprintf("nsenter --target 1 --mount --uts --ipc --net --pid -- %s", command),
		stdin,
		stdout,
		stderr,
		tty,
	)
}

func waitForPodReadiness(client kubernetes.Interface, namespace string, pod *corev1.Pod, timeoutSeconds int) error {
	watcher, err := client.CoreV1().Pods(namespace).Watch(metav1.SingleObject(pod.ObjectMeta))
	if err != nil {
		return err
	}

	watcherChannel := make(chan error)
	go func(watcher watch.Interface) {
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Modified:
				pod = event.Object.(*corev1.Pod)
				for _, cond := range pod.Status.Conditions {
					if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
						watcher.Stop()
						watcherChannel <- nil
					}
				}

			default:
				watcherChannel <- fmt.Errorf("unknown event type occurred: %v", event.Type)
			}
		}
	}(watcher)

	timeout := time.After(time.Duration(timeoutSeconds) * time.Second)

	for {
		select {
		case err := <-watcherChannel:
			if err != nil {
				return err
			}

		case <-timeout:
			return fmt.Errorf("failed to get pod after %d seconds", timeoutSeconds)
		}

		return nil
	}
}
