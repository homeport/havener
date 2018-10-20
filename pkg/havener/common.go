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
	"flag"
	"fmt"
	"os"

	"github.com/spf13/viper"
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
func OutOfClusterAuthentication() (*kubernetes.Clientset, *rest.Config, error) {
	// BuildConfigFromFlags is a helper function that builds configs from a master
	// url or a kubeconfig filepath.
	config, err := clientcmd.BuildConfigFromFlags("", getKubeConfig())
	if err != nil {
		ExitWithError("Unable to build the config from kubeconfig file", err)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)

	return clientset, config, err
}

// HomeDir returns the HOME env key
func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// ExitWithError defines a common exit log and exit code
func ExitWithError(msg string, err error) {
	fmt.Printf("Message: %s, Error: %s\n", msg, err.Error())
	os.Exit(1)
}
