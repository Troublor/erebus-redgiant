package storage_address_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestStorageAddress(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "StorageAddress Suite")
}
