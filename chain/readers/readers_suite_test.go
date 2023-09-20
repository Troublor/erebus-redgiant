package readers_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestReaders(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Chain Readers Suite")
}
