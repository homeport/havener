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
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"

	"github.com/pkg/errors"
)

var (
	helmBinary = "helm"
)

// UpdateHelmRelease will upgrade an existing release with provided override values
func UpdateHelmRelease(chartname string, chartPath string, valueOverrides []byte, reuseVal bool) error {
	err := VerifyHelmBinary()
	if err != nil {
		return err
	}

	_, err = RunHelmBinary("version")
	if err != nil {
		return err
	}

	helmChartPath, err := PathToHelmChart(chartPath)
	if err != nil {
		return err
	}

	overridesFile, err := GenerateConfigFile(valueOverrides)
	if err != nil {
		return err
	}

	_, err = RunHelmBinary("upgrade", chartname, helmChartPath,
		"--timeout", strconv.Itoa(MinutesToSeconds(3)),
		"--wait",
		"--reuse-values",
		"-f", overridesFile)

	if err != nil {
		return err
	}

	os.Remove(overridesFile)

	return nil
}

// DeployHelmRelease will initialize a helm in both client and server
func DeployHelmRelease(chartname string, namespace string, chartPath string, timeOut int, valueOverrides []byte) error {
	err := VerifyHelmBinary()
	if err != nil {
		return err
	}

	_, err = RunHelmBinary("version")
	if err != nil {
		return err
	}

	logf(Verbose, "Locating helm chart location...")

	helmChartPath, err := PathToHelmChart(chartPath)
	if err != nil {
		return err
	}
	logf(Verbose, "Installing release in namespace %s...", namespace)

	overridesFile, err := GenerateConfigFile(valueOverrides)
	if err != nil {
		return err
	}

	_, err = RunHelmBinary("install", helmChartPath, "--name", chartname,
		"--namespace", namespace,
		"--timeout", strconv.Itoa(MinutesToSeconds(5)),
		"--wait",
		"-f", overridesFile)

	if err != nil {
		return err
	}

	os.Remove(overridesFile)

	return nil
}

// RunHelmBinary will execute helm with the provided
// arguments.
func RunHelmBinary(args ...string) ([]byte, error) {
	cmd := exec.Command(helmBinary, args...)
	stdOutput, err := cmd.CombinedOutput()
	if err != nil {
		output := string(stdOutput)
		return nil, errors.Wrapf(err, "helm failed: %s", output)
	}
	return stdOutput, nil
}

// VerifyHelmBinary checks if the helm binary is
// available on your host.
func VerifyHelmBinary() error {
	_, err := exec.LookPath(helmBinary)
	if err != nil {
		return errors.Wrap(err, "Helm binary not found, please install it in order to proceed.")
	}
	return nil
}

// GenerateConfigFile wil dump config bytes into a tmp file
func GenerateConfigFile(valueOverrides []byte) (string, error) {
	tmpFile, err := ioutil.TempFile("", "value-overrides")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.Write(valueOverrides); err != nil {
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

// HelmReleases defines the struct when
// listing releases via helm in json format
type HelmReleases struct {
	Next     string     `json:"Next"`
	Releases []Releases `json:"Releases"`
}

// Releases defines the release metadata
type Releases struct {
	Name       string `json:"Name"`
	Revision   int    `json:"Revision"`
	Updated    string `json:"Updated"`
	Status     string `json:"Status"`
	Chart      string `json:"Chart"`
	AppVersion string `json:"AppVersion"`
	Namespace  string `json:"Namespace"`
}

// ReleaseExist returns true for an existing release
func ReleaseExist(list HelmReleases, releaseName string) bool {
	for _, release := range list.Releases {
		if release.Name == releaseName {
			return true
		}
	}
	return false
}

// GetReleaseByName returns true for an existing release
func GetReleaseByName(releaseName string) (Releases, error) {
	releasesList := HelmReleases{}

	stdOutput, err := RunHelmBinary("list", "-a", "--output", "json")
	if err != nil {
		return Releases{}, err
	}

	err = json.Unmarshal(stdOutput, &releasesList)
	if err != nil {
		return Releases{}, err
	}

	for _, release := range releasesList.Releases {
		if release.Name == releaseName {
			return release, nil
		}
	}
	return Releases{}, errors.New("Release not found")
}
