package havener

import (
	"flag"
	"os"

	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" //from https://github.com/kubernetes/client-go/issues/345
	"k8s.io/client-go/tools/clientcmd"
)

// StandIn is just a stand in
func StandIn() string {
	return "standin"
}

//OutOfClusterAuthentication ...
func OutOfClusterAuthentication() (*kubernetes.Clientset, error) {
	var kubeconfig *string
	// var clientset *kubernetes.Clientset
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", viper.GetString("kubeconfig"), "(optional) absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)

	return clientset, err
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
