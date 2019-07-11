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
	"time"

	"github.com/gonvenience/bunt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/homeport/havener/pkg/havener"
)

var exampleOutput = []*havener.ExecMessage{
	&havener.ExecMessage{
		Prefix: "Prefix1",
		Text:   "bin    etc    lib    mnt    proc   run    srv    tmp    var",
		Date:   time.Date(2019, time.July, 11, 8, 20, 16, 333, time.UTC),
	},
	&havener.ExecMessage{
		Prefix: "Prefix2",
		Text:   "Successfully read content of directory!",
		Date:   time.Date(2019, time.July, 11, 8, 20, 15, 995, time.UTC),
	},
	&havener.ExecMessage{
		Prefix: "Prefix1",
		Text:   "Successfully read content of directory!",
		Date:   time.Date(2019, time.July, 11, 8, 20, 15, 670, time.UTC),
	},
	&havener.ExecMessage{
		Prefix: "Prefix2",
		Text:   "bin    etc",
		Date:   time.Date(2019, time.July, 11, 8, 20, 16, 334, time.UTC),
	},
	&havener.ExecMessage{
		Prefix: "Prefix1",
		Text:   "dev    home   media  opt    root   sbin   sys    usr",
		Date:   time.Date(2019, time.July, 11, 8, 20, 16, 0, time.UTC),
	},
}

var _ = Describe("Distributed", func() {
	It("should add leading zeros to single-digit time units", func() {
		input := time.Date(2019, time.July, 11, 8, 15, 0, 500, time.UTC)

		timeString := getHumanReadableTime(input)

		expected := "08:15:00"
		Expect(timeString).To(BeEquivalentTo(expected))
	})

	It("should sort exec messages by time", func() {
		inputSlice := make([]*havener.ExecMessage, len(exampleOutput))
		copy(inputSlice, exampleOutput)
		sortedOutput := SortDistributedExecOutput(inputSlice, len(inputSlice), false)

		expected := []*havener.ExecMessage{
			&havener.ExecMessage{
				Prefix: "Prefix1",
				Text:   "Successfully read content of directory!",
				Date:   time.Date(2019, time.July, 11, 8, 20, 15, 670, time.UTC),
			},
			&havener.ExecMessage{
				Prefix: "Prefix2",
				Text:   "Successfully read content of directory!",
				Date:   time.Date(2019, time.July, 11, 8, 20, 15, 995, time.UTC),
			},
			&havener.ExecMessage{
				Prefix: "Prefix1",
				Text:   "dev    home   media  opt    root   sbin   sys    usr",
				Date:   time.Date(2019, time.July, 11, 8, 20, 16, 0, time.UTC),
			},
			&havener.ExecMessage{
				Prefix: "Prefix1",
				Text:   "bin    etc    lib    mnt    proc   run    srv    tmp    var",
				Date:   time.Date(2019, time.July, 11, 8, 20, 16, 333, time.UTC),
			},
			&havener.ExecMessage{
				Prefix: "Prefix2",
				Text:   "bin    etc",
				Date:   time.Date(2019, time.July, 11, 8, 20, 16, 334, time.UTC),
			},
		}
		Expect(sortedOutput).To(BeEquivalentTo(expected))
	})

	It("should sort exec messages by blocks", func() {
		inputSlice := make([]*havener.ExecMessage, len(exampleOutput))
		copy(inputSlice, exampleOutput)
		sortedOutput := SortDistributedExecOutput(inputSlice, len(inputSlice), true)

		expected := []*havener.ExecMessage{
			&havener.ExecMessage{
				Prefix: "Prefix1",
				Text:   "Successfully read content of directory!",
				Date:   time.Date(2019, time.July, 11, 8, 20, 15, 670, time.UTC),
			},
			&havener.ExecMessage{
				Prefix: "Prefix1",
				Text:   "dev    home   media  opt    root   sbin   sys    usr",
				Date:   time.Date(2019, time.July, 11, 8, 20, 16, 0, time.UTC),
			},
			&havener.ExecMessage{
				Prefix: "Prefix1",
				Text:   "bin    etc    lib    mnt    proc   run    srv    tmp    var",
				Date:   time.Date(2019, time.July, 11, 8, 20, 16, 333, time.UTC),
			},
			&havener.ExecMessage{
				Prefix: "Prefix2",
				Text:   "Successfully read content of directory!",
				Date:   time.Date(2019, time.July, 11, 8, 20, 15, 995, time.UTC),
			},
			&havener.ExecMessage{
				Prefix: "Prefix2",
				Text:   "bin    etc",
				Date:   time.Date(2019, time.July, 11, 8, 20, 16, 334, time.UTC),
			},
		}
		Expect(sortedOutput).To(BeEquivalentTo(expected))
	})

	It("should format exec messages", func() {
		formatedOutput, err := FormatDistributedExecOutput(exampleOutput, len(exampleOutput))
		Expect(err).To(BeNil())

		expected := "08:20:16 Prefix1 | bin    etc    lib    mnt    proc   run    srv    tmp    var\r\n" +
			"08:20:15 Prefix2 | Successfully read content of directory!\r\n" +
			"08:20:15 Prefix1 | Successfully read content of directory!\r\n" +
			"08:20:16 Prefix2 | bin    etc\r\n" +
			"08:20:16 Prefix1 | dev    home   media  opt    root   sbin   sys    usr"

		Expect(bunt.RemoveAllEscapeSequences(formatedOutput)).To(BeEquivalentTo(expected))
	})
})
