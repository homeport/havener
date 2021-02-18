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

import "fmt"

// NoHelmChartFoundError means that no Helm Charts were found at a given location
type NoHelmChartFoundError struct {
	Location string
}

// invalidPathInsideZip means that the path does not exists in the zip file
type invalidPathInsideZip struct {
	fileName string
	path     string
}

// invalidZipFileName means that the file is not of the form <file-name>.zip
type invalidZipFileName struct {
	fileName string
}

func (e *NoHelmChartFoundError) Error() string {
	return fmt.Sprintf("unable to determine Helm Chart location of '%s', it is not a local path, nor is it defined in %s or %s",
		e.Location,
		chartRoomURL,
		helmChartsURL)
}

func (e *invalidPathInsideZip) Error() string {
	return fmt.Sprintf("Error: The provided path: %v, does not exist under the %v file", e.fileName, e.path)
}

func (e *invalidZipFileName) Error() string {
	return fmt.Sprintf("Error: The provided file under the %v URL, is not a valid zip file.", e.fileName)
}
