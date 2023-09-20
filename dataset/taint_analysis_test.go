package dataset_test

import (
	"math/big"

	"github.com/Troublor/erebus-redgiant/dyengine/tracers"
	"github.com/Troublor/erebus-redgiant/helpers"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"

	"github.com/Troublor/erebus-redgiant/analysis/data_flow"
	"github.com/Troublor/erebus-redgiant/analysis/summary"
	. "github.com/Troublor/erebus-redgiant/contract"
	. "github.com/Troublor/erebus-redgiant/dataset"
	engine "github.com/Troublor/erebus-redgiant/dyengine"
	. "github.com/Troublor/erebus-redgiant/dyengine/state"
)

//go:generate solc --bin --asm --abi --storage-layout ./__test__/taint_analysis/Contract.sol -o ./__test__/taint_analysis --overwrite
var _ = Describe("TaintAnalysis", func() {
	var exeVM engine.ExeVM
	var state *MemoryState
	var vmContext *engine.VMContext
	var creator common.Address
	var contract common.Address
	var caller common.Address

	// send tx and get execution path on a copied state.
	getExecutionPath := func(
		from common.Address, to *common.Address, value *big.Int, input []byte,
	) (*types.Receipt, tracers.Trace, error) {
		summaryTracer := summary.NewTxSummaryTracer(summary.Config{IncludeTrace: true})
		tracer := exeVM.Tracer
		exeVM.SetTracer(summaryTracer)
		defer exeVM.SetTracer(tracer)
		_, r, err := exeVM.DebuggingCall(state.Copy(), vmContext.Copy(), from, to, value, input)
		return r, summaryTracer.Summary.FlattenedExecutionPath(), err
	}

	BeforeEach(func() {
		exeVM = *helpers.NewDebuggingExeVM()
		state = NewMemoryState()
		vmContext = helpers.DebuggingVMContext()
		creator = common.HexToAddress("0x0")
		caller = common.HexToAddress("0x1")
	})

	Context("with toy contract", func() {
		var contractAbi abi.ABI
		BeforeEach(func() {
			code := LoadBinaryFromFile("./__test__/taint_analysis/Contract.bin")
			contractAbi = LoadAbiFromFile("./__test__/taint_analysis/Contract.abi")
			// deploy contract
			state.StateDB.AddBalance(creator, big.NewInt(1000000000))
			_, receipt, err := exeVM.DebuggingCall(
				state,
				vmContext,
				creator,
				nil,
				big.NewInt(100),
				code,
			)
			if err != nil {
				Fail(err.Error())
			}
			contract = receipt.ContractAddress
		})

		type tx struct {
			input []byte
			value *big.Int
		}
		type testCase struct {
			sharedVariable     summary.StateVariable
			victimTx, attackTx tx
			oracle             func(analyzer *TaintAnalyzer)
		}

		genTest := func(c testCase) {
			// run with out alteration
			receipt, refPath, err := getExecutionPath(
				caller, &contract, c.victimTx.value, c.victimTx.input,
			)
			if receipt.Status != 1 {
				log.Info().Msg("victim tx failed in attack-free scenario")
			}
			Expect(err).ToNot(HaveOccurred())

			// alter shared variable
			_, r, err := exeVM.DebuggingCall(
				state, vmContext, caller, &contract, c.attackTx.value, c.attackTx.input,
			)
			Expect(r.Status).To(Equal(uint64(1)))
			Expect(err).ToNot(HaveOccurred())

			analyzer := NewTaintAnalyzer(state, c.sharedVariable, refPath)
			tracer := data_flow.NewDataFlowTracer(analyzer)
			exeVM.SetTracer(tracer)
			_, r, err = exeVM.DebuggingCall(
				state, vmContext, caller, &contract, c.victimTx.value, c.victimTx.input,
			)
			if r.Status == 0 {
				log.Info().Msg("victim transaction failed in attack scenario")
			}
			Expect(err).ToNot(HaveOccurred())

			c.oracle(analyzer)
		}

		Context("when propagate throw data flow", Ordered, func() {
			It("should propagate storage variable", func() {
				genTest(testCase{
					sharedVariable: summary.StorageVariable{
						Address: contract,
						Storage: common.HexToHash("0x0"),
					},
					victimTx: tx{
						input: MustPack(contractAbi, "transfer"),
						value: big.NewInt(0),
					},
					attackTx: tx{
						input: MustPack(contractAbi, "setBalance", big.NewInt(1)),
						value: big.NewInt(0),
					},
					oracle: func(analyzer *TaintAnalyzer) {
						Expect(analyzer.Logs).To(HaveLen(1))
						Expect(analyzer.Calls).To(HaveLen(1))
						Expect(analyzer.Jumps).To(HaveLen(1))
					},
				})
			})
		})

		Context("when propagate throw direct control dependency", Ordered, func() {
			It("should propagate storage variable", func() {
				genTest(testCase{
					sharedVariable: summary.StorageVariable{
						Address: contract,
						Storage: common.HexToHash("0x0"),
					},
					victimTx: tx{
						input: MustPack(contractAbi, "withdraw"),
						value: big.NewInt(0),
					},
					attackTx: tx{
						input: MustPack(contractAbi, "setBalance", big.NewInt(101)),
						value: big.NewInt(0),
					},
					oracle: func(analyzer *TaintAnalyzer) {
						Expect(analyzer.Logs).To(HaveLen(1))
						Expect(analyzer.Calls).To(HaveLen(1))
					},
				})
			})
		})

		Context("when propagate throw control depending data flow", Ordered, func() {
			It("should propagate local variable that control-depend on storage variable", func() {
				genTest(testCase{
					sharedVariable: summary.StorageVariable{
						Address: contract,
						Storage: common.HexToHash("0x0"),
					},
					victimTx: tx{
						input: MustPack(contractAbi, "withdrawAmountOrOne", big.NewInt(100)),
						value: big.NewInt(0),
					},
					attackTx: tx{
						input: MustPack(contractAbi, "setBalance", big.NewInt(1)),
						value: big.NewInt(0),
					},
					oracle: func(analyzer *TaintAnalyzer) {
						Expect(analyzer.Logs).To(HaveLen(1))
						Expect(analyzer.Calls).To(HaveLen(1))
					},
				})
			})
		})
	})
})
