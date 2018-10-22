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
	"bytes"
	"fmt"
	"strings"

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

/* TODO The functions in here lack a common style and symetry. One needs the
   the Kubernetes client and a POD reference, the other just a name. Think
   about ideas on whether it makes sense to harmonize this a little bit. */

/* TODO Introduce a timeout to the wait loop that checks whether the pod
   becomes ready. Idea would be like an addional check if N seconds elapsed
   and no result was return to abort the execution completely. */

/* TODO Rework result types, so that we do not have a seperate STDOUT and
   STDERR, but maybe only one output stream that has both streams combined.
   However, this should not be simply a 2>&1, but should still come with the
   possible distinction between the different outputs if required. */

// PodExec executes the provided command in the referenced pod.
func PodExec(client kubernetes.Interface, restconfig *rest.Config, pod *corev1.Pod, command string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pod.Spec.Containers[0].Name,
			Command:   []string{"/bin/sh", "-c", command},
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(restconfig, "POST", req.URL())
	if err != nil {
		return "", "", fmt.Errorf("failed to initialize remote executor: %v", err)
	}

	if err = executor.Stream(remotecommand.StreamOptions{Stdout: &stdout, Stderr: &stderr, Tty: false}); err != nil {
		switch err.(type) {
		case exec.CodeExitError:
			// In case this needs to be refactored in a way where the exit code of the remote command is interesting
			return "", "", fmt.Errorf("remote command failed with exit code %d", err.(exec.CodeExitError).Code)

		default:
			return "", "", fmt.Errorf("could not execute: %v", err)
		}
	}

	return stdout.String(), stderr.String(), nil
}

// NodeExec executes the provided command on the given node.
func NodeExec(node string, command string) (string, string, error) {
	// TODO These fields should be made customizable using a configuration file
	namespace := "kube-system"
	containerName := "runon"
	containerImage := "debian:jessie"

	client, restconfig, err := OutOfClusterAuthentication()
	if err != nil {
		return "", "", err
	}

	jobName := strings.ToLower(RandomStringWithPrefix("node-runner-", 24))
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
		return "", "", err
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

	pods, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: fmt.Sprintf("controller-uid=%s,job-name=%s", job.ObjectMeta.UID, jobName)})
	if err != nil {
		return "", "", err
	}

	if len(pods.Items) != 1 {
		return "", "", fmt.Errorf("unable to get pod for job")
	}

	// The reference to the pod that was spawned for the job.
	pod := &pods.Items[0]
	watcher, err := client.CoreV1().Pods(namespace).Watch(metav1.SingleObject(pod.ObjectMeta))
	if err != nil {
		return "", "", err
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
			return "", "", fmt.Errorf("unknown event type occurred: %v", event.Type)
		}
	}

	return PodExec(client, restconfig, pod, fmt.Sprintf("nsenter --target 1 --mount --uts --ipc --net --pid -- /bin/sh -c '%s'", command))
}
