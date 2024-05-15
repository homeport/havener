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
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/gonvenience/text"
)

// ListNamespaces lists all namespaces
func (h *Hvnr) ListNamespaces() ([]string, error) {
	logf(Verbose, "Listing all namespaces")

	namespaceList, err := h.client.CoreV1().Namespaces().List(h.ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]string, len(namespaceList.Items))
	for i, namespace := range namespaceList.Items {
		result[i] = namespace.Name
	}

	logf(Verbose, "Found %s", text.Plural(len(result), "namespace"))
	return result, nil
}

// ListPods lists all pods in the given namespaces, if no namespace is given,
// then all namespaces currently available in the cluster will be used
func (h *Hvnr) ListPods(namespaces ...string) ([]*corev1.Pod, error) {
	logf(Verbose, "Listing all pods in %s", func() string {
		if len(namespaces) == 0 {
			return "all namespaces"
		}

		return strings.Join(namespaces, ", ")
	}())

	if len(namespaces) == 0 {
		var err error
		namespaces, err = h.ListNamespaces()
		if err != nil {
			return nil, err
		}
	}

	type list struct {
		sync.Mutex
		pods []*corev1.Pod
	}

	var min = func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}

	var result list

	var work = make(chan string, len(namespaces))
	var errs = make(chan error)

	for _, namespace := range namespaces {
		work <- namespace
	}
	close(work)

	var wg sync.WaitGroup
	for i := 0; i < min(concurrency, len(namespaces)); i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for namespace := range work {
				listResp, err := h.client.CoreV1().Pods(namespace).List(h.ctx, metav1.ListOptions{})
				if err != nil {
					errs <- err
					continue
				}

				for i := range listResp.Items {
					result.Lock()
					result.pods = append(result.pods, &listResp.Items[i])
					result.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		return nil, err
	}

	return result.pods, nil
}

// ListSecrets lists all secrets in the given namespaces, if no namespace is given,
// then all namespaces currently available in the cluster will be used
func (h *Hvnr) ListSecrets(namespaces ...string) (result []*corev1.Secret, err error) {
	if len(namespaces) == 0 {
		namespaces, err = h.ListNamespaces()
		if err != nil {
			return nil, err
		}
	}

	for _, namespace := range namespaces {
		listResp, err := h.client.CoreV1().Secrets(namespace).List(h.ctx, metav1.ListOptions{})
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
		namespaces, err = h.ListNamespaces()
		if err != nil {
			return nil, err
		}
	}

	for _, namespace := range namespaces {
		listResp, err := h.client.CoreV1().ConfigMaps(namespace).List(h.ctx, metav1.ListOptions{})
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
		list, _ := client.Resource(runtimeClassGVR).List(h.ctx, metav1.ListOptions{})
		return list.Items, nil
	}

	return nil, fmt.Errorf("desired resource %s, was not found", crdName)
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
	nodeList, err := h.client.CoreV1().Nodes().List(h.ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get list of nodes: %w", err)
	}

	return nodeList.Items, nil
}

// ListNodeNames returns a list of the names of the nodes in the cluster
func (h *Hvnr) ListNodeNames() ([]string, error) {
	nodeList, err := h.client.CoreV1().Nodes().List(h.ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get list of nodes: %w", err)
	}

	result := make([]string, len(nodeList.Items))
	for i, node := range nodeList.Items {
		result[i] = node.Name
	}

	return result, nil
}
