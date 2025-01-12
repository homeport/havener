// Copyright © 2021 The Homeport Team
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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/rest"
)

// LogDirName is the subdirectory name where retrieved logs are stored
const LogDirName = "retrieved-logs"

var logFinds = []string{
	"find var/vcap/sys -type f -name '*.log' -size +0c 2>/dev/null",
	"find var/vcap/sys -type f -name '*.log.*' -size +0c 2>/dev/null",
	"find var/vcap/monit -type f -size +0c 2>/dev/null",
	"find var/log -type f -size +0c 2>/dev/null",
}

var cfgFinds = []string{
	"find var/vcap/jobs -type f -size +0c 2>/dev/null",
	"find etc/nginx -type f -size +0c 2>/dev/null",
	"find opt/fissile -type f -size +0c 2>/dev/null",
}

const retrieveScript = `
#!/bin/sh

FILES="$(cd / && %s)"

if [ ! -z "${FILES}" ]; then
  (cd / && ls -1Sr ${FILES}) | while read -r FILENAME; do
    case "${FILENAME}" in
      *log)
        echo "${FILENAME}"
        ;;

      *)
        case "$(file --brief --mime-type "${FILENAME}")" in
          text/*)
            echo "${FILENAME}"
            ;;
        esac
        ;;
    esac
  done | GZIP=-9 tar --create --gzip --file=- --files-from=- || true
fi
`

func createDirectory(path string) error {
	if _, err := os.Stat(path); err != nil {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	return nil
}

// RetrieveLogs downloads log and configuration files from some well known location of all the pods
// of all the namespaces and stored them in the local file system.
func (h *Hvnr) RetrieveLogs(parallelDownloads int, target string, includeConfigFiles bool) error {
	if absolute, err := filepath.Abs(target); err == nil {
		target = absolute
	}

	type task struct {
		assignment string
		pod        *corev1.Pod
		baseDir    string
	}

	tasks := make(chan *task)
	errs := []error{}

	var wg sync.WaitGroup
	wg.Add(parallelDownloads)
	for i := 0; i < parallelDownloads; i++ {
		go func() {
			defer wg.Done()
			for task := range tasks {
				switch task.assignment {
				case "known-logs":
					for _, err := range h.retrieveFilesFromPod(task.pod, task.baseDir, logFinds) {
						switch err {
						case io.EOF, gzip.ErrHeader, gzip.ErrChecksum:
							continue
						}

						if err != nil {
							errs = append(errs, err)
						}
					}

				case "config-files":
					for _, err := range h.retrieveFilesFromPod(task.pod, task.baseDir, cfgFinds) {
						switch err {
						case io.EOF, gzip.ErrHeader, gzip.ErrChecksum:
							continue
						}

						if err != nil {
							errs = append(errs, err)
						}
					}

				case "container-logs":
					errs = append(
						errs,
						h.retrieveContainerLogs(task.pod, task.baseDir)...,
					)

				case "describe-pods":
					if err := h.writeDescribePodToDisk(task.pod, task.baseDir); err != nil {
						errs = append(errs, err)
					}

				case "store-yaml":
					if err := h.saveDeploymentYAML(task.pod, task.baseDir); err != nil {
						errs = append(errs, err)
					}
				}
			}
		}()
	}

	pods, err := h.ListPods()
	if err != nil {
		return err
	}

	sort.Slice(pods, func(i, j int) bool {
		return pods[i].CreationTimestamp.
			After(pods[j].CreationTimestamp.Time)
	})

	var clusterName = h.ClusterName()
	for idx := range pods {
		pod := pods[idx]
		baseDir := filepath.Join(
			target,
			LogDirName,
			clusterName,
			pod.Namespace,
		)

		// Create an empty directory for the pod details
		if err := createDirectory(filepath.Join(baseDir, pod.Name)); err != nil {
			return err
		}

		// Store the describe output of the pod
		tasks <- &task{
			assignment: "describe-pods",
			pod:        pod,
			baseDir:    baseDir,
		}

		// Store the deployment YAML of the pod
		tasks <- &task{
			assignment: "store-yaml",
			pod:        pod,
			baseDir:    baseDir,
		}

		// Download the container logs
		tasks <- &task{
			assignment: "container-logs",
			pod:        pod,
			baseDir:    filepath.Join(baseDir, pod.Name, "container-logs"),
		}

		if pod.Status.Phase == corev1.PodRunning {
			// For running pods, download known log files
			tasks <- &task{
				assignment: "known-logs",
				pod:        pod,
				baseDir:    filepath.Join(baseDir, pod.Name, "container-filesystem"),
			}

			// For running pods, download configuration file
			if includeConfigFiles {
				tasks <- &task{
					assignment: "config-files",
					pod:        pod,
					baseDir:    filepath.Join(baseDir, pod.Name, "container-filesystem"),
				}
			}
		}
	}

	close(tasks)
	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("failed to retrieve logs from cluster: %w", errors.Join(errs...))
	}

	return nil
}

func (h *Hvnr) retrieveFilesFromPod(pod *corev1.Pod, baseDir string, findCommands []string) []error {
	errors := []error{}

	for _, container := range pod.Spec.Containers {
		// Ignore all container that have no shell available
		if err := h.PodExec(pod, container.Name, ExecConfig{Command: []string{"/bin/sh", "-c", "true"}, Stdout: io.Discard, Stderr: io.Discard}); err != nil {
			continue
		}

		if err := createDirectory(baseDir); err != nil {
			errors = append(errors, err)
			continue
		}

		read, write := io.Pipe()
		go func() {
			defer write.Close()
			err := h.PodExec(
				pod,
				container.Name,
				ExecConfig{
					Command: []string{"/bin/sh", "-c",
						fmt.Sprintf(
							retrieveScript,
							strings.Join(findCommands, "; "),
						)},
					Stdout: write,
				},
			)

			if err != nil {
				errors = append(errors, err)
			}
		}()

		if err := untar(read, filepath.Join(baseDir, container.Name)); err != nil {
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

func (h *Hvnr) retrieveContainerLogs(pod *corev1.Pod, baseDir string) []error {
	if err := createDirectory(baseDir); err != nil {
		return []error{err}
	}

	errors := []error{}

	streamToFile := func(req *rest.Request, filename string) error {
		readCloser, err := req.Stream(h.ctx)
		if err != nil {
			return err
		}

		defer readCloser.Close()

		file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, os.FileMode(0644))
		if err != nil {
			return err
		}

		if _, err := io.Copy(file, readCloser); err != nil {
			return err
		}

		return nil
	}

	streamContainerLogs := func(container corev1.Container, filename string) error {
		return streamToFile(
			h.client.CoreV1().RESTClient().
				Get().
				Namespace(pod.GetNamespace()).
				Name(pod.Name).
				Resource("pods").
				SubResource("log").
				Param("container", container.Name).
				Param("timestamps", strconv.FormatBool(true)),
			filename,
		)
	}

	for _, container := range pod.Spec.InitContainers {
		if err := streamContainerLogs(container, filepath.Join(baseDir, "init-"+container.Name+".log")); err != nil {
			errors = append(errors, err)
			continue
		}
	}

	if pod.Status.Phase == corev1.PodRunning {
		for _, container := range pod.Spec.Containers {
			if err := streamContainerLogs(container, filepath.Join(baseDir, "container-"+container.Name+".log")); err != nil {
				errors = append(errors, err)
				continue
			}
		}
	}

	return errors
}

func (h *Hvnr) writeDescribePodToDisk(pod *corev1.Pod, baseDir string) error {
	description, err := h.describePod(pod)
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(baseDir, pod.Name, "pod-describe.output"),
		[]byte(description),
		0644,
	)
}

func (h *Hvnr) saveDeploymentYAML(pod *corev1.Pod, baseDir string) error {
	// Whatever GroupVersionKind really is, but if it is empty the printer will
	// refuse to work, so set `Kind` and `Version` with reasonable defaults
	// knowing that this will only be pods.
	if pod.GetObjectKind().GroupVersionKind().Empty() {
		pod.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Kind:    "Pod",
			Version: "v1",
		})
	}

	var (
		printer printers.YAMLPrinter
		buf     bytes.Buffer
	)

	if err := printer.PrintObj(pod, &buf); err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(baseDir, pod.Name, "pod.yaml"),
		buf.Bytes(),
		0644,
	)
}
