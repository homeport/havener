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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gonvenience/wrap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// ListStatefulSetsInNamespace returns all names of stateful sets in the given namespace
func ListStatefulSetsInNamespace(client kubernetes.Interface, namespace string) ([]string, error) {
	list, err := client.AppsV1beta1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
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
	list, err := client.AppsV1beta1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
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
	namespaceList, err := client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
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
	secretList, err := client.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]string, len(secretList.Items))
	for i, secret := range secretList.Items {
		result[i] = secret.Name
	}

	return result, nil
}

// SecretsInNamespace lists all secrets in a given namespace
func SecretsInNamespace(client kubernetes.Interface, namespace string) ([]corev1.Secret, error) {
	secretList, err := client.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return secretList.Items, nil
}

// ListPods lists all pods in all namespaces
// Deprecated: Use Havener interface function ListPods instead
func ListPods(client kubernetes.Interface) ([]*corev1.Pod, error) {
	hvnr := Hvnr{client: client}
	return hvnr.ListPods()
}

// ListPods lists all pods in the given namespaces, if no namespace is given,
// then all namespaces currently available in the cluster will be used
func (h *Hvnr) ListPods(namespaces ...string) (result []*corev1.Pod, err error) {
	if len(namespaces) == 0 {
		namespaces, err = ListNamespaces(h.client)
		if err != nil {
			return nil, err
		}
	}

	for _, namespace := range namespaces {
		listResp, err := h.client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for i := range listResp.Items {
			result = append(result, &listResp.Items[i])
		}
	}

	return result, nil
}

// ListSecrets lists all secrets in the given namespaces, if no namespace is given,
// then all namespaces currently available in the cluster will be used
func (h *Hvnr) ListSecrets(namespaces ...string) (result []*corev1.Secret, err error) {
	if len(namespaces) == 0 {
		namespaces, err = ListNamespaces(h.client)
		if err != nil {
			return nil, err
		}
	}

	for _, namespace := range namespaces {
		listResp, err := h.client.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for i := range listResp.Items {
			result = append(result, &listResp.Items[i])
		}
	}

	return result, nil
}

// ListConfigMaps lists all confimaps in the given namespaces, if no namespace is given,
// then all namespaces currently available in the cluster will be used
func (h *Hvnr) ListConfigMaps(namespaces ...string) (result []*corev1.ConfigMap, err error) {
	if len(namespaces) == 0 {
		namespaces, err = ListNamespaces(h.client)
		if err != nil {
			return nil, err
		}
	}

	for _, namespace := range namespaces {
		listResp, err := h.client.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for i := range listResp.Items {
			result = append(result, &listResp.Items[i])
		}
	}

	return result, nil
}

// ListCustomResourceDefinition lists all instances of an specific CRD
func (h *Hvnr) ListCustomResourceDefinition(crdName string) (result []unstructured.Unstructured, err error) {

	var runtimeClassGVR schema.GroupVersionResource

	_, apiResourceList, err := h.client.Discovery().ServerGroupsAndResources()
	if err != nil {
		return nil, err
	}

	crdExist, runtimeClassGVR := apiCRDResourceExist(apiResourceList, crdName)

	if crdExist {
		client, _ := dynamic.NewForConfig(h.restconfig)
		list, _ := client.Resource(runtimeClassGVR).List(context.TODO(), metav1.ListOptions{})

		for i := range list.Items {
			result = append(result, list.Items[i])
		}
		return result, nil
	}

	return result, fmt.Errorf("desired resource %s, was not found", crdName)
}

// ListNodes lists all nodes of the cluster
// Deprecated: Use Havener interface function ListNodeNames instead
func ListNodes(client kubernetes.Interface) ([]string, error) {
	nodeList, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]string, len(nodeList.Items))
	for i, node := range nodeList.Items {
		result[i] = node.Name
	}

	return result, nil
}

// ListNodes returns a list of the nodes in the cluster
func (h *Hvnr) ListNodes() ([]corev1.Node, error) {
	nodeList, err := h.client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, wrap.Error(err, "failed to get list of nodes")
	}

	return nodeList.Items, nil
}

// ListNodeNames returns a list of the names of the nodes in the cluster
func (h *Hvnr) ListNodeNames() ([]string, error) {
	nodeList, err := h.client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, wrap.Error(err, "failed to get list of nodes")
	}

	result := make([]string, len(nodeList.Items))
	for i, node := range nodeList.Items {
		result[i] = node.Name
	}

	return result, nil
}
