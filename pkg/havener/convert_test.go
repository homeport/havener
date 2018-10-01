package havener_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.ibm.com/hatch/havener/pkg/havener"
)

var _ = Describe("Convert", func() {
	Context("value conversions", func() {
		It("should return the human readable version of given number of bytes", func() {
			Expect(HumanReadableSize(15784004812)).To(BeEquivalentTo("14.7 GiB"))
		})
	})
})
