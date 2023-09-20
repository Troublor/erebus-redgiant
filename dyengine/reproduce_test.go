package dyengine_test

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/params"

	"github.com/Troublor/erebus-redgiant/helpers"

	state2 "github.com/Troublor/erebus-redgiant/dyengine/state"

	"github.com/Troublor/erebus-redgiant/chain"

	"github.com/Troublor/erebus-redgiant/dyengine"

	"github.com/Troublor/erebus-redgiant/dyengine/tracers"
	"github.com/Troublor/erebus-redgiant/global"
	"github.com/rs/zerolog/log"

	logger2 "github.com/ethereum/go-ethereum/eth/tracers/logger"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type jsonRpcStructLog struct {
	Pc      uint64             `json:"pc"`
	Op      string             `json:"op"`
	Gas     uint64             `json:"gas"`
	GasCost uint64             `json:"gasCost"`
	Depth   int                `json:"depth"`
	Error   string             `json:"error,omitempty"`
	Stack   *[]string          `json:"stack,omitempty"`
	Memory  *[]string          `json:"memory,omitempty"`
	Storage *map[string]string `json:"storage,omitempty"`
}

type structLogMatcher struct {
	expected []jsonRpcStructLog
}

func matchStructLogFromJsonRpc(rpcClient *rpc.Client, txHash common.Hash) *structLogMatcher {
	var r struct {
		Gas         uint64             `json:"gas"`
		Failed      bool               `json:"failed"`
		ReturnValue string             `json:"returnValue"`
		StructLogs  []jsonRpcStructLog `json:"structLogs"`
	}
	err := rpcClient.CallContext(
		global.Ctx(),
		&r,
		"debug_traceTransaction",
		txHash.Hex(),
		logger2.Config{
			Debug:          true,
			EnableMemory:   false,
			DisableStorage: true,
			DisableStack:   true,
		},
	)
	if err != nil {
		// if we fail to get the trace, skip the check
		log.Error().
			Err(err).
			Str("tx", txHash.Hex()).
			Msg("Failed to get the struct logs from Eth rpc")
	}
	return &structLogMatcher{
		expected: r.StructLogs,
	}
}

func (m *structLogMatcher) Match(actual interface{}) (success bool, err error) {
	var logs []logger2.StructLog
	var ok bool
	logs, ok = actual.([]logger2.StructLog)
	if !ok {
		return false, errors.New("actual is not []logger2.StructLog")
	}

	if len(logs) != len(m.expected) {
		return false, fmt.Errorf("expected %d logs, got %d", len(m.expected), len(logs))
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("struct log does not match: %v", r)
			success = false
		}
	}()

	for i, ll := range logs {
		expectedLog := m.expected[i]
		if success, err = Equal(expectedLog.Pc).Match(ll.Pc); !success {
			return success, err
		}
		if expectedLog.Op == "SHA3" {
			expectedLog.Op = "KECCAK256"
		}
		if success, err = Equal(expectedLog.Op).Match(ll.Op.String()); !success {
			return success, err
		}
		if success, err = Equal(expectedLog.Gas).Match(ll.Gas); !success {
			return success, err
		}
		if success, err = Equal(expectedLog.Depth).Match(ll.Depth); !success {
			return success, err
		}
		if success, err = Equal(expectedLog.Error).Match(ll.ErrorString()); !success {
			return success, err
		}
	}
	return true, nil
}

type receiptMatcher struct {
	expected *types.Receipt
}

func matchReceipt(receipt *types.Receipt) *receiptMatcher {
	return &receiptMatcher{
		expected: receipt,
	}
}

func (m *receiptMatcher) Match(actual interface{}) (success bool, err error) {
	var actualReceipt *types.Receipt
	var ok bool
	actualReceipt, ok = actual.(*types.Receipt)
	if !ok {
		return false, errors.New("actual is not *types.Receipt")
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("receipt does not match: %v", r)
			success = false
		}
	}()

	expectedReceipt := m.expected
	if success, err = Equal(expectedReceipt.Type).Match(actualReceipt.Type); !success {
		return success, err
	}
	if success, err = Equal(expectedReceipt.Status).Match(actualReceipt.Status); !success {
		return success, err
	}
	if success, err = Equal(expectedReceipt.CumulativeGasUsed).Match(actualReceipt.CumulativeGasUsed); !success {
		return success, err
	}
	//Expect(actualReceipt.PostState).To(BeEquivalentTo(expectedReceipt.PostState))
	// FIXME Erigon ReadReceipt seems to return an empty Bloom
	//Expect(actualReceipt.Bloom).To(Same(expectedReceipt.Bloom))
	if success, err = HaveLen(len(expectedReceipt.Logs)).Match(actualReceipt.Logs); !success {
		return success, err
	}
	for i, expectedLog := range expectedReceipt.Logs {
		actualLog := actualReceipt.Logs[i]
		if success, err = BeEquivalentTo(expectedLog).Match(actualLog); !success {
			return success, err
		}
	}
	if success, err = Equal(expectedReceipt.TxHash).Match(actualReceipt.TxHash); !success {
		return success, err
	}
	if success, err = Equal(expectedReceipt.ContractAddress).Match(actualReceipt.ContractAddress); !success {
		return success, err
	}
	if success, err = Equal(expectedReceipt.GasUsed).Match(actualReceipt.GasUsed); !success {
		return success, err
	}
	if success, err = Equal(expectedReceipt.BlockHash).Match(actualReceipt.BlockHash); !success {
		return success, err
	}
	if success, err = Equal(expectedReceipt.BlockNumber).Match(actualReceipt.BlockNumber); !success {
		return success, err
	}
	if success, err = Equal(expectedReceipt.TransactionIndex).Match(actualReceipt.TransactionIndex); !success {
		return success, err
	}
	return true, nil
}

func (m *receiptMatcher) FailureMessage(actual interface{}) (message string) {
	return "receipt not match"
}

func (m *receiptMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return "receipt not match"
}

func (m *structLogMatcher) FailureMessage(actual interface{}) (message string) {
	return "struct log not match"
}

func (m *structLogMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return "struct log not match"
}

var _ = Describe("Reproduce", Ordered, func() {
	ctx := global.Ctx()
	var rpcClient *rpc.Client
	var err error
	var chainReader chain.BlockchainReader
	BeforeAll(func() {
		chainReader = global.BlockchainReader()
		if chainReader == nil {
			Skip("blockchain reader not available")
		}
		rpcClient = global.JsonRpcClient()
		if rpcClient == nil {
			Skip(fmt.Sprintf("Eth JSON rpc not available: %e", err))
		}
	})

	DescribeTable(
		"should reproduce transaction execution",
		func(hashStr string) {
			txHash := common.HexToHash(hashStr)
			tx, _, err := chainReader.TransactionByHash(ctx, txHash)
			Expect(err).ShouldNot(HaveOccurred())
			receipt, err := chainReader.TransactionReceipt(ctx, txHash)
			Expect(err).ShouldNot(HaveOccurred())
			state, vmContext, err := helpers.PrepareStateAndContext(
				ctx, chainReader, receipt.BlockNumber, receipt.TransactionIndex,
			)
			Expect(err).ShouldNot(HaveOccurred())

			tracer := tracers.NewStructLogTracer(tracers.StructLogConfig{
				Limit: 0,
			})
			exeVM := dyengine.NewExeVM(helpers.VMConfigOnMainnet())
			exeVM.SetTracer(tracer)
			_, actualReceipt, err := exeVM.ApplyTransaction(state, tx, vmContext, false, true)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(actualReceipt).To(matchReceipt(receipt))
			Expect(tracer.StructLogs()).To(matchStructLogFromJsonRpc(rpcClient, txHash))
		},
		Entry(
			"one transaction",
			"0xbcab6b1ef2346cc5c3ff67c9f029c346b68cd2b07a20dcb1f2a0b68a30119eff",
		),
		// Entry("stuck transaction", "0x48db4231fb7de49b624f0860fed5f459ca7231ff1f22ef58f00c1cefd6e454ef"),
		Entry("t0", "0x234267db95da907859c320280aa9a035191aa1e7e4e73fefb94ec85e34cfa713"),
	)

	DescribeTable(
		"should reproduce block execution",
		func(blockNumber *big.Int) {
			block, err := chainReader.BlockByNumber(ctx, blockNumber)
			Expect(err).ShouldNot(HaveOccurred())
			tracer := tracers.NewStructLogTracer(tracers.StructLogConfig{
				Limit: 0,
			})
			exeVM := &dyengine.ExeVM{
				Config: helpers.VMConfigOnMainnet(),
				Tracer: tracer,
			}
			tracer.TxEnd = func(
				structLogs []logger2.StructLog, tx *types.Transaction,
				vmContext *dyengine.VMContext, state dyengine.State, receipt *types.Receipt,
			) {
				expectedReceipt, err := chainReader.TransactionReceipt(ctx, tx.Hash())
				Expect(err).ShouldNot(HaveOccurred())
				Expect(expectedReceipt).To(matchReceipt(receipt))
				Expect(structLogs).To(matchStructLogFromJsonRpc(rpcClient, tx.Hash()))
			}
			state, err := state2.NewForkedState(
				chainReader,
				new(big.Int).Sub(blockNumber, big.NewInt(1)),
			)
			Expect(err).ShouldNot(HaveOccurred())
			vmContext := helpers.VMContextFromBlock(block)

			_, _, _, err = exeVM.ApplyTransactions(
				state, block.Transactions(), vmContext, false, true,
			)
			Expect(err).ShouldNot(HaveOccurred())
		},
		//Entry("block 2000002", big.NewInt(2000002)),
		// Entry("block 10000000", big.NewInt(10000000)),
		Entry(
			"a block before London",
			new(big.Int).Add(params.MainnetChainConfig.ByzantiumBlock, big.NewInt(1)),
		),
		Entry(
			"a block at London",
			new(big.Int).Add(params.MainnetChainConfig.LondonBlock, big.NewInt(1)),
		),
		Entry("a block with gas refund", big.NewInt(13005003)),
	)

	//It("should reproduce a block with special case", func() {
	//	blockNumber := big.NewInt(13002030)
	//	reproduceBlock(blockNumber)
	//})
})
