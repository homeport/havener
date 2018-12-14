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
	"strings"

	"gopkg.in/yaml.v2"
	"k8s.io/helm/pkg/proto/hapi/release"
)

// ListManifestFiles breaks up the manifest string of a Helm Release to return
// a map with the template filename as the key and the unmarshaled YAML data
// as the value.
func ListManifestFiles(release *release.Release) (map[string]yaml.MapSlice, error) {
	result := make(map[string]yaml.MapSlice)

	for _, document := range strings.Split(release.GetManifest(), "---\n") {
		if document == "\n" {
			continue
		}

		if lines := strings.Split(document, "\n"); len(lines) > 0 {
			if firstLine := lines[0]; strings.HasPrefix(firstLine, "# Source: ") {
				source := strings.Replace(firstLine, "# Source: ", "", -1)

				var data yaml.MapSlice
				if err := yaml.Unmarshal([]byte(strings.Join(lines[1:], "\n")), &data); err == nil {
					result[source] = data
				}
			}
		}
	}

	return result, nil
}
