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
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/* TODO Add code to purge functions to wait until the resources
   are no longer there. */

var defaultPropagationPolicy = metav1.DeletePropagationForeground

var defaultHelmDeleteTimeout = int64(15 * 60)

// PurgeHelmReleases removes all helm releases including their respective resources.
func PurgeHelmReleases(kubeClient kubernetes.Interface, helmClient *helm.Client, helmReleases ...string) error {
	errors := make(chan error, len(helmReleases))

	for _, name := range helmReleases {
		go func(helmRelease string) {
			errors <- PurgeHelmRelease(kubeClient, helmClient, helmRelease)
		}(name)
	}

	for i := 0; i < len(helmReleases); i++ {
		if err := <-errors; err != nil {
			return err
		}
	}

	return nil
}

// PurgeHelmRelease removes the given helm release including all its resources.
func PurgeHelmRelease(kubeClient kubernetes.Interface, helmClient *helm.Client, helmRelease string) error {
	statusResp, err := helmClient.ReleaseStatus(helmRelease)
	if err != nil {
		return err
	}

	errors := make(chan error, 4)

	go func(namespace string) { errors <- PurgeDeploymentsInNamespace(kubeClient, namespace) }(statusResp.Namespace)
	go func(namespace string) { errors <- PurgeStatefulSetsInNamespace(kubeClient, namespace) }(statusResp.Namespace)
	go func(namespace string) { errors <- PurgeNamespace(kubeClient, namespace) }(statusResp.Namespace)

	go func(helmRelease string) {
		_, err := helmClient.DeleteRelease(helmRelease,
			helm.DeletePurge(true),
			helm.DeleteTimeout(defaultHelmDeleteTimeout))
		errors <- err
	}(helmRelease)

	for i := 0; i < 4; i++ {
		if err := <-errors; err != nil {
			return err
		}
	}

	return nil
}

// PurgeDeploymentsInNamespace removes all deployments in the given namespace.
func PurgeDeploymentsInNamespace(kubeClient kubernetes.Interface, namespace string) error {
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
	return kubeClient.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{
		PropagationPolicy: &defaultPropagationPolicy,
	})
}
