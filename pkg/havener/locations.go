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
	"os"
	"path/filepath"

	git "gopkg.in/src-d/go-git.v4"
)

const chartRoomURL = "https://github.com/homeport/chartroom"
const helmChartsURL = "https://github.com/helm/charts"

func havenerHomeDir() string {
	return HomeDir() + "/.havener"
}

func chartStoreLocation() string {
	return havenerHomeDir() + "/chartstore"
}

func chartRoomLocation() string {
	return chartStoreLocation() + "/chartroom"
}

func helmChartsLocation() string {
	return chartStoreLocation() + "/helmcharts"
}

func updateLocalChartStore() error {
	if _, err := os.Stat(chartStoreLocation()); os.IsNotExist(err) {
		os.MkdirAll(chartStoreLocation(), os.ModePerm)
	}

	if err := cloneOrPull(chartRoomLocation(), chartRoomURL); err != nil {
		return err
	}

	if err := cloneOrPull(helmChartsLocation(), helmChartsURL); err != nil {
		return err
	}

	return nil
}

func cloneOrPull(location string, url string) error {
	if _, err := os.Stat(location); os.IsNotExist(err) {
		if _, err := git.PlainClone(location, false, &git.CloneOptions{URL: url}); err != nil {
			return err
		}

	} else {
		r, err := git.PlainOpen(location)
		if err != nil {
			return err
		}

		w, err := r.Worktree()
		if err != nil {
			return err
		}

		err = w.Pull(&git.PullOptions{RemoteName: "origin"})
		if err != nil && err.Error() != "already up-to-date" {
			return err
		}
	}

	return nil
}

// pathToHelmChart returns an absolute path to the location of the Helm Chart
// directory refereced by the input string. In case the path cannot be found
// locally in the file system, Git repositories containing curated lists of
// Helm Charts will be cloned into the Havener app directory and searched for
// the provided location, too.
func pathToHelmChart(input string) (string, error) {
	// Return the absolute path if the input is actually an existing local location
	if isHelmChartLocation(input) {
		return filepath.Abs(input)
	}

	// Update Git repos with curated applications for Kubernetes
	if err := updateLocalChartStore(); err != nil {
		return "", err
	}

	// Check whether the input matches a Helm Chart in one of the curated lists of charts
	for _, candidate := range []string{chartRoomLocation() + "/charts/" + input, helmChartsLocation() + "/" + input} {
		if isHelmChartLocation(candidate) {
			return filepath.Abs(candidate)
		}
	}

	return "", &NoHelmChartFoundError{Location: input}
}

func isHelmChartLocation(path string) bool {
	// Make sure the candidate directory contains Chart.yaml, values.yaml and a templates directory
	if stat, err := os.Stat(path); err == nil && stat.IsDir() {
		if stat, err := os.Stat(path + "/Chart.yaml"); !os.IsNotExist(err) && !stat.IsDir() {
			if stat, err := os.Stat(path + "/values.yaml"); !os.IsNotExist(err) && !stat.IsDir() {
				if stat, err := os.Stat(path + "/templates"); !os.IsNotExist(err) && stat.IsDir() {
					return true
				}
			}
		}
	}

	return false
}
