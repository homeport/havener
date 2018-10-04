package havener

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" //from https://github.com/kubernetes/client-go/issues/345
	"k8s.io/client-go/rest"
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

// GetHelmClient creates a new client for the Helm-Tiller protocol
func GetHelmClient(kubeConfig []byte) (*helm.Client, error) {
	var tillerTunnel *kube.Tunnel

	clientSet, config, err := OutOfClusterAuthentication()
	if err != nil {
		ExitWithError("Unable to authenticate to the cluster from the outside", err)
	}

	tillerTunnel, err = portforwarder.New("kube-system", clientSet, config)
	if err != nil {
		ExitWithError("Unable to create and initialize the tunnel", err)
	}
	tillerTunnelAddress := fmt.Sprintf("localhost:%d", tillerTunnel.Local)
	hClient := helm.NewClient(helm.Host(tillerTunnelAddress))

	return hClient, nil
}

// ListHelmReleases returns a list of releases
// Based on https://github.com/helm/helm/blob/7cad59091a9451b2aa4f95aa882ea27e6b195f98/pkg/proto/hapi/services/tiller.pb.go
func ListHelmReleases() (*rls.ListReleasesResponse, error) {
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

// ExitWithError defines a common exit log and exit code
func ExitWithError(msg string, err error) {
	fmt.Printf("Message: %s, Error: %s\n", msg, err.Error())
	os.Exit(1)
}
