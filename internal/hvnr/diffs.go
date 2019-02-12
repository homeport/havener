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

package hvnr

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/HeavyWombat/dyff/pkg/v1/dyff"
	"github.com/homeport/ytbx/pkg/v1/ytbx"
	"github.com/homeport/gonvenience/pkg/v1/bunt"
	"github.com/homeport/havener/pkg/havener"
	"gopkg.in/yaml.v2"
	"k8s.io/helm/pkg/helm"
)

// ShowHelmReleaseDiff provides a difference report using the packages of the `dyff` tool.
func ShowHelmReleaseDiff(chartname string, chartPath string, valueOverrides []byte, reuseVal bool) error {
	helmClient, err := havener.GetHelmClient(viper.GetString("kubeconfig"))
	if err != nil {
		return err
	}

	// Grab the current version of the Helm Release
	contentResp, err := helmClient.ReleaseContent(chartname)
	if err != nil {
		return err
	}

	// Perform a dry-run to get the fully rendered new version of the Helm Release
	helmChart, err := havener.GetHelmChart(chartPath)
	if err != nil {
		return err
	}

	updateResp, err := helmClient.UpdateReleaseFromChart(
		chartname,
		helmChart,
		helm.UpdateValueOverrides(valueOverrides),
		helm.ReuseValues(reuseVal),
		helm.UpgradeDryRun(true))
	if err != nil {
		return errors.Wrap(err, "failed to do the dry-run")
	}

	// By definition, from means the current/old version
	from, err := ListManifestFiles(contentResp.GetRelease())
	if err != nil {
		return err
	}

	// And, to references the next/new version
	to, err := ListManifestFiles(updateResp.GetRelease())
	if err != nil {
		return err
	}

	// Perform a standard two-way comparision based on the filename-data maps
	for filename, fromData := range from {
		if toData, ok := to[filename]; ok {
			// Both from and to have an entry for the filename
			compare(filename, fromData, toData)

		} else {
			// Only from has the file, it was deleted or renamed
			compare(filename, fromData, nil)
		}
	}

	for filename, toData := range to {
		if _, ok := from[filename]; !ok {
			// Only to has the file, it was added or was renamed
			compare(filename, nil, toData)
		}
	}

	return nil
}

func compare(filename string, from yaml.MapSlice, to yaml.MapSlice) error {
	report, err := dyff.CompareInputFiles(
		ytbx.InputFile{
			Documents: []interface{}{from},
		},
		ytbx.InputFile{
			Documents: []interface{}{to},
		})

	if err != nil {
		return err
	}

	reportWriter := &dyff.HumanReport{
		Report:            report,
		DoNotInspectCerts: false,
		NoTableStyle:      false,
		ShowBanner:        false,
	}

	if len(report.Diffs) > 0 {
		var buf bytes.Buffer

		if err := reportWriter.WriteReport(bufio.NewWriter(&buf)); err != nil {
			return errors.Wrap(err, "failed to write differences report")
		}

		bunt.Printf("Changes in CadetBlue{%s}:\n", filename)
		for _, line := range strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n") {
			bunt.Printf("│ %s\n", line)
		}
		bunt.Println()
	}

	return nil
}
