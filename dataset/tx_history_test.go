package dataset_test

import (
	"math/big"

	. "github.com/Troublor/erebus-redgiant/dataset"
	"github.com/Troublor/erebus-redgiant/global"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TxHistory", func() {
	var txHistory *TxHistory
	BeforeEach(func() {
		txHistory = NewTxHistory(global.BlockchainReader(), global.TroubEth())
	})

	Describe("TxHistorySession", func() {
		Context("on single block starting with 13000003", Ordered, func() {
			var blockNumber uint64 = 13000003
			var session *TxHistorySession

			It("should be started with specified window size", func() {
				session = txHistory.StartSession(
					global.Ctx(),
					global.GoroutinePool(),
					blockNumber,
					1,
				)
				Expect(session.Size()).To(Equal(1))
			})

			It("should contains same amount of transactions", func() {
				block, err := global.BlockchainReader().
					BlockByNumber(global.Ctx(), big.NewInt(int64(blockNumber)))
				Expect(err).Should(BeNil())
				allRecords := session.SliceTxRecords()
				// +1 for the block reward pseudo transaction
				Expect(allRecords).To(HaveLen(block.Transactions().Len() + 1))
			})

			It("should slide prerequisites correctly", func() {
				// targetTx := common.HexToHash("0x92cf149f11d9ecbe1d8edc2ca6a3a0be58995db5626d3ea77227974662c4a6d1")
				// boundTx := common.HexToHash("0xb854be2f2368cf5adb869c88f8200f69a1a9875e756030dbb75b33b4f181db36")
				targetTxPos := TxPosition{blockNumber, 142}
				boundTxPos := TxPosition{blockNumber, 137}
				target := session.GetTxRecord(targetTxPos)
				bound := session.GetTxRecord(boundTxPos)
				prerequisites := session.SlicePrerequisites(target, bound)
				Expect(prerequisites).To(HaveLen(2)) // two prerequisites:
				// 0xdf09fba4c1a315cf76ea226c65a3e4d51be6131c0d1d7ff7e09d883325c9ed52
				// 0xcc279325df5fbde6f957af5eecb1da0799aa1d08a60c5a1370af653684876c68
			})
		})

		Context("on multiple blocks 13000008 - 13000009", Ordered, func() {
			var blockNumber uint64 = 13000008
			var windowSize = 2
			var session *TxHistorySession

			It("should be started with specified window size", func() {
				session = txHistory.StartSession(
					global.Ctx(),
					global.GoroutinePool(),
					blockNumber,
					windowSize,
				)
				Expect(session.Size()).To(Equal(windowSize))
			})

			It("should contains same amount of transactions", func() {
				total := 0
				for i := 0; i < windowSize; i++ {
					block, err := global.BlockchainReader().BlockByNumber(global.Ctx(),
						big.NewInt(int64(blockNumber)+int64(i)))
					Expect(err).Should(BeNil())
					total += block.Transactions().Len()
				}
				allRecords := session.SliceTxRecords()
				// +1 for the block reward pseudo transaction
				Expect(allRecords).To(HaveLen(total + windowSize))
			})

			It("should slide cross-block prerequisites correctly", func() {
				// targetTx := common.HexToHash("0x68bb5d0aedb28115533d820a63999561c7352543ad314f0ca15572b3ef552fd6")
				// boundTx := common.HexToHash("0xed3ce0285d2ffa43533d8ee48dbc50d4ff69279e64786232042be058987d5b43")
				targetTxPos := TxPosition{blockNumber + 1, 101}
				boundTxPos := TxPosition{blockNumber, 0}
				target := session.GetTxRecord(targetTxPos)
				bound := session.GetTxRecord(boundTxPos)
				prerequisites := session.SlicePrerequisites(target, bound)
				Expect(prerequisites).To(HaveLen(7))
			})
		})
	})
})
