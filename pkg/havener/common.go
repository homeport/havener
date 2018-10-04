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

// StandIn is just a stand in
func StandIn() string {
	return "standin"
}

//OutOfClusterAuthentication for kube authentication from the outside
func OutOfClusterAuthentication() (*kubernetes.Clientset, *rest.Config, error) {
	var kubeconfig *string

	if home := HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", viper.GetString("kubeconfig"), "(optional) absolute path to the kubeconfig file")
	}
	flag.Parse()

	// BuildConfigFromFlags is a helper function that builds configs from a master
	// url or a kubeconfig filepath.
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
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
