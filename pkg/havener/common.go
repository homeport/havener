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
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gonvenience/wrap"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" //from https://github.com/kubernetes/client-go/issues/345
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

// MinutesToSeconds returns the amount of seconds
func MinutesToSeconds(minutes int) int {
	return minutes * 60
}

// SetConfigEnv processes the env operators of the config
// and sets them as environmental variables
func SetConfigEnv(config *Config) error {
	for _, mapItem := range config.Env {
		key, value := fmt.Sprintf("%v", mapItem.Key), fmt.Sprintf("%v", mapItem.Value)

		value, err := ProcessOperators(value)
		if err != nil {
			return wrap.Error(err, "failed to process env section")
		}
		os.Setenv(key, value)
	}

	return nil
}

// ParseConfigFile reads the havener config file, unmarshals it
// and returns the resulting Config structure.
func ParseConfigFile(path string) (*Config, error) {
	source, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, wrap.Error(err, "unable to read havener configuration")
	}

	var config Config
	if err = yaml.Unmarshal(source, &config); err != nil {
		return nil, wrap.Error(err, "failed to unmarshal havener configuration")
	}

	return &config, nil
}

// getSecretValue returns the value of the provided key of the given value.
// It returns the value as decoded string.
func getSecretValue(namespace string, secretName string, secretKey string) (string, error) {
	client, _, err := OutOfClusterAuthentication("")
	if err != nil {
		return "", wrap.Error(err, "unable to get access to cluster")
	}

	secret, err := client.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, v1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get secret: '%s' of namespace: '%s'\nerror message: %s", secretName, namespace, err.Error())
	}

	secretValue, ok := secret.Data[secretKey]
	if !ok {
		return "", fmt.Errorf("secret: '%s' has no key: '%s'", secretName, secretKey)
	}

	return string(secretValue), nil
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

	for _, entry := range cfg["clusters"].([]interface{}) {
		switch entry.(type) {
		case map[interface{}]interface{}:
			for key, value := range entry.(map[interface{}]interface{}) {
				if key == "name" {
					return value.(string), nil
				}
			}
		}
	}

	return "", fmt.Errorf("unable to determine cluster name based on Kubernetes configuration")
}

func apiCRDResourceExist(arl []*metav1.APIResourceList, crdName string) (bool, schema.GroupVersionResource) {
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
