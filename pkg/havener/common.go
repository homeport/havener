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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	// https://github.com/kubernetes/client-go/issues/345
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	// https://github.com/homeport/havener/issues/420
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/gonvenience/wrap"
	"gopkg.in/yaml.v3"
)

// KubeConfigDefault returns assumed default locaation of the Kubernetes
// configuration, which is expected to be `$HOME/.kube/config`.
func KubeConfigDefault() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", wrap.Error(err, "unable to get home directory")
	}

	return filepath.Join(home, ".kube", "config"), nil
}

// outOfClusterAuthentication for kube authentication from the outside
func outOfClusterAuthentication(kubeConfig string) (*kubernetes.Clientset, *rest.Config, error) {
	if kubeConfig == "" {
		return nil, nil, fmt.Errorf("no kube config supplied")
	}

	logf(Verbose, "Connecting to Kubernetes cluster...")

	// BuildConfigFromFlags is a helper function that builds configs from a master
	// url or a kubeconfig filepath.
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return nil, nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)

	logf(Verbose, "Successfully connected to Kubernetes cluster.")
	return clientset, config, err
}

func isSystemNamespace(namespace string) bool {
	switch namespace {
	case "default", "kube-system", "ibm-system":
		return true
	}

	return false
}

func clusterName(kubeConfig string) (string, error) {
	data, err := os.ReadFile(kubeConfig)
	if err != nil {
		return "", err
	}

	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", err
	}

	if name, ok := cfg["current-context"]; ok {
		return fmt.Sprintf("%v", name), nil
	}

	return "", fmt.Errorf("unable to determine cluster name based on Kubernetes configuration")
}

func apiCRDResourceExist(arl []*meta.APIResourceList, crdName string) (bool, schema.GroupVersionResource) {
	for _, ar := range arl {
		// Look for a CRD based on it´s singular or
		// different short names.
		for _, r := range ar.APIResources {
			if crdName == r.SingularName || containsItem(r.ShortNames, crdName) {
				groupVersion := strings.Split(ar.GroupVersion, "/")
				return true, schema.GroupVersionResource{
					Group:    groupVersion[0],
					Version:  groupVersion[1],
					Resource: r.Name,
				}
			}
		}
	}
	return false, schema.GroupVersionResource{}
}

func containsItem(l []string, s string) bool {
	for _, a := range l {
		if a == s {
			return true
		}
	}
	return false
}
