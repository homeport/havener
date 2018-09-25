package havener_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHavener(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Havener Suite")
}
