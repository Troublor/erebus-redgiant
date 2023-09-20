package contract_test

import (
	"github.com/Troublor/erebus-redgiant/contract"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Abi", func() {
	Context("ERC721 ABI", func() {
		It("should contain overloaded method as expected", func() {
			_, ok := contract.ERC721ABI.Methods["safeTransferFrom"]
			Expect(ok).To(BeTrue())
			_, ok = contract.ERC721ABI.Methods["safeTransferFrom0"]
			Expect(ok).To(BeTrue())
			_, ok = contract.ERC721ABI.Methods["safeTransferFrom1"]
			Expect(ok).To(BeFalse())
		})
	})
})
