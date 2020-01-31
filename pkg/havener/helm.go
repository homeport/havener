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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gonvenience/wrap"
)

var (
	helmBinary = "helm"
)

// UpdateHelmRelease will upgrade an existing release with provided override values
func (h *Hvnr) UpdateHelmRelease(chartname string, chartPath string, valueOverrides []byte, reuseVal bool) error {
	err := VerifyHelmBinary()
	if err != nil {
		return err
	}

	_, err = h.RunHelmBinary("version")
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

	_, err = h.RunHelmBinary("upgrade", chartname, helmChartPath,
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
func (h *Hvnr) DeployHelmRelease(chartname string, namespace string, chartPath string, timeOut int, valueOverrides []byte) error {
	err := VerifyHelmBinary()
	if err != nil {
		return err
	}

	_, err = h.RunHelmBinary("version")
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

	_, err = h.RunHelmBinary("install", helmChartPath, "--name", chartname,
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
func (h *Hvnr) RunHelmBinary(args ...string) ([]byte, error) {
	cmd := exec.Command(helmBinary,
		append([]string{"--kubeconfig", h.kubeConfigPath}, args...)...,
	)

	stdOutput, err := cmd.CombinedOutput()
	if err != nil {
		output := string(stdOutput)
		return nil, wrap.Errorf(err, "helm failed: %s", output)
	}

	return stdOutput, nil
}

// VerifyHelmBinary checks if the helm binary is
// available on your host.
func VerifyHelmBinary() error {
	_, err := exec.LookPath(helmBinary)
	if err != nil {
		return wrap.Error(err, "Helm binary not found, please install it in order to proceed.")
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
	Next     string        `json:"Next"`
	Releases []HelmRelease `json:"Releases"`
}

// HelmRelease defines the release metadata
type HelmRelease struct {
	Name       string `json:"Name"`
	Revision   int    `json:"Revision"`
	Updated    string `json:"Updated"`
	Status     string `json:"Status"`
	Chart      string `json:"Chart"`
	AppVersion string `json:"AppVersion"`
	Namespace  string `json:"Namespace"`
}

// ListHelmReleases lists all known Helm Releases
func (h *Hvnr) ListHelmReleases() ([]HelmRelease, error) {
	result := []HelmRelease{}

	var next string
	for {
		stdOutput, err := h.RunHelmBinary(
			"list",
			"--all",
			"--output", "json",
			"--max", "1",
			"--offset", next,
		)

		if err != nil {
			return nil, err
		}

		releasesList := HelmReleases{}
		if err := json.Unmarshal(stdOutput, &releasesList); err != nil {
			return nil, err
		}

		next = releasesList.Next
		result = append(result, releasesList.Releases...)

		if len(next) == 0 {
			break
		}
	}

	return result, nil
}

// ReleaseExist returns true for an existing release
func (h *Hvnr) ReleaseExist(list HelmReleases, releaseName string) bool {
	for _, release := range list.Releases {
		if release.Name == releaseName {
			return true
		}
	}
	return false
}

// GetReleaseByName returns true for an existing release
func (h *Hvnr) GetReleaseByName(releaseName string) (HelmRelease, error) {
	list, err := h.ListHelmReleases()
	if err != nil {
		return HelmRelease{}, err
	}

	for _, release := range list {
		if release.Name == releaseName {
			return release, nil
		}
	}

	return HelmRelease{}, fmt.Errorf("Release %s not found", releaseName)
}

// GetReleaseMessage combines a custom message with the release notes
// from the helm binary.
func (h *Hvnr) GetReleaseMessage(release Release, message string) (string, error) {
	var releaseNotes string

	result, err := h.RunHelmBinary("status", release.ChartName)
	if err != nil {
		return "", wrap.Error(err, "failed to get notes of release")
	}

	releaseNotes = substringFrom(string(result), "NOTES:")
	if len(releaseNotes) != 0 {
		message = message + "\n\n" + releaseNotes
	}

	return message, nil
}

// substringFrom gets substring from the beginning of sep string to the end
func substringFrom(value string, sep string) string {
	pos := strings.LastIndex(value, sep)
	if pos == -1 {
		return ""
	}

	return value[pos:]
}
