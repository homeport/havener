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
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gonvenience/term"
	"github.com/gonvenience/text"
	"github.com/gonvenience/wrap"
	"golang.org/x/crypto/ssh/terminal"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

/* TODO In general, more comments and explanations are required. */

// PodExec executes the provided command in the referenced pod's container.
func PodExec(client kubernetes.Interface, restconfig *rest.Config, pod *corev1.Pod, container string, command []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, tty bool) error {
	logf(Verbose, "Executing command on pod: %#v", command)

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     stdin != nil,
			Stdout:    stdout != nil,
			Stderr:    stderr != nil,
			TTY:       tty,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(restconfig, "POST", req.URL())
	if err != nil {
		return wrap.Error(err, "failed to initialize remote executor")
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
		return wrap.Errorf(err, "failed to exectue command on pod %s, container %s", pod.Name, container)
	}

	logf(Verbose, "Successfully executed command.")

	return nil
}

// NodeExec executes the provided command on the given node.
func NodeExec(client kubernetes.Interface, restconfig *rest.Config, node *corev1.Node, containerImage string, timeoutSeconds int, command []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, tty bool) error {
	logf(Verbose, "Executing command on node: %#v", command)

	namespace := "kube-system"
	podName := text.RandomStringWithPrefix("node-exec-", 15) // Create unique pod/container name
	trueThat := true

	// Make sure to stop pod after command execution
	defer PurgePod(client, namespace, podName, 10, metav1.DeletePropagationForeground)
	AddShutdownFunction(func() {
		PurgePod(client, namespace, podName, 10, metav1.DeletePropagationBackground)
	})

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

	// Create pod in given namespace based on configuration
	logf(Verbose, "Creating temporary pod %s in namespace %s", podName, namespace)
	pod, err := client.CoreV1().Pods(namespace).Create(pod)
	if err != nil {
		return err
	}

	logf(Verbose, "Waiting for temporary pod to be started...")
	if err := waitForPodReadiness(client, namespace, pod, timeoutSeconds); err != nil {
		return err
	}

	// Execute command on pod and redirect output to users provided stdout and stderr
	return PodExec(
		client,
		restconfig,
		pod,
		podName,
		append([]string{"nsenter", "--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid", "--"}, command...),
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
