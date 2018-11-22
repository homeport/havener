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
	"archive/zip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	git "gopkg.in/src-d/go-git.v4"
)

const chartRoomURL = "https://github.com/homeport/chartroom"
const helmChartsURL = "https://github.com/helm/charts"
const downloadedCharts = "/extracted"

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

//unzipArtifactFile will extract and copy a desired path under a zip file
//into ~/.havener/extracted
func unzipArtifactFile(filePath string, insidePath string) (string, error) {
	destinatedLocation := havenerHomeDir() + downloadedCharts

	// os.RemoveAll(destinatedLocation)

	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	for _, f := range r.Reader.File {
		zip, _ := f.Open()
		defer zip.Close()
		path := filepath.Join(destinatedLocation, f.Name)

		//only open files if the current path is a substring of the
		//desired location
		if strings.Contains(path, destinatedLocation+"/"+insidePath) {
			if f.FileInfo().IsDir() {
				os.MkdirAll(path, f.Mode())
			} else {
				f, err := os.OpenFile(
					path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
				if err != nil {
					return "", err
				}
				defer f.Close()
				_, err = io.Copy(f, zip)
				if err != nil {
					return "", err
				}
			}
		}
	}

	//if location under ~/.havener/extracted is empty, return an error
	if _, err := os.Stat(destinatedLocation); os.IsNotExist(err) {
		return "", &invalidPathInsideZip{insidePath, filePath}
	}
	return destinatedLocation + "/" + insidePath, err
}

// downloadArtifact returns a path that points to a location locally
// where an specific path of an extracted file was copied.
func downloadArtifact(artifactURL string, artifactPath string) (string, error) {
	var file string

	//grab the leftmost match of the provided path,
	//this should give you the compressed file name
	//see https://regex101.com/r/5r3n6u/1
	shellRegexArtifact := regexp.MustCompile(`[^/]+[.zip]$`)
	if matches := shellRegexArtifact.FindStringSubmatch(artifactURL); len(matches) > 0 {
		file = matches[0]
	} else {
		return "", &invalidZipFileName{artifactURL}
	}

	if _, err := os.Stat(havenerHomeDir()); os.IsNotExist(err) {
		os.MkdirAll(havenerHomeDir(), os.ModePerm)
	}

	output, err := os.Create(havenerHomeDir() + "/" + file)
	if err != nil {
		return "", err
	}
	defer output.Close()

	resp, err := http.Get(artifactURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	_, err = io.Copy(output, resp.Body)
	if err != nil {
		return "", err
	}

	localPath, err := unzipArtifactFile(havenerHomeDir()+"/"+file, artifactPath)
	if err != nil {
		return "", err
	}

	return localPath, nil
}

// validateRegex returns a boolean if the regular expression matches the input.
func validateRegex(regexExpression *regexp.Regexp, input string) bool {
	if matches := regexExpression.FindAllStringSubmatch(input, -1); len(matches) > 0 {
		return true
	}
	return false
}

// PathToHelmChart returns an absolute path to the location of the Helm Chart
// directory refereced by the input string. In case the path cannot be found
// locally in the file system, Git repositories containing curated lists of
// Helm Charts will be cloned into the Havener app directory and searched for
// the provided location, too.
func PathToHelmChart(input string) (string, error) {
	var artifactPath string
	var artifactURL string

	// Return the absolute path if the input is actually an existing local location
	if isHelmChartLocation(input) {
		return filepath.Abs(input)
	}

	// Each switch case applies an specific regular expression for an expected input.
	switch {
	// Only try to git clone a chart that follows the correct syntax.
	// syntax is <path-inside-compressed-file>@<url-to-compressed-file>, example:
	// helm/cf-opensuse@https://github.com/SUSE/scf/releases/download/2.13.3/scf-opensuse-2.13.3+cf2.7.0.0.gf95d9aed.zip
	// see https://regex101.com/r/5r3n6u/2
	case validateRegex(regexp.MustCompile(`^(.+)[@](.+)$`), input):
		matches := regexp.MustCompile(`^(.+)[@](.+)$`).FindAllStringSubmatch(input, -1)
		for _, match := range matches {
			artifactPath = match[1]
			artifactURL = match[2]
		}
		localPath, error := downloadArtifact(artifactURL, artifactPath)
		if error != nil {
			return "", error
		}
		return filepath.Abs(localPath)

	// Only try to git clone a chart that follows the correct syntax.
	// example valid paths: "tomcat/stable", "scf/2.14.1"
	// example invalid paths: "/tomcat/stable", "tomcat/stable/", "tomcat/stable/dev"
	case validateRegex(regexp.MustCompile(`^[\w\d-]+\/[\w\d._-]+$`), input):
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
	default:
		return "", &NoHelmChartFoundError{Location: input}
	}
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
