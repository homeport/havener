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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// LogDirName is the subdirectory name where retrieved logs are stored
const LogDirName = "retrieved-logs"

var logFinds = []string{
	"find var/vcap/sys -type f -name '*.log*' -size +0c 2>/dev/null",
	"find var/vcap/monit -type f -size +0c 2>/dev/null",
	"find var/log -type f -size +0c 2>/dev/null",
}

var cfgFinds = []string{
	"find var/vcap/jobs -type f -size +0c 2>/dev/null",
	"find etc/nginx -type f -size +0c 2>/dev/null",
	"find opt/fissile -type f -size +0c 2>/dev/null",
}

const retrieveCommand = `/bin/sh -c '
#!/bin/sh

FILES="$(cd / && %s )"

if [ ! -z "${FILES}" ]; then
  ( cd / && ls -1Sr ${FILES} ) | while read -r FILENAME; do
    case "$(file --brief --mime-type "${FILENAME}")" in
      text/*)
        echo "${FILENAME}"
        ;;
    esac
  done | GZIP=-9 tar --create --gzip --file=- --files-from=- 2>/dev/null
fi

' 2>/dev/null`

var parallelDownloads = 32

func createDirectory(path string) error {
	if _, err := os.Stat(path); err != nil {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	return nil
}

// ClusterName returns the name of the (first) cluster defined in the Kubernetes configuration file.
func ClusterName() (string, error) {
	data, err := ioutil.ReadFile(getKubeConfig())
	if err != nil {
		return "", err
	}

	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", err
	}

	for _, entry := range cfg["clusters"].([]interface{}) {
		switch entry.(type) {
		case map[interface{}]interface{}:
			for key, value := range entry.(map[interface{}]interface{}) {
				if key == "name" {
					return value.(string), nil
				}
			}
		}
	}

	return "", fmt.Errorf("unable to determine cluster name based on Kubernetes configuration")
}

// RetrieveLogs downloads log and configuration files from some well known location of all the pods
// of all the namespaces and stored them in the local file system.
func RetrieveLogs(client kubernetes.Interface, restconfig *rest.Config, target string, includeConfigFiles bool) error {
	clusterName, err := ClusterName()
	if err != nil {
		return err
	}

	if absolute, err := filepath.Abs(target); err == nil {
		target = absolute
	}

	type task struct {
		assignment string
		pod        *corev1.Pod
		baseDir    string
	}

	tasks := make(chan *task)
	errors := []error{}

	var wg sync.WaitGroup
	for i := 0; i < parallelDownloads; i++ {
		wg.Add(1)
		go func() {
			for task := range tasks {
				switch task.assignment {
				case "cf-logs":
					for _, err := range retrieveFilesFromPod(client, restconfig, task.pod, task.baseDir, includeConfigFiles) {
						switch err {
						case io.EOF, gzip.ErrHeader, gzip.ErrChecksum:
							continue
						}

						errors = append(errors, err)
					}

				case "container-logs":
					errors = append(
						errors,
						retrieveContainerLogs(client, task.pod, task.baseDir)...,
					)
				}
			}

			wg.Done()
		}()
	}

	namespaces, err := ListNamespaces(client)
	if err != nil {
		return combineErrors(append(errors, err))
	}

	for _, namespace := range namespaces {
		listResult, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
		if err != nil {
			errors = append(errors, err)
			continue
		}

		for p := range listResult.Items {
			if listResult.Items[p].Status.Phase == corev1.PodRunning {
				tasks <- &task{
					assignment: "cf-logs",
					pod:        &listResult.Items[p],
					baseDir: filepath.Join(
						target,
						LogDirName,
						clusterName,
						namespace,
					),
				}
			}

			tasks <- &task{
				assignment: "container-logs",
				pod:        &listResult.Items[p],
				baseDir: filepath.Join(
					target,
					LogDirName,
					clusterName,
					namespace,
				),
			}
		}
	}

	close(tasks)
	wg.Wait()

	return combineErrors(errors)
}

func retrieveFilesFromPod(client kubernetes.Interface, restconfig *rest.Config, pod *corev1.Pod, baseDir string, includeConfigFiles bool) []error {
	errors := []error{}

	finds := append([]string{}, logFinds...)
	if includeConfigFiles {
		finds = append(finds, cfgFinds...)
	}

	for _, container := range pod.Spec.Containers {
		targetPath := filepath.Join(
			baseDir,
			pod.Name,
			container.Name,
		)

		read, write := io.Pipe()
		go func() {
			PodExec(
				client,
				restconfig,
				pod,
				container.Name,
				fmt.Sprintf(retrieveCommand, strings.Join(finds, "; ")),
				nil,
				write,
				os.Stderr,
				false)
			write.Close()
		}()

		if err := untar(read, targetPath); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

func untar(inputStream io.Reader, targetPath string) error {
	gzipReader, err := gzip.NewReader(inputStream)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		switch {
		case err == io.EOF:
			return nil

		case err != nil:
			return err

		case header == nil:
			continue
		}

		target := filepath.Join(targetPath, header.Name)
		switch header.Typeflag {
		case tar.TypeDir: // directory entry
			if err := createDirectory(target); err != nil {
				return err
			}

		case tar.TypeReg: // file entry
			dir, _ := filepath.Split(target)
			if err := createDirectory(dir); err != nil {
				return err
			}

			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				return err
			}

			file.Close()
		}
	}
}

func retrieveContainerLogs(client kubernetes.Interface, pod *corev1.Pod, baseDir string) []error {
	errors := []error{}

	for _, container := range pod.Spec.Containers {
		req := client.CoreV1().RESTClient().
			Get().
			Namespace(pod.GetNamespace()).
			Name(pod.Name).
			Resource("pods").
			SubResource("log").
			Param("container", container.Name)

		readCloser, err := req.Stream()
		if err != nil {
			errors = append(errors, err)
			continue
		}

		defer readCloser.Close()

		targetDir := filepath.Join(baseDir, pod.Name)
		if err := createDirectory(targetDir); err != nil {
			errors = append(errors, err)
			continue
		}

		target := filepath.Join(targetDir, container.Name+".log")
		file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(0644))
		if err != nil {
			errors = append(errors, err)
			continue
		}

		if _, err := io.Copy(file, readCloser); err != nil {
			errors = append(errors, err)
			continue
		}
	}

	return errors
}

func combineErrors(errors []error) error {
	if len(errors) > 0 {
		var buf bytes.Buffer
		for _, err := range errors {
			buf.WriteString(" - ")
			buf.WriteString(err.Error())
			buf.WriteString("\n")
		}

		return fmt.Errorf("some issues during log download:\n%s", buf.String())
	}

	return nil
}
