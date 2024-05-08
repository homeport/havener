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
	"context"
	"fmt"
	"io"
	"os"
	"time"

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
	ctx            context.Context
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

// Option provides a way to set specific settings for creating the Havener setup
type Option func(*Hvnr)

// KubeConfig is an option to set a specific Kubernetes configuration path
//
// Deprecated: Use WithKubeConfigPath instead
func KubeConfig(kubeConfig string) Option {
	return func(h *Hvnr) {
		h.kubeConfigPath = kubeConfig
	}
}

func WithKubeConfigPath(kubeConfig string) Option {
	return func(h *Hvnr) { h.kubeConfigPath = kubeConfig }
}

// WithContext is an option to set the context
func WithContext(ctx context.Context) Option {
	return func(h *Hvnr) { h.ctx = ctx }
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
func NewHavener(opts ...Option) (hvnr *Hvnr, err error) {
	hvnr = &Hvnr{}
	for _, opt := range opts {
		opt(hvnr)
	}

	// Set default backgroud context if nothing is set
	if hvnr.ctx == nil {
		hvnr.ctx = context.Background()
	}

	// In case `KUBECONFIG` environment variable is set, this will take
	// precedence over command line flag or default value
	if value, ok := os.LookupEnv("KUBECONFIG"); ok {
		hvnr.kubeConfigPath = value
	}

	// In case there is no Kubernetes configuration set, use the default
	if hvnr.kubeConfigPath == "" {
		hvnr.kubeConfigPath, err = KubeConfigDefault()
		if err != nil {
			return nil, fmt.Errorf("failed to look-up default kube config: %w", err)
		}
	}

	hvnr.client, hvnr.restconfig, err = outOfClusterAuthentication(hvnr.kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to get access to cluster: %w", err)
	}

	hvnr.clusterName, err = clusterName(hvnr.kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to get cluster name: %w", err)
	}

	return hvnr, nil
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
