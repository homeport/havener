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

package cmd_test

import (
	"github.com/gonvenience/term"

	. "github.com/gonvenience/bunt"
	. "github.com/homeport/havener/internal/cmd"
	. "github.com/homeport/havener/pkg/havener"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("usage details string rendering", func() {
	BeforeEach(func() {
		SetColorSettings(OFF, OFF)
		term.FixedTerminalWidth = 120
		term.FixedTerminalHeight = 40
	})

	AfterEach(func() {
		SetColorSettings(AUTO, AUTO)
		term.FixedTerminalWidth = -1
		term.FixedTerminalHeight = -1
	})

	Context("render node details", func() {
		It("should render the node details in a somehow pleasant and readable form", func() {
			Expect(term.GetTerminalWidth()).To(BeEquivalentTo(120))
			Expect(RenderNodeDetails(&TopDetails{
				Nodes: map[string]NodeDetails{
					"node1": {
						TotalCPU:    4000,
						TotalMemory: 16384000,
						UsedCPU:     2000,
						UsedMemory:  8192000,
						LoadAvg:     []float64{4.0, 2.0, 1.0},
					},
					"node2": {
						TotalCPU:    4000,
						TotalMemory: 16384000,
						UsedCPU:     4000,
						UsedMemory:  16384000,
						LoadAvg:     []float64{10.0, 10.0, 10.0},
					},
				},
			})).To(BeEquivalentTo(`╭ CPU and Memory usage by Node
│ node2  Load 10.0 10.0 10.0  CPU ■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■ 100.0%  Memory ■■■■■■■■■■■■■■■■■■■ 15.6 MiB/15.6 MiB
│ node1  Load 4.0 2.0 1.0     CPU ■■■■■■■■■■■■■■■■■                  50.0%  Memory ■■■■■■■■■■           7.8 MiB/15.6 MiB
╵
`))
		})

		It("should render the node details even if not all details are available", func() {
			Expect(term.GetTerminalWidth()).To(BeEquivalentTo(120))
			Expect(RenderNodeDetails(&TopDetails{
				Nodes: map[string]NodeDetails{
					"node1": {
						TotalCPU:    4000,
						TotalMemory: 16384000,
						UsedCPU:     2000,
						UsedMemory:  8192000,
						LoadAvg:     nil,
					},
					"node2": {
						TotalCPU:    4000,
						TotalMemory: 16384000,
						UsedCPU:     4000,
						UsedMemory:  16384000,
						LoadAvg:     []float64{10.0, 10.0, 10.0},
					},
				},
			})).To(BeEquivalentTo(`╭ CPU and Memory usage by Node
│ node2  Load 10.0 10.0 10.0  CPU ■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■ 100.0%  Memory ■■■■■■■■■■■■■■■■■■■ 15.6 MiB/15.6 MiB
│ node1  Load (no data)       CPU ■■■■■■■■■■■■■■■■■                  50.0%  Memory ■■■■■■■■■■           7.8 MiB/15.6 MiB
╵
`))
		})
	})
})
