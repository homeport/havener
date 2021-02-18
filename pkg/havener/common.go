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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" //from https://github.com/kubernetes/client-go/issues/345

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeconfig *string

func getKubeConfig() string {
	if kubeconfig == nil {
		if home := HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", viper.GetString("kubeconfig"), "(optional) absolute path to the kubeconfig file")
		}
		flag.Parse()
	}

	return *kubeconfig
}

//OutOfClusterAuthentication for kube authentication from the outside
func OutOfClusterAuthentication(kubeConfig string) (*kubernetes.Clientset, *rest.Config, error) {
	logf(Verbose, "Connecting to Kubernetes cluster...")

	if kubeConfig == "" {
		kubeConfig = getKubeConfig()
	}

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

// HomeDir returns the HOME env key
func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func isSystemNamespace(namespace string) bool {
	switch namespace {
	case "default", "kube-system", "ibm-system":
		return true
	}

	return false
}

func clusterName(kubeConfig string) (string, error) {
	data, err := ioutil.ReadFile(kubeConfig)
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

func apiCRDResourceExist(arl []*v1.APIResourceList, crdName string) (bool, schema.GroupVersionResource) {
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
