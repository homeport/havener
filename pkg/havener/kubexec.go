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
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/utils/pointer"

	"github.com/gonvenience/text"
	"github.com/mattn/go-isatty"
	"golang.org/x/term"
)

type NodeExecHelperPodConfig struct {
	namespace string
	podName   string

	Annotations    map[string]string
	ContainerImage string
	ContainerCmd   []string
	ContainerArgs  []string
	WaitTimeout    time.Duration
}

type ExecConfig struct {
	Command []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	TTY     bool
}

// PodExec executes the provided command in the referenced pod's container.
func (h *Hvnr) PodExec(pod *corev1.Pod, container string, execConfig ExecConfig) error {
	logf(Verbose, "Executing command on pod: `%v`", strings.Join(execConfig.Command, " "))

	req := h.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   execConfig.Command,
			Stdin:     execConfig.Stdin != nil,
			Stdout:    execConfig.Stdout != nil,
			Stderr:    execConfig.Stderr != nil,
			TTY:       execConfig.TTY,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(h.restconfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to initialize remote executor: %w", err)
	}

	var tsq *terminalSizeQueue
	if execConfig.TTY && isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		tsq = setupTerminalResizeWatcher()
		defer tsq.stop()
	}

	// Terminal needs to run in raw mode for the actual command execution when TTY is enabled.
	// The raw mode is the one where characters are not printed twice in the terminal. See
	// https://en.wikipedia.org/wiki/POSIX_terminal_interface#History for a bit more details.
	if execConfig.TTY {
		oldState, err := term.MakeRaw(0)
		if err != nil {
			return fmt.Errorf("failed to use raw terminal: %w", err)
		}
		defer func() { _ = term.Restore(0, oldState) }()
	}

	var streamOption = remotecommand.StreamOptions{
		Stdin:             execConfig.Stdin,
		Stdout:            execConfig.Stdout,
		Stderr:            execConfig.Stderr,
		Tty:               execConfig.TTY,
		TerminalSizeQueue: tsq,
	}

	if err = executor.StreamWithContext(h.ctx, streamOption); err != nil {
		return fmt.Errorf("failed to execute command on pod %s, container %s: %w", pod.Name, container, err)
	}

	logf(Verbose, "Successfully executed command.")
	return nil
}

// NodeExec executes the provided command on the given node.
func (h *Hvnr) NodeExec(node corev1.Node, hlpPodConfig NodeExecHelperPodConfig, execConfig ExecConfig) error {
	hlpPodConfig.podName = text.RandomStringWithPrefix("node-exec-", 15) // unique pod name
	hlpPodConfig.namespace = "kube-system"

	// Make sure to stop pod after command execution
	defer func() {
		_ = h.PurgePod(hlpPodConfig.namespace, hlpPodConfig.podName, 0, metav1.DeletePropagationBackground)
	}()

	pod, err := h.preparePodOnNode(node, hlpPodConfig)
	if err != nil {
		return err
	}

	// Unset the stderr in case TTY is set
	// https://github.com/kubernetes/kubectl/blob/5b7c8b24b4361a97ab19de1d1e301a6c1bbaed1a/pkg/cmd/exec/exec.go#L370-L372
	if execConfig.TTY {
		execConfig.Stderr = nil
	}

	// Execute command on pod and redirect output to users provided stdout and stderr
	logf(Verbose, "Executing command on node: `%v`", strings.Join(execConfig.Command, " "))

	return h.PodExec(
		pod, "node-exec-container",
		ExecConfig{
			Command: append([]string{"nsenter", "--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid", "--"}, execConfig.Command...),
			Stdin:   execConfig.Stdin,
			Stdout:  execConfig.Stdout,
			Stderr:  execConfig.Stderr,
			TTY:     execConfig.TTY,
		},
	)
}

func (h *Hvnr) preparePodOnNode(node corev1.Node, hlpPodConfig NodeExecHelperPodConfig) (*corev1.Pod, error) {
	// Add pod deletion to shutdown sequence list (in case of Ctrl+C exit)
	AddShutdownFunction(func() {
		_ = h.PurgePod(hlpPodConfig.namespace, hlpPodConfig.podName, 10, metav1.DeletePropagationBackground)
	})

	// Pod configuration
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        hlpPodConfig.podName,
			Namespace:   hlpPodConfig.namespace,
			Annotations: hlpPodConfig.Annotations,
		},
		Spec: corev1.PodSpec{
			NodeSelector: map[string]string{
				// Deploy pod on specific node using label selector
				corev1.LabelHostname: node.Name,
			},
			HostPID:                       true,
			HostNetwork:                   true,
			RestartPolicy:                 corev1.RestartPolicyNever,
			TerminationGracePeriodSeconds: pointer.Int64(0),
			Containers: []corev1.Container{
				{
					Name:            "node-exec-container",
					Image:           hlpPodConfig.ContainerImage,
					Command:         hlpPodConfig.ContainerCmd,
					Args:            hlpPodConfig.ContainerArgs,
					ImagePullPolicy: corev1.PullIfNotPresent,
					SecurityContext: &corev1.SecurityContext{
						Privileged: pointer.Bool(true),
					},
				},
			},
		},
	}

	// Mirror the node taints as tolerations to make the pod being able to start on the node
	for _, taint := range node.Spec.Taints {
		pod.Spec.Tolerations = append(pod.Spec.Tolerations, corev1.Toleration{
			Key:      taint.Key,
			Operator: corev1.TolerationOpEqual,
			Value:    taint.Value,
			Effect:   taint.Effect,
		})
	}

	// Create pod in given namespace based on configuration
	logf(Verbose, "Creating temporary pod _%s_/*%s*", hlpPodConfig.namespace, hlpPodConfig.podName)
	pod, err := h.client.CoreV1().Pods(hlpPodConfig.namespace).Create(h.ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	logf(Verbose, "Waiting for temporary pod to be started...")
	if err := h.waitForPodReadiness(pod, hlpPodConfig.WaitTimeout); err != nil {
		return nil, err
	}

	return pod, nil
}

func (h *Hvnr) waitForPodReadiness(pod *corev1.Pod, waitTimeout time.Duration) error {
	watcher, err := h.client.CoreV1().Pods(pod.Namespace).Watch(h.ctx, metav1.SingleObject(pod.ObjectMeta))
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

	timeout := time.After(waitTimeout)

	for {
		select {
		case err := <-watcherChannel:
			return err

		case <-timeout:
			description, err := h.describePod(pod)
			if err != nil {
				description = "Unable to provide further details regarding the state of the pod."
			}

			return fmt.Errorf("Giving up waiting for pod %s in namespace %s to become ready within %v: %w",
				pod.Name,
				pod.Namespace,
				waitTimeout,
				fmt.Errorf("status of pod at the moment of the timeout:\n\n%s", description),
			)
		}
	}
}
