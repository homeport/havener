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
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/viper"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

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

// GetHelmChart() loads a chart from file. It will discover the chart encoding
// and hand off to the appropriate chart reader.
// TODO: other options for loading the chart, e.g. downloading
func GetHelmChart(path string) (requestedChart *chart.Chart, err error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		ExitWithError("File to load does not exist", err)
	}

	requestedChart, err = chartutil.Load(path)
	if err != nil {
		ExitWithError("Unable to load the chart", err)
	}
	return requestedChart, nil
}

//DeployHelmRelease will initialize a helm in both client and server
func DeployHelmRelease(chartname string, namespace string, chartPath string, valueOverrides []byte) (*rls.InstallReleaseResponse, error) {
	cfg, err := ioutil.ReadFile(viper.GetString("kubeconfig"))
	if err != nil {
		ExitWithError("Unable to read the kube config file", err)
	}

	chartRequested, err := GetHelmChart(chartPath)
	if err != nil {
		return nil, fmt.Errorf("error loading chart: %v", err)
	}

	helmClient, _ := GetHelmClient(cfg)
	if err != nil {
		return nil, err
	}

	installRelease, err := helmClient.InstallReleaseFromChart(
		chartRequested,
		namespace,
		helm.ValueOverrides(valueOverrides), // ValueOverrides specifies a list of values to include when installing.
		helm.ReleaseName(chartname),
		helm.InstallDryRun(false),
		helm.InstallReuseName(false),
		helm.InstallDisableHooks(false),
		helm.InstallTimeout(20*60),
		helm.InstallWait(true))
	if err != nil {
		ExitWithError("Error deploying chart", err)
	}
	return installRelease, err
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
