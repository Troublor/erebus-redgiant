package summary_test

import (
	"math/big"

	"github.com/Troublor/erebus-redgiant/helpers"

	. "github.com/Troublor/erebus-redgiant/analysis/summary"
	"github.com/Troublor/erebus-redgiant/global"

	engine "github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Summarize", func() {
	When("using global.Erigon Database", func() {
		BeforeEach(func() {
			if global.BlockchainReader() == nil {
				Skip("global.Erigon not available")
			}
		})

		It("should summarize a historic transaction", func() {
			txHash := common.HexToHash(
				"0xe83ea41f35bd75ed4e09752a40bf6e314ef2a701a90c296cbb2d2e2297398831",
			)
			s, err := SummarizeTxByHash(global.Ctx(), global.BlockchainReader(), txHash)
			Expect(err).Should(BeNil())
			Expect(s.OverallDefs()).To(HaveLen(3))
			Expect(
				s.OverallUses(),
			).To(HaveLen(13))
			// 8 distinct SLOAD, 1 SELFBALANCE, 2 CALL, 2 EXTCODESIZE
			Expect(s.OverallProfits()).To(HaveLen(3))
			Expect(s.NestedSummaries()).To(HaveLen(2))
			Expect(s.NestedSummaries()[0].MsgCall().Value).
				To(BeEquivalentTo(new(big.Int).Mul(
					big.NewInt(3),
					new(big.Int).Exp(big.NewInt(10), big.NewInt(16), nil),
				)))
			Expect(s.NestedSummaries()[1].NestedSummaries()).To(HaveLen(1))
		})

		It("should summarize a new transaction", func() {
			txHash := common.HexToHash(
				"0xe83ea41f35bd75ed4e09752a40bf6e314ef2a701a90c296cbb2d2e2297398831",
			)
			tx, _, err := global.BlockchainReader().TransactionByHash(global.Ctx(), txHash)
			Expect(err).ShouldNot(HaveOccurred())
			receipt, err := global.BlockchainReader().TransactionReceipt(global.Ctx(), txHash)
			Expect(err).ShouldNot(HaveOccurred())
			state, vmContext, err := helpers.PrepareStateAndContext(
				global.Ctx(),
				global.BlockchainReader(),
				receipt.BlockNumber,
				receipt.TransactionIndex,
			)
			Expect(err).Should(BeNil())
			sender := common.HexToAddress("0x7c20b174f7275a59be485a11652f0990866764a8")
			t := engine.NewTx(sender, &types.LegacyTx{ // same transaction content as previous
				Nonce:    state.GetNonce(sender),
				Gas:      tx.Gas(),
				To:       tx.To(),
				Value:    tx.Value(),
				Data:     tx.Data(),
				GasPrice: tx.GasFeeCap(),
			})
			Expect(t.Hash()).NotTo(Equal(txHash))
			s, err := SummarizeTx(state, vmContext, t)
			Expect(err).Should(BeNil())
			Expect(s.OverallDefs()).To(HaveLen(3))
			Expect(s.OverallUses()).To(HaveLen(13))
			Expect(s.OverallProfits()).To(HaveLen(3))
		})

		It("should summarize transaction 0xccf50f", func() {
			txHash := common.HexToHash(
				"0xccf50f8291815360a122980e4b857b1785929ebd7eec0bd198585e0ab0b072f9",
			)
			s, err := SummarizeTxByHash(global.Ctx(), global.BlockchainReader(), txHash)
			Expect(err).To(BeNil())
			Expect(s.MsgCall().Receipt.Status).To(BeEquivalentTo(1))
			defs := s.OverallDefs()
			Expect(defs).To(HaveLen(7))
		})

		DescribeTable(
			"should summarize corner-case transaction",
			func(txHash string) {
				hash := common.HexToHash(txHash)
				s, err := SummarizeTxByHash(global.Ctx(), global.BlockchainReader(), hash)
				if err != nil {
					println(err.Error())
				}
				Expect(err).To(BeNil())
				Expect(s).NotTo(BeNil())
			},
			Entry("tx0", "0x41bf3158f86c14ea1c5fa8ab12a97aca57ca92b31ba57ceb944470941564fd83"),
			Entry("tx1", "0x234267db95da907859c320280aa9a035191aa1e7e4e73fefb94ec85e34cfa713"),
			Entry("tx2", "0x25732828bc7f7dec8af0635e81b4317d4dff25128ffc1c3aef005f9b677c8a94"),
		)
	})
})
