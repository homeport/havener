// Copyright © 2019 The Havener
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
	"encoding/json"
	"sync"
	"time"

	"github.com/gonvenience/wrap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeDetails consists of the used and total values for CPU and Memory
type NodeDetails struct {
	UsedCPU     int64
	TotalCPU    int64
	UsedMemory  int64
	TotalMemory int64
}

// ContainerDetails consists of the used values for CPU and Memory of a
// pod container plus the name of the cluster node it runs on
type ContainerDetails struct {
	Nodename   string
	UsedCPU    int64
	UsedMemory int64
}

// TopDetails contains the top statistics and data of Kubernetes resources
type TopDetails struct {
	Nodes      map[string]NodeDetails
	Containers map[string]map[string]map[string]ContainerDetails
}

type nodeMetricsList struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		SelfLink string `json:"selfLink"`
	} `json:"metadata"`
	Items []struct {
		Timestamp time.Time `json:"timestamp"`
		Window    string    `json:"window"`
		Metadata  struct {
			CreationTimestamp time.Time `json:"creationTimestamp"`
			Name              string    `json:"name"`
			SelfLink          string    `json:"selfLink"`
		} `json:"metadata"`
		Usage struct {
			CPU    string `json:"cpu"`
			Memory string `json:"memory"`
		} `json:"usage"`
	} `json:"items"`
}

type podMetricsList struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		SelfLink string `json:"selfLink"`
	} `json:"metadata"`
	Items []struct {
		Timestamp time.Time `json:"timestamp"`
		Window    string    `json:"window"`
		Metadata  struct {
			CreationTimestamp time.Time `json:"creationTimestamp"`
			Name              string    `json:"name"`
			Namespace         string    `json:"namespace"`
			SelfLink          string    `json:"selfLink"`
		} `json:"metadata"`
		Containers []struct {
			Name  string `json:"name"`
			Usage struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
			} `json:"usage"`
		} `json:"containers"`
	} `json:"items"`
}

// TopDetails retrieves top statistics and data of Kubernetes resources
func (h *Havener) TopDetails() (*TopDetails, error) {
	var result = TopDetails{
		Nodes:      map[string]NodeDetails{},
		Containers: map[string]map[string]map[string]ContainerDetails{},
	}

	nodeList, err := h.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, node := range nodeList.Items {
		result.Nodes[node.Name] = NodeDetails{
			TotalCPU:    int64(node.Status.Capacity.Cpu().MilliValue()),
			TotalMemory: int64(node.Status.Capacity.Memory().Value()),
		}
	}

	pods, err := ListPods(h.clientset)
	if err != nil {
		return nil, err
	}

	for i := range pods {
		pod := pods[i]
		namespace := pod.Namespace
		nodename := pod.Spec.NodeName

		if _, ok := result.Containers[namespace]; !ok {
			result.Containers[namespace] = map[string]map[string]ContainerDetails{}
		}

		if _, ok := result.Containers[namespace][pod.Name]; !ok {
			result.Containers[namespace][pod.Name] = map[string]ContainerDetails{}
		}

		containers := []corev1.Container{}
		containers = append(containers, pod.Spec.Containers...)
		containers = append(containers, pod.Spec.InitContainers...)
		for _, container := range containers {
			result.Containers[namespace][pod.Name][container.Name] = ContainerDetails{
				Nodename: nodename,
			}
		}
	}

	var (
		wg      sync.WaitGroup
		errChan = make(chan error, 2)
	)

	wg.Add(2)

	go func() {
		nodeMetricsJSON, err := h.clientset.RESTClient().Get().AbsPath("apis/metrics.k8s.io/v1beta1/nodes").DoRaw()
		if err != nil {
			errChan <- err
			return
		}

		var nodeMetrics nodeMetricsList
		if err := json.Unmarshal(nodeMetricsJSON, &nodeMetrics); err != nil {
			errChan <- err
			return
		}

		for _, node := range nodeMetrics.Items {
			nodeDetails := result.Nodes[node.Metadata.Name]
			nodeDetails.UsedCPU = parseQuantity(node.Usage.CPU).MilliValue()
			nodeDetails.UsedMemory = parseQuantity(node.Usage.Memory).Value()
			result.Nodes[node.Metadata.Name] = nodeDetails
		}

		wg.Done()
	}()

	go func() {
		podMetricsJSON, err := h.clientset.RESTClient().Get().AbsPath("apis/metrics.k8s.io/v1beta1/pods").DoRaw()
		if err != nil {
			errChan <- err
			return
		}

		var podMetrics podMetricsList
		if err := json.Unmarshal(podMetricsJSON, &podMetrics); err != nil {
			errChan <- err
			return
		}

		for _, pod := range podMetrics.Items {
			for _, container := range pod.Containers {
				tmp := result.Containers[pod.Metadata.Namespace][pod.Metadata.Name][container.Name]
				tmp.UsedCPU = parseQuantity(container.Usage.CPU).MilliValue()
				tmp.UsedMemory = parseQuantity(container.Usage.Memory).Value()
				result.Containers[pod.Metadata.Namespace][pod.Metadata.Name][container.Name] = tmp
			}
		}

		wg.Done()
	}()

	wg.Wait()
	close(errChan)

	errors := []error{}
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return nil, wrap.Errors(errors, "failed to retrieve usage details from cluster")
	}

	return &result, nil
}

func parseQuantity(input string) *resource.Quantity {
	quantity := resource.MustParse(input)
	return &quantity
}
