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
	func main() {
		pi := wait.NewProgressIndicator("operation in progress")

		pi.Start()
		defer pi.Done("Ok, done")

		time.Sleep(2 * time.Second)

		pi.SetText("operation *still* in progress")
		pi.SetElapsedTimeFormat("really, we wait %d seconds already")

		time.Sleep(3 * time.Second)
	}
*/
package wait

import (
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lucasb-eyer/go-colorful"

	"github.com/homeport/gonvenience/pkg/v1/bunt"
	"github.com/homeport/gonvenience/pkg/v1/term"
)

const resetLine = "\r\x1b[K"
const hideCursor = "\x1b[?25l"
const showCursor = "\x1b[?25h"
const refreshIntervalInMs = 250

var symbols = []rune(`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`)

// ProgressIndicator is a handle to a progress indicator (spinner).
type ProgressIndicator struct {
	out  io.Writer
	text string

	start   time.Time
	running uint64
	counter uint64

	elapsedTimeColor  colorful.Color
	elapsedTimeFormat string
}

// NewProgressIndicator creates a new progress indicator handle. The provided
// text is shown as long as the progress indicator runs, or if new text is
// supplied during runtime.
func NewProgressIndicator(text string) *ProgressIndicator {
	return &ProgressIndicator{
		out:               os.Stdout,
		text:              text,
		elapsedTimeColor:  bunt.DimGray,
		elapsedTimeFormat: "%d seconds",
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
	bunt.Fprint(pi.out, hideCursor)
	go func() {
		for atomic.LoadUint64(&pi.running) > 0 {
			mainContentText := bunt.Sprint(pi.text)
			elapsedTimeText := bunt.Sprintf(pi.elapsedTimeFormat, int(time.Since(pi.start).Seconds()))

			padding := term.GetTerminalWidth() - 3 -
				bunt.PlainTextLength(mainContentText) -
				bunt.PlainTextLength(elapsedTimeText)

			bunt.Fprint(pi.out,
				resetLine, " ", pi.nextSymbol(), " ",
				mainContentText,
				strings.Repeat(" ", padding),
				bunt.Colorize(elapsedTimeText, pi.elapsedTimeColor),
			)

			time.Sleep(refreshIntervalInMs * time.Millisecond)
		}
	}()

	return pi
}

// SetText updates the waiting text.
func (pi *ProgressIndicator) SetText(text string) {
	pi.text = text
}

// SetOutputWriter sets the output writer to used to print the progress
// indicator texts to, e.g. `os.Stderr` or `os.Stdout`.
func (pi *ProgressIndicator) SetOutputWriter(out io.Writer) {
	pi.out = out
}

// SetElapsedTimeColor sets the color to be used to colorize the elapsed time
// text, which is added to the progress indicator.
func (pi *ProgressIndicator) SetElapsedTimeColor(color colorful.Color) {
	pi.elapsedTimeColor = color
}

// SetElapsedTimeFormat sets a custom elapsed time format, which is used to
// create the elapsed time text. The format must contain one integer placeholder
// (%d), which is fed with the elapsed time in seconds.
func (pi *ProgressIndicator) SetElapsedTimeFormat(format string) {
	pi.elapsedTimeFormat = format
}

// Done stops the progress indicator.
func (pi *ProgressIndicator) Done(texts ...string) bool {
	if x := atomic.SwapUint64(&pi.running, 0); x > 0 {
		bunt.Fprint(pi.out, showCursor)
		bunt.Fprint(pi.out, resetLine)

		if len(texts) > 0 {
			for _, text := range texts {
				bunt.Fprint(pi.out, text)
			}

			bunt.Fprint(pi.out, "\n")
		}

		return true
	}

	return false
}

func (pi *ProgressIndicator) nextSymbol() string {
	pi.counter++
	return string(symbols[pi.counter%uint64(len(symbols))])
}
