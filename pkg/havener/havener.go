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
	"fmt"
	"io"
	"time"

	"github.com/gonvenience/wrap"
	"golang.org/x/sync/syncmap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/describe"
)

// Helpful imports:
// batchv1 "k8s.io/api/batch/v1"
// corev1 "k8s.io/api/core/v1"
// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

var m = new(syncmap.Map)

// AddShutdownFunction adds a function to be called in case GracefulShutdown is
// called, for example to clean up resources.
func AddShutdownFunction(f func()) {
	m.Store(time.Now().String(), f)
}

// GracefulShutdown brings down the havener package by going through registered
// on-shutdown functions.
func GracefulShutdown() {
	m.Range(func(key, value interface{}) bool {
		switch f := value.(type) {
		case func():
			f()
		}

		return true
	})
}

// Hvnr is the internal handle to consolidate required cluster access variables
type Hvnr struct {
	kubeConfigPath string
	client         kubernetes.Interface
	restconfig     *rest.Config
	clusterName    string
}

// Havener is an interface to work with a cluster through the havener
// abstraction layer
type Havener interface {
	Client() kubernetes.Interface
	RESTConfig() *rest.Config
	ClusterName() string

	ListPods(namespaces ...string) ([]*corev1.Pod, error)
	ListNodes() ([]corev1.Node, error)
	ListSecrets(namespaces ...string) ([]*corev1.Secret, error)
	ListConfigMaps(namespaces ...string) ([]*corev1.ConfigMap, error)
	ListCustomResourceDefinition(string) ([]unstructured.Unstructured, error)

	TopDetails() (*TopDetails, error)
	RetrieveLogs(parallelDownloads int, target string, includeConfigFiles bool) error

	PodExec(pod *corev1.Pod, container string, command []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, tty bool) error
	NodeExec(node corev1.Node, containerImage string, timeoutSeconds int, command []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, tty bool) error
}

// NewHavenerFromFields returns a new Havener handle using the provided
// input arguments (use for unit testing only)
func NewHavenerFromFields(client kubernetes.Interface, restconfig *rest.Config, clusterName string, kubeConfigPath string) *Hvnr {
	return &Hvnr{
		client:         client,
		restconfig:     restconfig,
		clusterName:    clusterName,
		kubeConfigPath: kubeConfigPath,
	}
}

// NewHavener returns a new Havener handle to perform cluster actions
func NewHavener(kubeConfigs ...string) (*Hvnr, error) {
	var kubeConfigPath string
	switch len(kubeConfigs) {
	case 0:
		kubeConfigPath = getKubeConfig()

	case 1:
		kubeConfigPath = kubeConfigs[0]

	default:
		return nil, fmt.Errorf("multiple Kubernetes configurations are currently not supported")
	}

	client, restconfig, err := OutOfClusterAuthentication(kubeConfigPath)
	if err != nil {
		return nil, wrap.Error(err, "unable to get access to cluster")
	}

	clusterName, err := clusterName(kubeConfigPath)
	if err != nil {
		return nil, wrap.Error(err, "unable to get cluster name")
	}

	return &Hvnr{
		client:         client,
		restconfig:     restconfig,
		clusterName:    clusterName,
		kubeConfigPath: kubeConfigPath,
	}, nil
}

// ClusterName returns the name of the currently configured cluster
func (h *Hvnr) ClusterName() string {
	return h.clusterName
}

// Client returns the Kubernetes interface client for the configured cluster
func (h *Hvnr) Client() kubernetes.Interface {
	return h.client
}

// RESTConfig returns the REST config handle for the configured cluster
func (h *Hvnr) RESTConfig() *rest.Config {
	return h.restconfig
}

func (h *Hvnr) describePod(pod *corev1.Pod) (string, error) {
	describer, ok := describe.DescriberFor(schema.GroupKind{Group: corev1.GroupName, Kind: "Pod"}, h.restconfig)
	if !ok {
		return "", fmt.Errorf("failed to setup up describer for pods")
	}

	return describer.Describe(
		pod.Namespace,
		pod.Name,
		describe.DescriberSettings{
			ShowEvents: true,
		},
	)
}
