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

package havener

import (
	"context"
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
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// PodExec executes the provided command in the referenced pod's container.
func (h *Hvnr) PodExec(pod *corev1.Pod, container string, command []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, tty bool) error {
	logf(Verbose, "Executing command on pod: %#v", command)

	req := h.client.CoreV1().RESTClient().Post().
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

	executor, err := remotecommand.NewSPDYExecutor(h.restconfig, "POST", req.URL())
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
		return wrap.Errorf(err, "failed to execute command on pod %s, container %s", pod.Name, container)
	}

	logf(Verbose, "Successfully executed command.")

	return nil
}

// NodeExec executes the provided command on the given node.
func (h *Hvnr) NodeExec(node corev1.Node, containerImage string, timeoutSeconds int, command []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, tty bool) error {
	logf(Verbose, "Executing command on node: %#v", command)

	var (
		podName   = text.RandomStringWithPrefix("node-exec-", 15) // unique pod name
		namespace = "kube-system"
	)

	pod, err := h.preparePodOnNode(node, namespace, podName, containerImage, timeoutSeconds, stdin != nil)
	if err != nil {
		return err
	}

	// Make sure to stop pod after command execution
	defer PurgePod(h.client, pod.Namespace, pod.Name, 10, metav1.DeletePropagationForeground)

	// Execute command on pod and redirect output to users provided stdout and stderr
	return h.PodExec(
		pod,
		"node-exec-container",
		append([]string{"nsenter", "--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid", "--"}, command...),
		stdin,
		stdout,
		stderr,
		tty,
	)
}

func (h *Hvnr) preparePodOnNode(node corev1.Node, namespace string, name string, containerImage string, timeoutSeconds int, useStdin bool) (*corev1.Pod, error) {
	trueThat := true

	// Add pod deletion to shutdown sequence list (in case of Ctrl+C exit)
	AddShutdownFunction(func() {
		PurgePod(h.client, namespace, name, 10, metav1.DeletePropagationBackground)
	})

	// Pod confoguration
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			NodeSelector:  map[string]string{"kubernetes.io/hostname": node.Name}, // Deploy pod on specific node using label selector
			HostPID:       true,
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "node-exec-container",
					Image: containerImage,
					Stdin: useStdin,
					SecurityContext: &corev1.SecurityContext{
						Privileged: &trueThat,
					},
				},
			},
		},
	}

	// Create pod in given namespace based on configuration
	logf(Verbose, "Creating temporary pod *%s* in namespace *%s*", name, namespace)
	pod, err := h.client.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	logf(Verbose, "Waiting for temporary pod to be started...")
	if err := h.waitForPodReadiness(namespace, pod, timeoutSeconds); err != nil {
		return nil, err
	}

	return pod, nil
}

func (h *Hvnr) waitForPodReadiness(namespace string, pod *corev1.Pod, timeoutSeconds int) error {
	watcher, err := h.client.CoreV1().Pods(namespace).Watch(context.TODO(), metav1.SingleObject(pod.ObjectMeta))
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
			return err

		case <-timeout:
			description, err := h.describePod(pod)
			if err != nil {
				description = "Unable to provide further details regarding the state of the pod."
			}

			return wrap.Errorf(fmt.Errorf("status of pod at the moment of the timeout:\n\n%s", description),
				"Giving up waiting for pod %s in namespace %s to become ready within %s",
				pod.Name,
				pod.Namespace,
				text.Plural(timeoutSeconds, "second"),
			)
		}
	}
}
