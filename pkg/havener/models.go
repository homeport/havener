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

// Config is the Havener configuration main structure contract.
type Config struct {
	Name     string            `yaml:"name"`
	Releases []Release         `yaml:"releases"`
	Env      map[string]string `yaml:"env,omitempty"`
	Before   *Task             `yaml:"before,omitempty"`
	After    *Task             `yaml:"after,omitempty"`
}

// Release is the Havener configuration Helm Release abstraction, which
// consists of the Havener specific additional details, e.g. Overrides.
type Release struct {
	ChartName      string      `yaml:"chart_name"`
	ChartNamespace string      `yaml:"chart_namespace"`
	ChartLocation  string      `yaml:"chart_location"`
	ChartVersion   int         `yaml:"chart_version"`
	Overrides      interface{} `yaml:"overrides"`
	Before         *Task       `yaml:"before,omitempty"`
	After          *Task       `yaml:"after,omitempty"`
}

// Task is the obviously relative generic definition of a list of steps to be
// evaluated. A task can be configured before and/or after a deployment.
// Furthermore, a task can also be configured before and/or after each release.
type Task []interface{}
