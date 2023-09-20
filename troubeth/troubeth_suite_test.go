package troubeth_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTroubeth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Troubeth Suite")
}
