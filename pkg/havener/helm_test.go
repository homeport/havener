package havener_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.ibm.com/hatch/havener/pkg/havener"
)

var _ = Describe("Helm", func() {
	Context("some context", func() {
		It("should return a string", func() {
			Expect(StandIn()).To(BeEquivalentTo("standin"))
		})
	})
})
