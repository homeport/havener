// Copyright © 2019 The Havener
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
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gonvenience/bunt"
	"github.com/homeport/havener/pkg/havener"
	colorful "github.com/lucasb-eyer/go-colorful"
)

// SortDistributedExecOutput sorts a slice of ExecMessages.
func SortDistributedExecOutput(output []*havener.ExecMessage, items int, blockFlag bool) []*havener.ExecMessage {
	switch {
	case blockFlag:
		sort.Slice(output, func(i, j int) bool {
			if output[i].Prefix < output[j].Prefix {
				return true
			}
			if output[i].Prefix > output[j].Prefix {
				return false
			}
			return output[i].Date.Before(output[j].Date)
		})
	default:
		sort.Slice(output, func(i, j int) bool { return output[i].Date.Before(output[j].Date) })
	}

	return output
}

// FormatDistributedExecOutput formats and merges a slice of ExecMessages to a single string.
// It also assigns a color to each prefex and therefor needs the overall number of prefixes
// (items) as parameter.
func FormatDistributedExecOutput(output []*havener.ExecMessage, items int) (string, error) {
	colors, err := colorful.HappyPalette(items)
	if err != nil {
		return "", err
	}
	colorDictionary := map[string]colorful.Color{}

	lines := []string{}
	colorIndex := 0
	for _, message := range output {
		var color colorful.Color

		if dictColor, ok := colorDictionary[message.Prefix]; ok {
			color = dictColor
		} else {
			color = colors[colorIndex]
			colorDictionary[message.Prefix] = color
			colorIndex++
		}

		var (
			prefix string = bunt.Style(
				fmt.Sprintf("%s", message.Prefix),
				bunt.Foreground(color),
				bunt.Bold(),
			)
			date        string = bunt.Sprintf("DimGray{%s}", getHumanReadableTime(message.Date))
			separator   string = bunt.Sprintf("DimGray{%s}", "|")
			messageText string = message.Text
		)

		lines = append(lines, fmt.Sprintf("%s %s %s %s", date, prefix, separator, messageText))
	}

	return strings.Join(lines, "\r\n"), nil
}

// getHumanReadableTime returns the time as string in the format: hh:mm:ss from a date object.
func getHumanReadableTime(date time.Time) string {
	return fmt.Sprintf("%02d:%02d:%02d", date.Hour(), date.Minute(), date.Second())
}
