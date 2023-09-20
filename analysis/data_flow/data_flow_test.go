package data_flow_test

import (
	"context"
	"fmt"
	"math/big"

	"github.com/Troublor/erebus-redgiant/analysis/storage_address"
	. "github.com/Troublor/erebus-redgiant/chain"
	"github.com/Troublor/erebus-redgiant/global"

	"github.com/Troublor/erebus-redgiant/helpers"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/core/vm"

	. "github.com/Troublor/erebus-redgiant/analysis/data_flow"
	engine "github.com/Troublor/erebus-redgiant/dyengine"
	. "github.com/Troublor/erebus-redgiant/dyengine/state"
	"github.com/ethereum/go-ethereum/accounts/abi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

//go:generate solc --bin --asm --abi --storage-layout ./__test__/contract.sol -o ./__test__/contract --overwrite
var _ = Describe("DataFlow", func() {
	Context("on dummy contract", func() {
		var bytecode []byte
		var state engine.State
		var contractAbi abi.ABI
		var exeVM *engine.ExeVM
		var vmContext *engine.VMContext

		var account common.Address
		var anotherAccount common.Address
		var contract common.Address

		setupInitialState := func() {
			args, err := contractAbi.Pack("", big.NewInt(100))
			if err != nil {
				Fail(err.Error())
			}
			// deploy contract
			r, receipt, err := exeVM.DebuggingCall(state, vmContext, account, nil, nil, append(bytecode, args...))
			if err != nil {
				Fail(fmt.Sprintf("Failed to deploy contract: %s", err.Error()))
			}
			if r.Failed() {
				Fail(fmt.Sprintf("Failed to deploy contract: %s", r.Err.Error()))
			}
			contract = receipt.ContractAddress
		}

		BeforeEach(func() {
			bytecode = helpers.LoadBinaryFromFile("./__test__/contract/Contract.bin")
			account = common.HexToAddress("0x0000000000000000000000000000000000000001")
			anotherAccount = common.HexToAddress("0x0000000000000000000000000000000000000002")
			contractAbi = helpers.LoadAbiFromFile("./__test__/contract/Contract.abi")

			state = NewMemoryState()
			exeVM = helpers.NewDebuggingExeVM()
			vmContext = helpers.DebuggingVMContext()

			setupInitialState()
		})

		It("account should have 100 balance", func() {
			args, err := contractAbi.Pack("balanceOf", account)
			Expect(err).To(BeNil())
			r, _, err := exeVM.DebuggingCall(state, vmContext, account, &contract, nil, args)
			Expect(err).To(BeNil())
			Expect(r.Failed()).To(BeFalse())
			returns, err := contractAbi.Unpack("balanceOf", r.Return())
			Expect(err).To(BeNil())
			Expect(returns[0].(*big.Int).String()).To(Equal("100"))
		})

		It("should taint the return data in balanceOf function with source as arguments", func() {
			analyzer := &analyzer{
				operation: func(operation *Operation) (isSource, isSink bool) {
					switch operation.OpCode() {
					case vm.CALLDATALOAD:
						return true, false
					case vm.RETURN:
						return false, true
					default:
						return false, false
					}
				},
				oracle: func(collection *TrackerCollection, flowedValue FlowNode) {
					Expect(flowedValue.Operation().OpCode()).To(Equal(vm.RETURN))
					Expect(collection.Call.GetReturnData(0, 32)).
						To(ContainElement(TaintedByNode(func(node FlowNode) bool {
							switch node.Operation().OpCode() {
							case vm.CALLDATALOAD, vm.CALLDATACOPY:
								return true
							default:
								return false
							}
						})))
				},
			}
			tracer := NewDataFlowTracer(analyzer)
			exeVM.Tracer = tracer

			args, _ := contractAbi.Pack("balanceOf", account)
			r, _, err := exeVM.DebuggingCall(state, vmContext, account, &contract, nil, args)
			Expect(err).To(BeNil())
			Expect(r.Failed()).To(BeFalse())
			Expect(analyzer.sinkTainted).To(BeTrue())
		})

		It("should taint the log in transferFrom function with source as arguments", func() {
			analyzer := &analyzer{
				operation: func(operation *Operation) (isSource, isSink bool) {
					switch operation.OpCode() {
					case vm.CALLDATALOAD, vm.CALLDATACOPY:
						return true, false
					case vm.LOG0, vm.LOG1, vm.LOG2, vm.LOG3, vm.LOG4:
						return false, true
					default:
						return false, false
					}
				},
				oracle: func(collection *TrackerCollection, flowedValue FlowNode) {
					Expect(flowedValue.Operation().OpCode()).
						To(And(BeNumerically(">=", vm.LOG0), BeNumerically("<=", vm.LOG4)))
					Expect(flowedValue).To(TaintedByNode(func(node FlowNode) bool {
						switch node.Operation().OpCode() {
						case vm.CALLDATALOAD, vm.CALLDATACOPY:
							return true
						default:
							return false
						}
					}))
				},
			}
			exeVM.Tracer = NewDataFlowTracer(analyzer)

			args, _ := contractAbi.Pack("transferFrom", account, anotherAccount, big.NewInt(100))
			r, _, err := exeVM.DebuggingCall(state, vmContext, account, &contract, nil, args)
			Expect(err).To(BeNil())
			Expect(r.Failed()).To(BeFalse())
			Expect(analyzer.sinkTainted).To(BeTrue())
		})

		It("should flow tainted value through message call", func() {
			analyzer := &analyzer{
				operation: func(operation *Operation) (isSource, isSink bool) {
					switch operation.OpCode() {
					case vm.CALLDATALOAD, vm.CALLDATACOPY:
						return true, false
					case vm.LOG0, vm.LOG1, vm.LOG2, vm.LOG3, vm.LOG4:
						return false, true
					default:
						return false, false
					}
				},
				oracle: func(collection *TrackerCollection, flowedValue FlowNode) {
					Expect(flowedValue.Operation().OpCode()).
						To(And(BeNumerically(">=", vm.LOG0), BeNumerically("<=", vm.LOG4)))
					Expect(flowedValue).To(TaintedByNode(func(node FlowNode) bool {
						switch node.Operation().OpCode() {
						case vm.CALLDATALOAD, vm.CALLDATACOPY:
							return node.Operation().MsgCall().Position.String() == "root"
						default:
							return false
						}
					}))
				},
			}
			exeVM.Tracer = NewDataFlowTracer(analyzer)

			args, _ := contractAbi.Pack("transfer", anotherAccount, big.NewInt(100))
			r, _, err := exeVM.DebuggingCall(state, vmContext, account, &contract, nil, args)
			Expect(err).To(BeNil())
			Expect(r.Failed()).To(BeFalse())
			Expect(analyzer.sinkTainted).To(BeTrue())
		})

		It("should flow tainted value with isolation for different analyzers", func() {
			analyzer := &analyzer{
				operation: func(operation *Operation) (isSource, isSink bool) {
					switch operation.OpCode() {
					case vm.CALLDATALOAD, vm.CALLDATACOPY:
						return true, false
					case vm.LOG0, vm.LOG1, vm.LOG2, vm.LOG3, vm.LOG4:
						return false, true
					default:
						return false, false
					}
				},
				oracle: func(collection *TrackerCollection, flowedValue FlowNode) {
					Expect(flowedValue.Operation().OpCode()).
						To(And(BeNumerically(">=", vm.LOG0), BeNumerically("<=", vm.LOG4)))
					Expect(flowedValue).To(TaintedByNode(func(node FlowNode) bool {
						switch node.Operation().OpCode() {
						case vm.CALLDATALOAD, vm.CALLDATACOPY:
							return node.Operation().MsgCall().Position.String() == "root"
						default:
							return false
						}
					}))
				},
			}
			var addressingPaths []storage_address.AddressingPath
			anotherAnalyzer := &storage_address.StorageAddressingPathAnalyzer{
				OnStorageStoredOrLoaded: func(
					op vm.OpCode, addressingPathCandidates []storage_address.AddressingPath,
				) {
					addressingPaths = append(addressingPaths, addressingPathCandidates...)
				},
			}
			exeVM.Tracer = NewDataFlowTracer(analyzer, anotherAnalyzer)

			args, _ := contractAbi.Pack("transfer", anotherAccount, big.NewInt(100))
			r, _, err := exeVM.DebuggingCall(state, vmContext, account, &contract, nil, args)
			Expect(err).To(BeNil())
			Expect(r.Failed()).To(BeFalse())
			Expect(analyzer.sinkTainted).To(BeTrue())

			Expect(addressingPaths).To(HaveLen(5))
		})
	})

	Context("using real transaction", func() {
		var ctx context.Context
		var err error
		var erigon BlockchainReader
		var exeVM *engine.ExeVM

		BeforeEach(func() {
			ctx = global.Ctx()
			erigon = global.BlockchainReader()
			if err != nil {
				Skip(fmt.Sprintf("Erigon not available: %e", err))
			}

			exeVM = engine.NewExeVM(helpers.VMConfigOnMainnet())
		})

		It("should taint a Uniswap exchange transaction", func() {
			txHash := common.HexToHash("0x8b57ca5ab975e3f6a0b65e97d65ae0acc61a054ef010029cea04e50d65b6612c")
			tx, _, err := erigon.TransactionByHash(ctx, txHash)
			Expect(err).To(BeNil())
			receipt, err := erigon.TransactionReceipt(ctx, txHash)
			Expect(err).To(BeNil())
			state, vmContext, err := helpers.PrepareStateAndContext(
				ctx, erigon, receipt.BlockNumber, receipt.TransactionIndex,
			)
			Expect(err).To(BeNil())

			analyzer := &analyzer{
				operation: func(operation *Operation) (isSource, isSink bool) {
					switch operation.OpCode() {
					case vm.CALLDATALOAD, vm.CALLDATASIZE:
						return true, false
					case vm.LOG3:
						return false, true
					default:
						return false, false
					}
				},
				oracle: func(collection *TrackerCollection, flowedValue FlowNode) {
					Expect(flowedValue.Operation().OpCode()).To(Equal(vm.LOG3))
					Expect(flowedValue).To(TaintedByNode(func(node FlowNode) bool {
						switch node.Operation().OpCode() {
						case vm.CALLDATALOAD, vm.CALLDATACOPY:
							return node.Operation().MsgCall().Position.String() == "root"
						default:
							return false
						}
					}))
				},
			}
			exeVM.Tracer = NewDataFlowTracer(analyzer)

			r, _, err := exeVM.ApplyTransaction(state, tx, vmContext, false, false)
			Expect(err).To(BeNil())
			Expect(r.Failed()).To(BeFalse())
			Expect(analyzer.sinkTainted).To(BeTrue())
		})

		It("should taint path condition reading from storage", func() {
			// this tx is front-run by another transaction by filling 0x order in advance
			txHash := common.HexToHash("0x5fb29c4359a81166f072e59c7fc8a72fc386d3924f6efe5a014857ed39eb03f1")
			tx, _, err := erigon.TransactionByHash(ctx, txHash)
			Expect(err).To(BeNil())
			receipt, err := erigon.TransactionReceipt(ctx, txHash)
			Expect(err).To(BeNil())
			state, vmContext, err := helpers.PrepareStateAndContext(
				ctx, erigon, receipt.BlockNumber, receipt.TransactionIndex,
			)
			Expect(err).To(BeNil())

			analyzer := &analyzer{
				operation: func(operation *Operation) (isSource, isSink bool) {
					switch operation.OpCode() {
					case vm.SLOAD:
						return true, false
					case vm.JUMPI:
						return false, true
					default:
						return false, false
					}
				},
				oracle: func(collection *TrackerCollection, flowedValue FlowNode) {
					Expect(flowedValue.Operation().OpCode()).To(Equal(vm.JUMPI))
					Expect(flowedValue).To(TaintedByNode(func(node FlowNode) bool {
						switch node.Operation().OpCode() {
						case vm.SLOAD:
							return node.Operation().Arg(0).Hash() == common.HexToHash(
								"0xf76429700eefa3bbf421a3697eed227f07ce0f64550ba9784777f1c19c014d0c",
							)
						default:
							return false
						}
					}))
				},
			}
			exeVM.Tracer = NewDataFlowTracer(analyzer)

			r, _, err := exeVM.ApplyTransaction(state, tx, vmContext, false, false)
			Expect(err).To(BeNil())
			Expect(r.Failed()).To(BeFalse())
			Expect(analyzer.sinkTainted).To(BeTrue())
		})
	})
})

type analyzer struct {
	operation   func(operation *Operation) (isSource, isSink bool)
	oracle      func(collection *TrackerCollection, flowedValue FlowNode)
	sinkTainted bool
}

func (a *analyzer) IsMyFlowNode(node FlowNode) bool {
	if r, ok := node.(*RawFlowNode); ok {
		if r.Label() == "test-analyzer" {
			return true
		}
	}
	return false
}

func (a *analyzer) NewFlowNode(operation *Operation) FlowNode {
	return NewRawFlowNode(operation, "test-analyzer")
}

func (a *analyzer) FlowPolicy() FlowPolicy {
	return GenDefaultFlowPolicy()
}

func (a *analyzer) CheckOperation(operation *Operation) (isSource, isSink bool) {
	return a.operation(operation)
}

func (a *analyzer) SinkTainted(collection *TrackerCollection, flowedValue FlowNode) {
	a.sinkTainted = true
	a.oracle(collection, flowedValue)
}

type taintedByNodeMatcher struct {
	SomeFlowNode func(FlowNode) bool
}

func TaintedByNode(checker func(node FlowNode) bool) *taintedByNodeMatcher {
	return &taintedByNodeMatcher{checker}
}

func (t *taintedByNodeMatcher) Match(actual interface{}) (success bool, err error) {
	if node, ok := actual.(FlowNode); ok {
		if t.SomeFlowNode(node) {
			return true, nil
		}

		for _, flowNode := range node.From() {
			success, err := t.Match(flowNode)
			if err != nil {
				return false, err
			}
			if success {
				return true, nil
			}
		}
		return false, nil
	}
	return false, fmt.Errorf("%T is not a FlowNode", actual)
}

func (t *taintedByNodeMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected %v to be tainted by specific source", actual)
}

func (t *taintedByNodeMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected %v not to be tainted by specific source", actual)
}
