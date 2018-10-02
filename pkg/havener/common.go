package havener

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" //from https://github.com/kubernetes/client-go/issues/345
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

// StandIn is just a stand in
func StandIn() string {
	return "standin"
}

//OutOfClusterAuthentication ...
func OutOfClusterAuthentication() (*kubernetes.Clientset, error) {
	var kubeconfig *string
	// var clientset *kubernetes.Clientset
	if home := HomeDir(); home != "" {
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

// HomeDir ...
func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// GetHelmClient ...
func GetHelmClient(kubeConfig []byte) (*helm.Client, error) {
	var tillerTunnel *kube.Tunnel

	// Create a client config that can rely on the KUBECONFIG var path, and that allow us to override the config
	// TODO atm this requires to always set KUBECONFIG, the --kubeconfig will not work here
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{CurrentContext: ""}).ClientConfig()
	if err != nil {
		panic(err.Error())
	}

	clientSet, err := OutOfClusterAuthentication()
	if err != nil {
		ExitWithError("Unable to authenticate to the cluster", err)
	}

	tillerTunnel, err = portforwarder.New("kube-system", clientSet, config)
	if err != nil {
		ExitWithError("Unable to create and initialize the tunnel", err)
	}
	tillerTunnelAddress := fmt.Sprintf("localhost:%d", tillerTunnel.Local)
	hClient := helm.NewClient(helm.Host(tillerTunnelAddress))

	return hClient, nil
}

// ListHelmReleases ...
// Based on https://github.com/helm/helm/blob/7cad59091a9451b2aa4f95aa882ea27e6b195f98/pkg/proto/hapi/services/tiller.pb.go
func ListHelmReleases(kubeConfig []byte) (*rls.ListReleasesResponse, error) {
	cfg, err := ioutil.ReadFile(viper.GetString("kubeconfig"))
	if err != nil {
		ExitWithError("Unable to read the kube config file", err)
	}
	helmClient, _ := GetHelmClient(cfg)
	var sortBy = int32(2)  //LAST_RELEASED
	var sortOrd = int32(1) //descendent

	ops := []helm.ReleaseListOption{
		helm.ReleaseListSort(sortBy),
		helm.ReleaseListOrder(sortOrd),
		helm.ReleaseListStatuses([]release.Status_Code{
			release.Status_DEPLOYED,
			release.Status_FAILED,
			release.Status_DELETING,
			release.Status_PENDING_INSTALL,
			release.Status_PENDING_UPGRADE,
			release.Status_PENDING_ROLLBACK}),
		// helm.ReleaseListNamespace("cf"),
	}
	resp, err := helmClient.ListReleases(ops...)
	if err != nil {
		ExitWithError("Unable to list the helm releases", err)
	}
	return resp, nil
}

// ExitWithError ...
func ExitWithError(msg string, err error) {
	fmt.Printf("Message: %s, Error: %s\n", msg, err.Error())
	os.Exit(1)
}
