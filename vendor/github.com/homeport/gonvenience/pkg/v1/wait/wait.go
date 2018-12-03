// Copyright © 2018 The Homeport Team
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

/*
Package wait contains convenience functions to create a progress indicator for
CLI applications, also know as a spinner. It is basically a text and a rapidly
changing symbol that provides feedback to the user that something is still
running even though there is no information how long it will continue to run.

Example:
	package main

	import (
		"time"

		"github.com/homeport/gonvenience/pkg/v1/wait"
	)

	func main() {
		pi := wait.NewProgressIndicator("operation in progress")

		pi.SetTimeout(10 * time.Second)
		pi.Start()

		time.Sleep(5 * time.Second)

		pi.SetText("operation *still* in progress")

		time.Sleep(5 * time.Second)

		pi.Done("Ok, done")
	}
*/
package wait

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lucasb-eyer/go-colorful"

	"github.com/homeport/gonvenience/pkg/v1/bunt"
	"github.com/homeport/gonvenience/pkg/v1/term"
)

const resetLine = "\r\x1b[K"
const refreshIntervalInMs = 250

var symbols = []rune(`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`)

var defaultElapsedTimeColor = bunt.DimGray

// ProgressIndicator is a handle to a progress indicator (spinner).
type ProgressIndicator struct {
	out    io.Writer
	format string
	args   []interface{}

	spin bool

	start   time.Time
	running uint64
	counter uint64

	timeout time.Duration

	timeInfoText func(time.Duration) (string, colorful.Color)
}

// NewProgressIndicator creates a new progress indicator handle. The provided
// text is shown as long as the progress indicator runs, or if new text is
// supplied during runtime.
func NewProgressIndicator(format string, args ...interface{}) *ProgressIndicator {
	return &ProgressIndicator{
		out:          os.Stdout,
		format:       format,
		args:         args,
		spin:         term.IsTerminal() && !term.IsDumbTerminal(),
		timeout:      0 * time.Second,
		timeInfoText: TimeInfoText,
	}
}

// Start starts the progress indicator. If it is already started, the this
// function returns immediately.
func (pi *ProgressIndicator) Start() *ProgressIndicator {
	// No-op, in case it is already running
	if atomic.LoadUint64(&pi.running) > 0 {
		return pi
	}

	pi.start = time.Now()
	atomic.StoreUint64(&pi.running, 1)

	if pi.spin {
		term.HideCursor()

		go func() {
			for atomic.LoadUint64(&pi.running) > 0 {
				elapsedTime := time.Since(pi.start)

				// Timeout reached, stopping the progress indicator
				if pi.timeout > time.Nanosecond && elapsedTime > pi.timeout {
					pi.Done("timeout occurred")
					break
				}

				mainContentText := removeLineFeeds(bunt.Sprintf(pi.format, pi.args...))
				elapsedTimeText, elapsedTimeColor := pi.timeInfoText(elapsedTime)

				padding := term.GetTerminalWidth() - 3 -
					bunt.PlainTextLength(mainContentText) -
					bunt.PlainTextLength(elapsedTimeText)

				// In case a timeout is set, smoothly blend the time info text color from
				// the provided color into red depending on how much time is left
				if pi.timeout > time.Nanosecond {
					// Use smooth curved gradient: http://fooplot.com/?lang=en#W3sidHlwZSI6MCwiZXEiOiIoMS1jb3MoeF4yKjMuMTQxNSkpLzIiLCJjb2xvciI6IiMwMDAwMDAifSx7InR5cGUiOjEwMDAsIndpbmRvdyI6WyIwIiwiMSIsIjAiLCIxIl19XQ--
					blendFactor := 0.5 * (1.0 - math.Cos(math.Pow(elapsedTime.Seconds()/pi.timeout.Seconds(), 2)*math.Pi))
					elapsedTimeColor = elapsedTimeColor.BlendLab(bunt.Red, blendFactor)
				}

				bunt.Fprint(pi.out,
					resetLine, " ", pi.nextSymbol(), " ",
					mainContentText,
					strings.Repeat(" ", padding),
					bunt.Colorize(elapsedTimeText, elapsedTimeColor),
				)

				time.Sleep(refreshIntervalInMs * time.Millisecond)
			}
		}()

	} else {
		bunt.Fprintf(pi.out, pi.format, pi.args...)
		bunt.Fprintln(pi.out)
	}

	return pi
}

// Stop stops the progress indicator by clearing the line one last time
func (pi *ProgressIndicator) Stop() bool {
	if x := atomic.SwapUint64(&pi.running, 0); x > 0 {
		if pi.spin {
			term.ShowCursor()
			bunt.Fprint(pi.out, resetLine)
		}

		return true
	}

	return false
}

// SetText updates the waiting text.
func (pi *ProgressIndicator) SetText(format string, args ...interface{}) {
	if bunt.Sprintf(format, args...) != bunt.Sprintf(pi.format, pi.args...) {
		pi.format = format
		pi.args = args

		if !pi.spin {
			bunt.Fprintf(pi.out, pi.format, pi.args...)
			bunt.Fprintln(pi.out)
		}
	}
}

// SetOutputWriter sets the output writer to used to print the progress
// indicator texts to, e.g. `os.Stderr` or `os.Stdout`.
func (pi *ProgressIndicator) SetOutputWriter(out io.Writer) {
	pi.out = out
}

// SetTimeout specifies that the progress indicator will timeout after the
// provided duration. A timeout duration lower than one nanosecond means that
// there is no timeout.
func (pi *ProgressIndicator) SetTimeout(timeout time.Duration) {
	pi.timeout = timeout
}

// SetTimeInfoTextFunc sets a custom time info text function that is called to
// create the string and the color to be used on the far right side of the
// progress indicator.
func (pi *ProgressIndicator) SetTimeInfoTextFunc(f func(time.Duration) (string, colorful.Color)) {
	pi.timeInfoText = f
}

// Done stops the progress indicator.
func (pi *ProgressIndicator) Done(format string, args ...interface{}) bool {
	defer func() {
		bunt.Fprintf(pi.out, format, args...)
		bunt.Fprintln(pi.out)
	}()

	return pi.Stop()
}

func (pi *ProgressIndicator) nextSymbol() string {
	pi.counter++
	return string(symbols[pi.counter%uint64(len(symbols))])
}

// TimeInfoText is the default implementation for the time information text on
// the far right side of the progress indicator line.
func TimeInfoText(elapsedTime time.Duration) (string, colorful.Color) {
	return humanReadableDuration(elapsedTime), defaultElapsedTimeColor
}

func humanReadableDuration(duration time.Duration) string {
	if duration < time.Second {
		return "less than a second"
	}

	seconds := int(duration.Seconds())
	minutes := 0
	hours := 0

	if seconds >= 60 {
		minutes = seconds / 60
		seconds = seconds % 60

		if minutes >= 60 {
			hours = minutes / 60
			minutes = minutes % 60
		}
	}

	parts := []string{}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d h", hours))
	}

	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d min", minutes))
	}

	if seconds > 0 {
		parts = append(parts, fmt.Sprintf("%d sec", seconds))
	}

	return strings.Join(parts, " ")
}

func removeLineFeeds(input string) string {
	return strings.Replace(input, "\n", " ", -1)
}
