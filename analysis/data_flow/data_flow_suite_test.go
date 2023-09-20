package data_flow_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDataFlow(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DataFlow Suite")
}
