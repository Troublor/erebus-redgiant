package contract_test

import (
	"context"
	_ "embed"

	"github.com/Troublor/erebus-redgiant/chain"

	"github.com/Troublor/erebus-redgiant/contract"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Troublor/erebus-redgiant/global"
	"github.com/ethereum/go-ethereum/common"
)

var _ = Describe("Ethereum", func() {
	ctx := context.Background()
	Context("using Erigon DB", func() {
		var chainReader chain.BlockchainReader
		BeforeEach(func() {
			chainReader = global.BlockchainReader()
			if chainReader == nil {
				Skip("no blockchain reader available")
			}
		})

		It("should parse a ERC20 Transfer event", func() {
			txHash := common.HexToHash("0x2f7f2e88c8e39f142bf21f6e7f9c3be033c1f5abec718ea4ca1854487ecbc195")
			receipt, err := chainReader.TransactionReceipt(ctx, txHash)
			Expect(err).ShouldNot(HaveOccurred())
			args, err := contract.ParseEvent(contract.ERC20ABI.Events["Transfer"], receipt.Logs[0])
			Expect(err).ShouldNot(HaveOccurred())
			Expect(args).To(HaveLen(3))
		})

		It("should parse a ERC721 Transfer event", func() {
			txHash := common.HexToHash("0x045926bb4d32997d68ea506bafdb2865e44484adf5252cc3e687d4d4f6e9c67c")
			receipt, err := chainReader.TransactionReceipt(ctx, txHash)
			Expect(err).ShouldNot(HaveOccurred())
			log := receipt.Logs[1]
			Expect(log.Topics[0]).To(Equal(common.HexToHash(
				"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
			)))
			args, err := contract.ParseEvent(contract.ERC721ABI.Events["Transfer"], log)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(args).To(HaveLen(3))
		})
	})
})
