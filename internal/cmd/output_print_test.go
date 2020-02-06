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

package cmd_test

import (
	"bytes"
	"io"
	"os"
	"time"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/term"

	. "github.com/homeport/havener/internal/cmd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var exampleOutput = []OutputMsg{
	{
		Stream:    "StdOut",
		Origin:    "Prefix1",
		Message:   "bin    etc    lib    mnt    proc   run    srv    tmp    var",
		Timestamp: time.Date(2019, time.July, 11, 8, 20, 16, 333, time.UTC),
	},
	{
		Stream:    "StdOut",
		Origin:    "Prefix2",
		Message:   "Successfully read content of directory!",
		Timestamp: time.Date(2019, time.July, 11, 8, 20, 15, 995, time.UTC),
	},
	{
		Stream:    "StdOut",
		Origin:    "Prefix1",
		Message:   "Successfully read content of directory!",
		Timestamp: time.Date(2019, time.July, 11, 8, 20, 15, 670, time.UTC),
	},
	{
		Stream:    "StdOut",
		Origin:    "Prefix2",
		Message:   "bin    etc",
		Timestamp: time.Date(2019, time.July, 11, 8, 20, 16, 334, time.UTC),
	},
	{
		Stream:    "StdOut",
		Origin:    "Prefix1",
		Message:   "dev    home   media  opt    root   sbin   sys    usr",
		Timestamp: time.Date(2019, time.July, 11, 8, 20, 16, 0, time.UTC),
	},
}

var _ = Describe("Output message printing", func() {
	var captureStdout = func(f func()) string {
		r, w, err := os.Pipe()
		Expect(err).ToNot(HaveOccurred())

		tmp := os.Stdout
		defer func() {
			os.Stdout = tmp
		}()

		os.Stdout = w
		f()
		w.Close()

		var buf bytes.Buffer
		io.Copy(&buf, r)

		return buf.String()
	}

	BeforeEach(func() {
		bunt.ColorSetting = bunt.OFF
		bunt.TrueColorSetting = bunt.OFF
		term.FixedTerminalWidth = 120
		term.FixedTerminalHeight = 40
	})

	AfterEach(func() {
		bunt.ColorSetting = bunt.AUTO
		bunt.TrueColorSetting = bunt.AUTO
		term.FixedTerminalWidth = -1
		term.FixedTerminalHeight = -1
	})

	It("should add leading zeros to single-digit time units", func() {
		timeString := GetHumanReadableTime(time.Date(2019, time.July, 11, 8, 15, 0, 500, time.UTC))
		Expect(timeString).To(BeEquivalentTo("08:15:00"))
	})

	It("should print an output message channel nicely", func() {
		exampleChannel := make(chan OutputMsg)
		go func() {
			for i := range exampleOutput {
				exampleChannel <- exampleOutput[i]
			}

			close(exampleChannel)
		}()

		actual := captureStdout(func() {
			PrintOutputMessage(exampleChannel)
		})

		expected := "08:20:16 Prefix1 │ bin    etc    lib    mnt    proc   run    srv    tmp    var\n" +
			"08:20:15 Prefix2 │ Successfully read content of directory!\n" +
			"08:20:15 Prefix1 │ Successfully read content of directory!\n" +
			"08:20:16 Prefix2 │ bin    etc\n" +
			"08:20:16 Prefix1 │ dev    home   media  opt    root   sbin   sys    usr\n"

		Expect(actual).To(BeEquivalentTo(expected))
	})

	It("should print an output message channel nicely sorted by Origin", func() {
		exampleChannel := make(chan OutputMsg)
		go func() {
			for i := range exampleOutput {
				exampleChannel <- exampleOutput[i]
			}

			close(exampleChannel)
		}()

		actual := captureStdout(func() {
			PrintOutputMessageAsBlock(exampleChannel)
		})

		expected := "08:20:16 Prefix1 │ bin    etc    lib    mnt    proc   run    srv    tmp    var\n" +
			"08:20:15 Prefix1 │ Successfully read content of directory!\n" +
			"08:20:16 Prefix1 │ dev    home   media  opt    root   sbin   sys    usr\n" +
			"08:20:15 Prefix2 │ Successfully read content of directory!\n" +
			"08:20:16 Prefix2 │ bin    etc\n"

		Expect(actual).To(BeEquivalentTo(expected))
	})
})
