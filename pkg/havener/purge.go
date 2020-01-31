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

import (
	"fmt"
	"strconv"

	"github.com/gonvenience/wrap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

var defaultPropagationPolicy = metav1.DeletePropagationForeground

// PurgeHelmReleaseByName purges the helm release with the given name
func (h *Hvnr) PurgeHelmReleaseByName(name string) error {
	helmRelease, err := GetReleaseByName(name)
	if err != nil {
		return err
	}

	return PurgeHelmRelease(h.Client(), helmRelease, helmRelease.Name)
}

// PurgeHelmRelease removes the given helm release including all its resources.
func PurgeHelmRelease(kubeClient kubernetes.Interface, release HelmRelease, helmRelease string) error {
	if err := PurgeDeploymentsInNamespace(kubeClient, release.Namespace); err != nil {
		return err
	}

	if err := PurgeStatefulSetsInNamespace(kubeClient, release.Namespace); err != nil {
		return err
	}
	_, err := RunHelmBinary("delete",
		helmRelease,
		"--purge",
		"--timeout", strconv.Itoa(MinutesToSeconds(15)))
	if err != nil {
		return err
	}

	return PurgeNamespace(kubeClient, release.Namespace)
}

// PurgeDeploymentsInNamespace removes all deployments in the given namespace.
func PurgeDeploymentsInNamespace(kubeClient kubernetes.Interface, namespace string) error {
	// Skip known system namespaces
	if isSystemNamespace(namespace) {
		return nil
	}

	if deployments, err := ListDeploymentsInNamespace(kubeClient, namespace); err == nil {
		for _, name := range deployments {
			err := kubeClient.AppsV1beta1().Deployments(namespace).Delete(name, &metav1.DeleteOptions{
				PropagationPolicy: &defaultPropagationPolicy,
			})

			if err != nil {
				return err
			}
		}
	}

	return nil
}

// PurgeStatefulSetsInNamespace removes all stateful sets in the given namespace.
func PurgeStatefulSetsInNamespace(kubeClient kubernetes.Interface, namespace string) error {
	// Skip known system namespaces
	if isSystemNamespace(namespace) {
		return nil
	}

	if statefulsets, err := ListStatefulSetsInNamespace(kubeClient, namespace); err == nil {
		for _, name := range statefulsets {
			err := kubeClient.AppsV1beta1().StatefulSets(namespace).Delete(name, &metav1.DeleteOptions{
				PropagationPolicy: &defaultPropagationPolicy,
			})

			if err != nil {
				return err
			}
		}
	}

	return nil
}

// PurgeNamespace removes the namespace from the cluster.
func PurgeNamespace(kubeClient kubernetes.Interface, namespace string) error {
	// Skip known system namespaces
	if isSystemNamespace(namespace) {
		return nil
	}

	ns, err := kubeClient.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		// Bail out if namespace is already deleted
		switch err.(type) {
		case *errors.StatusError:
			if err.Error() == fmt.Sprintf(`namespaces "%s" not found`, namespace) {
				return nil
			}
		}

		return err
	}

	// Bail out if namespace is already in Phase Terminating
	switch ns.Status.Phase {
	case corev1.NamespaceTerminating:
		return nil
	}

	watcher, err := kubeClient.CoreV1().Namespaces().Watch(metav1.SingleObject(ns.ObjectMeta))
	if err != nil {
		return err
	}

	if err := kubeClient.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{PropagationPolicy: &defaultPropagationPolicy}); err != nil {
		return err
	}

	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Deleted:
			watcher.Stop()

		case watch.Error:
			return wrap.Errorf(err, "failed to watch namespace %s during deletion", namespace)
		}
	}

	return nil
}

// PurgePod removes the pod in the given namespace.
func PurgePod(kubeClient kubernetes.Interface, namespace string, podName string, gracePeriodSeconds int64, propagationPolicy metav1.DeletionPropagation) error {
	logf(Verbose, "Deleting pod %s in namespace %s", podName, namespace)
	return kubeClient.CoreV1().Pods(namespace).Delete(
		podName,
		&metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriodSeconds,
			PropagationPolicy:  &propagationPolicy,
		})
}
