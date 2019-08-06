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

package cmd

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/gonvenience/bunt"
	colorful "github.com/lucasb-eyer/go-colorful"
)

// OutputMsg is a message with additional context like time, an origin (name),
// and stream (for example stdout).
type OutputMsg struct {
	Timestamp time.Time
	Stream    string
	Origin    string
	Message   string
}

func chanWriter(stream string, origin string, c chan OutputMsg) io.Writer {
	r, w := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			c <- OutputMsg{
				Timestamp: time.Now(),
				Stream:    stream,
				Origin:    origin,
				Message:   scanner.Text(),
			}
		}
	}()

	return w
}

// PrintOutputMessage reads from the given output message channel and prints the
// respective messages without any buffering or sorting.
func PrintOutputMessage(messages chan OutputMsg, items int) error {
	var (
		colors        = bunt.RandomTerminalFriendlyColors(items)
		originCounter = 0
		originColors  = map[string]colorful.Color{}
	)

	for msg := range messages {
		if _, ok := originColors[msg.Origin]; !ok {
			originColors[msg.Origin] = colors[originCounter]
			originCounter++
		}

		printMessage(originColors[msg.Origin], msg)
	}

	return nil
}

// PrintOutputMessageAsBlock reads from the given output message channel and
// buffers the input until the channel is closed. Once closed, it prints the
// output messages sorted by the origin.
func PrintOutputMessageAsBlock(messages chan OutputMsg, items int) {
	var (
		colors = bunt.RandomTerminalFriendlyColors(items)
		data   = map[string][]OutputMsg{}
		keys   = []string{}
	)

	// Fully read the input channel and store the output messages indexed by the
	// origin in a separate map
	for msg := range messages {
		if _, ok := data[msg.Origin]; !ok {
			data[msg.Origin] = []OutputMsg{}
			keys = append(keys, msg.Origin)
		}

		data[msg.Origin] = append(data[msg.Origin], msg)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	var i = 0
	for _, key := range keys {
		for _, msg := range data[key] {
			printMessage(colors[i], msg)
		}
		i++
	}
}

func printMessage(color colorful.Color, msg OutputMsg) {
	var (
		prefix = bunt.Style(
			msg.Origin,
			bunt.Foreground(color),
			bunt.Bold(),
		)
		date      = bunt.Sprintf("DimGray{%s}", GetHumanReadableTime(msg.Timestamp))
		separator = bunt.Sprintf("DimGray{%s}", "│")
		message   = msg.Message
	)

	// Add a red tint to all error stream messages
	if msg.Stream == "StdErr" {
		message = bunt.Style(message,
			bunt.Blend(),
			bunt.Foreground(bunt.Red),
		)
	}

	fmt.Printf("%s %s %s %s\n", date, prefix, separator, message)
}

// GetHumanReadableTime returns the time as string in the format: hh:mm:ss from a date object.
func GetHumanReadableTime(date time.Time) string {
	return fmt.Sprintf("%02d:%02d:%02d", date.Hour(), date.Minute(), date.Second())
}
