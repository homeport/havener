// Copyright © 2022 The Homeport Team
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

package havener_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/homeport/havener/pkg/havener"
)

var _ = Describe("Convert", func() {
	Context("value conversions", func() {
		It("should return the human readable version of given number of bytes", func() {
			Expect(HumanReadableSize(15784004812)).To(BeEquivalentTo("14.7 GiB"))
			Expect(HumanReadableSize(1073741824)).To(BeEquivalentTo("1.0 GiB"))
			Expect(HumanReadableSize(0)).To(BeEquivalentTo("0.0 Byte"))
		})

		It("should not fail on negative numbers", func() {
			// esoteric edge case (negative numbers are not processed)
			Expect(HumanReadableSize(-1)).To(BeEquivalentTo("-1.0 Byte"))
		})
	})
})
