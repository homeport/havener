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

/*
Package havener is a convenience layer to handle Containerized CF tasks on a
Kubernetes cluster, e.g. deploy, or upgrade.

It provides functions that wrap Kubernetes API calls (client-go) or Helm client
calls, or even both, to help with everyday tasks on a Kubernetes cluster that
runs Cloud Foundry in its containerized version. However, it is not limited to
this kind of workload.
*/
package havener

import (
	"github.com/gonvenience/wrap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Helpful imports:
// batchv1 "k8s.io/api/batch/v1"
// corev1 "k8s.io/api/core/v1"
// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

var onShutdownFuncs = []func(){}

// AddShutdownFunction adds a function to be called in case GracefulShutdown is
// called, for example to clean up resources.
func AddShutdownFunction(f func()) {
	onShutdownFuncs = append(onShutdownFuncs, f)
}

// GracefulShutdown brings down the havener package by going through registered
// on-shutdown functions.
func GracefulShutdown() {
	for _, f := range onShutdownFuncs {
		f()
	}
}

// Hvnr is the internal handle to consolidate required cluster access variables
type Hvnr struct {
	client      kubernetes.Interface
	restconfig  *rest.Config
	clusterName string
}

// Havener is an interface to work with a cluster through the havener
// abstraction layer
type Havener interface {
	TopDetails() (*TopDetails, error)
	ListPods(namespaces ...string) ([]*corev1.Pod, error)
	ClusterName() string
}

// NewHavener returns a new Havener handle to perform cluster actions
func NewHavener() (*Hvnr, error) {
	client, restconfig, err := OutOfClusterAuthentication("")
	if err != nil {
		return nil, wrap.Error(err, "unable to get access to cluster")
	}

	clusterName, err := ClusterName()
	if err != nil {
		return nil, wrap.Error(err, "unable to get cluster name")
	}

	return &Hvnr{
		client:      client,
		restconfig:  restconfig,
		clusterName: clusterName,
	}, nil
}

// ClusterName returns the name of the currently configured cluster
func (h *Hvnr) ClusterName() string {
	return h.clusterName
}
