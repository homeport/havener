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
	"os"

	"github.com/spf13/viper"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/cmd/helm/installer"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

// Hardcode tiller image because version.Version is overwritten when build helm release
// https://github.com/helm/helm/blob/99199c975236430fdf7599c69a956c6eb73b44e9/versioning.mk#L16-L19
const (
	ImageSpec = "gcr.io/kubernetes-helm/tiller:v2.10.0"
)

// ListHelmReleases returns a list of releases
// Based on https://github.com/helm/helm/blob/7cad59091a9451b2aa4f95aa882ea27e6b195f98/pkg/proto/hapi/services/tiller.pb.go
func ListHelmReleases() (*rls.ListReleasesResponse, error) {
	cfg := viper.GetString("kubeconfig")

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
		return nil, err
	}

	return resp, nil
}

// GetHelmChart loads a chart from file. It will discover the chart encoding
// and hand off to the appropriate chart reader.
// TODO: other options for loading the chart, e.g. downloading
func GetHelmChart(path string) (requestedChart *chart.Chart, err error) {
	helmChartPath, err := PathToHelmChart(path)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(helmChartPath); os.IsNotExist(err) {
		return nil, err
	}

	return chartutil.Load(helmChartPath)
}

//UpdateHelmRelease will upgrade an existing release with provided override values
func UpdateHelmRelease(chartname string, chartPath string, valueOverrides []byte, reuseVal bool) (*rls.UpdateReleaseResponse, error) {
	cfg := viper.GetString("kubeconfig")

	helmChartPath, err := PathToHelmChart(chartPath)
	if err != nil {
		return nil, err
	}

	chartRequested, err := GetHelmChart(helmChartPath)
	if err != nil {
		return nil, fmt.Errorf("error loading chart: %v", err)
	}

	helmClient, _ := GetHelmClient(cfg)
	if err != nil {
		return nil, err
	}

	return helmClient.UpdateReleaseFromChart(
		chartname,
		chartRequested,
		helm.UpdateValueOverrides(valueOverrides),
		helm.UpgradeDryRun(false),
		helm.UpgradeTimeout(30*60),
		helm.ReuseValues(reuseVal),
	)
}

//DeployHelmRelease will initialize a helm in both client and server
func DeployHelmRelease(chartname string, namespace string, chartPath string, timeOut int, valueOverrides []byte) (*rls.InstallReleaseResponse, error) {

	VerboseMessage("Reading kube config file...")

	cfg := viper.GetString("kubeconfig")

	VerboseMessage("Locating helm chart location...")

	helmChartPath, err := PathToHelmChart(chartPath)
	if err != nil {
		return nil, err
	}

	VerboseMessage("Loading chart in namespace %s...", namespace)

	chartRequested, err := GetHelmChart(helmChartPath)
	if err != nil {
		return nil, fmt.Errorf("error loading chart: %v", err)
	}

	VerboseMessage("Getting helm client...")

	helmClient, _ := GetHelmClient(cfg)
	if err != nil {
		return nil, err
	}

	VerboseMessage("Installing release in namespace %s...", namespace)

	//cast timeout to int64, as required by InstallReleaseFromChart
	timeOutInt64 := int64(MinutesToSeconds(timeOut))

	return helmClient.InstallReleaseFromChart(
		chartRequested,
		namespace,
		helm.ValueOverrides(valueOverrides), // ValueOverrides specifies a list of values to include when installing.
		helm.ReleaseName(chartname),
		helm.InstallDryRun(false),
		helm.InstallReuseName(false),
		helm.InstallDisableHooks(false),
		helm.InstallTimeout(timeOutInt64),
		helm.InstallWait(true),
	)
}

// GetHelmClient creates a new client for the Helm-Tiller protocol
func GetHelmClient(kubeConfig string) (*helm.Client, error) {
	var tillerTunnel *kube.Tunnel

	clientSet, config, err := OutOfClusterAuthentication(kubeConfig)
	if err != nil {
		return nil, err
	}

	err = InitTiller("kube-system", clientSet)
	if err != nil {
		return nil, err
	}

	tillerTunnel, err = portforwarder.New("kube-system", clientSet, config)
	if err != nil {
		return nil, err
	}

	tillerTunnelAddress := fmt.Sprintf("localhost:%d", tillerTunnel.Local)
	hClient := helm.NewClient(helm.Host(tillerTunnelAddress))

	return hClient, nil
}

// InitTiller installs Tiller or upgrade if needed
func InitTiller(namespace string, clientSet kubernetes.Interface) error {
	VerboseMessage("Installing Tiller in namespace %s...", namespace)
	if err := installer.Install(clientSet, &installer.Options{Namespace: namespace, ImageSpec: ImageSpec}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("error when installing: %s", err)
		}
		if err := installer.Upgrade(clientSet, &installer.Options{Namespace: namespace, ForceUpgrade: true, ImageSpec: ImageSpec}); err != nil {
			return fmt.Errorf("error when upgrading: %s", err)
		}

		VerboseMessage("Tiller has been upgraded to the current version.")

	} else {
		VerboseMessage("Tiller has been installed.")
	}

	return nil
}
