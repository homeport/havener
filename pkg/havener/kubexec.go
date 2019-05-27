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
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"
	"github.com/homeport/gonvenience/pkg/v1/text"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/exec"
)

/* TODO In general, more comments and explanations are required. */

/* TODO Introduce re-usable "runner" pods in case there are a series of
   commands to be executed on a node. In this case, do do not delete the job,
   but keep it running so that we have the pod around. Then, subsequent execs
   can be done to this pod. */

/* TODO The functions in here lack a common style and symmetry. One needs the
   the Kubernetes client and a POD reference, the other just a name. Think
   about ideas on whether it makes sense to harmonize this a little bit. */

/* TODO Introduce a timeout to the wait loop that checks whether the pod
   becomes ready. Idea would be like an additional check if N seconds elapsed
   and no result was return to abort the execution completely. */

// defaultTimeoutForGetPod is the timeout in seconds to wait until a newly created job spawned the actual pod
const defaultTimeoutForGetPod = 5

// PodExec executes the provided command in the referenced pod's container.
func PodExec(client kubernetes.Interface, restconfig *rest.Config, pod *corev1.Pod, container string, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer, tty bool) error {
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

	// Terminal needs to run in raw mode for the actual command execution when TTY is enabled.
	// The raw mode is the one where characters are not printed twice in the terminal. See
	// https://en.wikipedia.org/wiki/POSIX_terminal_interface#History for a bit more details.
	if tty {
		if stateToBeRestored, err := terminal.MakeRaw(0); err == nil {
			defer terminal.Restore(0, stateToBeRestored)
		}
	}

	if err = executor.Stream(remotecommand.StreamOptions{Stdin: stdin, Stdout: stdout, Stderr: stderr, Tty: tty}); err != nil {
		switch err := err.(type) {
		case exec.CodeExitError:
			// In case this needs to be refactored in a way where the exit code of the remote command is interesting
			return fmt.Errorf("remote command failed with exit code %d", err.Code)

		default:
			return fmt.Errorf("could not execute: %v", err)
		}
	}

	return nil
}

// NodeExec executes the provided command on the given node.
func NodeExec(client kubernetes.Interface, restconfig *rest.Config, node string, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer, tty bool) error {
	// TODO These fields should be made customizable using a configuration file
	namespace := "kube-system"
	containerName := "runon"
	containerImage := "debian:jessie"

	jobName := strings.ToLower(text.RandomStringWithPrefix("node-runner-", 24))
	trueThat := true
	jobDef := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector:  map[string]string{"kubernetes.io/hostname": node},
					HostPID:       true,
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    containerName,
							Image:   containerImage,
							Command: []string{"sleep", "8h"},
							SecurityContext: &corev1.SecurityContext{
								Privileged: &trueThat,
							},
						},
					},
				},
			},
		},
	}

	job, err := client.BatchV1().Jobs(namespace).Create(jobDef)
	if err != nil {
		return err
	}

	// Make sure that both the job and the pod it spawned are removed
	// from the clusters once this function reaches its end.
	defer func() {
		jobDeletionGracePeriod := int64(10)
		propagationPolicy := metav1.DeletePropagationForeground
		client.BatchV1().Jobs(namespace).Delete(job.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: &jobDeletionGracePeriod,
			PropagationPolicy:  &propagationPolicy,
		})
	}()

	pod, err := getPodUsingListOptions(client, namespace, metav1.ListOptions{LabelSelector: fmt.Sprintf("controller-uid=%s,job-name=%s", job.ObjectMeta.UID, jobName)})
	if err != nil {
		return err
	}

	// The reference to the pod that was spawned for the job.
	watcher, err := client.CoreV1().Pods(namespace).Watch(metav1.SingleObject(pod.ObjectMeta))
	if err != nil {
		return err
	}

	// Wait until the pod reports Ready state
	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Modified:
			pod = event.Object.(*corev1.Pod)
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					watcher.Stop()
				}
			}

		default:
			return fmt.Errorf("unknown event type occurred: %v", event.Type)
		}
	}

	return PodExec(
		client,
		restconfig,
		pod,
		pod.Spec.Containers[0].Name,
		fmt.Sprintf("nsenter --target 1 --mount --uts --ipc --net --pid -- %s", command),
		stdin,
		stdout,
		stderr,
		tty,
	)
}

func getPodUsingListOptions(client kubernetes.Interface, namespace string, listOptions metav1.ListOptions) (*corev1.Pod, error) {
	timeout := time.After(defaultTimeoutForGetPod * time.Second)

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pods, err := client.CoreV1().Pods(namespace).List(listOptions)
			if err != nil {
				return nil, err
			}

			if len(pods.Items) == 1 {
				return &pods.Items[0], nil
			}

		case <-timeout:
			return nil, fmt.Errorf("failed to get pod after %d seconds", defaultTimeoutForGetPod)
		}
	}
}
