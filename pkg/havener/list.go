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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ListStatefulSetsInNamespace returns all names of stateful sets in the given namespace
func ListStatefulSetsInNamespace(client kubernetes.Interface, namespace string) ([]string, error) {
	list, err := client.AppsV1beta1().StatefulSets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]string, len(list.Items))
	for idx, item := range list.Items {
		result[idx] = item.Name
	}

	return result, nil
}

// ListDeploymentsInNamespace returns all names of deployments in the given namespace
func ListDeploymentsInNamespace(client kubernetes.Interface, namespace string) ([]string, error) {
	list, err := client.AppsV1beta1().Deployments(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]string, len(list.Items))
	for idx, item := range list.Items {
		result[idx] = item.Name
	}

	return result, nil
}

// ListNamespaces lists all namespaces
func ListNamespaces(client kubernetes.Interface) ([]string, error) {
	namespaceList, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]string, len(namespaceList.Items))
	for i, namespace := range namespaceList.Items {
		result[i] = namespace.Name
	}

	return result, nil
}

// ListSecretsInNamespace lists all secrets in a given namespace
func ListSecretsInNamespace(client kubernetes.Interface, namespace string) ([]string, error) {
	secretList, err := client.CoreV1().Secrets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]string, len(secretList.Items))
	for i, secret := range secretList.Items {
		result[i] = secret.Name
	}

	return result, nil
}

// ListPods lists all pods in all namespaces
func ListPods(client kubernetes.Interface) ([]*corev1.Pod, error) {
	namespaces, err := ListNamespaces(client)
	if err != nil {
		return nil, err
	}

	result := []*corev1.Pod{}
	for _, namespace := range namespaces {
		listResp, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for i := range listResp.Items {
			result = append(result, &listResp.Items[i])
		}
	}

	return result, nil
}

// ListNodes lists all nodes of the cluster
func ListNodes(client kubernetes.Interface) ([]string, error) {
	nodeList, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]string, len(nodeList.Items))
	for i, node := range nodeList.Items {
		result[i] = node.Name
	}

	return result, nil
}
