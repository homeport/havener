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

package environment

import (
	"encoding/json"
	"io/ioutil"
	"os/exec"

	"github.com/homeport/havener/pkg/havener"
	"github.com/pkg/errors"
)

// Environment allows to setup the required configuration
// and funcs, to run integration tests.
type Environment struct {
	KubectlBinary string
	HelmBinary    string
}

// NewEnvironment returns a new struct
func NewEnvironment() *Environment {
	return &Environment{
		KubectlBinary: "kubectl",
		HelmBinary:    "helm",
	}
}

// SetUpEnvironment will return if a cluster is not accessible
// or if the cluster main components are in a bad state.
// it will also make normal checks to see if helm binary is
// installed and the tiller server is up.
func (e *Environment) SetUpEnvironment() (err error) {
	// This should tell us is the cluster is accessible
	if err := e.RunBinary(e.KubectlBinary, "get", "cs"); err != nil {
		return errors.Wrapf(err, "Failed triggering cmd: %s. Please make sure you have access to the Kubernetes cluster.", "kubectl get cs")
	}

	//Check if helm binary is installed
	err = havener.VerifyHelmBinary()
	if err != nil {
		return errors.Wrap(err, "Helm binary was not found.")
	}

	//Check if tiller is installed
	err = e.RunBinary(e.HelmBinary, "version")
	if err != nil {
		return errors.Wrapf(err, "Failed triggering cmd: %s. Please make sure tiller is up and running.", "helm version")
	}
	return
}

// RunBinary will execute the desire command
func (e *Environment) RunBinary(binaryName string, args ...string) error {
	cmd := exec.Command(binaryName, args...)
	stdOutput, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "%s cmd, failed with the following error: %s", cmd.Args, string(stdOutput))
	}
	return nil
}

// RunBinaryWithStdOutput with an specified list of commands
func (e *Environment) RunBinaryWithStdOutput(binaryName string, args ...string) ([]byte, error) {
	cmd := exec.Command(binaryName, args...)
	stdOutput, err := cmd.CombinedOutput()
	if err != nil {
		return []byte{}, errors.Wrapf(err, "%s cmd, failed with the following error: %s", cmd.Args, string(stdOutput))
	}
	return stdOutput, nil
}

// DeleteAllReleases remove all
func (e *Environment) DeleteAllReleases() error {
	releasesList := havener.HelmReleases{}
	stdOutput, err := e.RunBinaryWithStdOutput(e.HelmBinary, "list", "--output", "json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(stdOutput, &releasesList)
	if err != nil {
		return err
	}
	for _, release := range releasesList.Releases {
		err := e.RunBinary(e.HelmBinary, "delete", release.Name, "--purge")
		if err != nil {
			return err
		}
	}
	return nil
}

// GenerateConfigFile wil dump config bytes into a tmp file
func GenerateConfigFile(yamlConfigInBytes []byte) (string, error) {
	tmpFile, err := ioutil.TempFile("", "config-file")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.Write(yamlConfigInBytes); err != nil {
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}
