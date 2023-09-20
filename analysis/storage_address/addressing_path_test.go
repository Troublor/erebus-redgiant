package storage_address_test

import (
	"fmt"
	"math/big"

	"github.com/Troublor/erebus-redgiant/analysis/data_flow"
	"github.com/ethereum/go-ethereum/core/vm"

	. "github.com/Troublor/erebus-redgiant/analysis/storage_address"
	engine "github.com/Troublor/erebus-redgiant/dyengine"
	. "github.com/Troublor/erebus-redgiant/dyengine/state"
	"github.com/Troublor/erebus-redgiant/helpers"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

//go:generate solc --bin --asm --abi --storage-layout ./__test__/contract.sol -o ./__test__/contract --overwrite
var _ = Describe("AddressingPathAnalyzer", func() {
	var bytecode []byte
	var state engine.State
	var contractAbi abi.ABI
	var exeVM *engine.ExeVM
	var vmContext *engine.VMContext

	var account common.Address
	var contract common.Address

	setupInitialState := func() {
		args, err := contractAbi.Pack("")
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
		contractAbi = helpers.LoadAbiFromFile("./__test__/contract/Contract.abi")

		state = NewMemoryState()
		exeVM = helpers.NewDebuggingExeVM()
		vmContext = helpers.DebuggingVMContext()

		setupInitialState()
	})

	It("should capture the single value addressing path", func() {
		called := false
		analyzer := &StorageAddressingPathAnalyzer{
			OnStorageStoredOrLoaded: func(op vm.OpCode, addressingPathCandidates []AddressingPath) {
				Expect(op).To(Equal(vm.SSTORE))
				Expect(len(addressingPathCandidates)).To(Equal(1))
				Expect(len(addressingPathCandidates[0])).To(Equal(2))
				called = true
			},
		}
		tracer := data_flow.NewDataFlowTracer(analyzer)
		exeVM.SetTracer(tracer)

		args, _ := contractAbi.Pack("setValue", big.NewInt(100))
		r, _, err := exeVM.DebuggingCall(state, vmContext, account, &contract, nil, args)
		Expect(err).To(BeNil())
		Expect(r.Failed()).To(BeFalse())

		Expect(called).To(BeTrue())
	})

	It("should capture the array value addressing path", func() {
		var paths []AddressingPath
		analyzer := &StorageAddressingPathAnalyzer{
			OnStorageStoredOrLoaded: func(op vm.OpCode, addressingPathCandidates []AddressingPath) {
				paths = append(paths, addressingPathCandidates...)
			},
		}
		tracer := data_flow.NewDataFlowTracer(analyzer)
		exeVM.SetTracer(tracer)

		args, _ := contractAbi.Pack("addAddress", account)
		r, _, err := exeVM.DebuggingCall(state, vmContext, account, &contract, nil, args)
		Expect(err).To(BeNil())
		Expect(r.Failed()).To(BeFalse())

		Expect(len(paths)).To(Equal(4))

		// read array.length
		Expect(paths[0].Opcode()).To(Equal(vm.SLOAD))
		Expect(len(paths[0])).To(Equal(2))
		Expect(paths[0].Seed()).To(Equal(common.HexToHash("0x0")))
		Expect(paths[0].Address()).To(Equal(common.HexToHash("0x0")))

		// write array.length
		Expect(paths[1].Opcode()).To(Equal(vm.SSTORE))
		Expect(len(paths[1])).To(Equal(2))
		Expect(paths[1].Seed()).To(Equal(common.HexToHash("0x0")))
		Expect(paths[1].Address()).To(Equal(common.HexToHash("0x0")))

		// read array[0]
		Expect(paths[2].Opcode()).To(Equal(vm.SLOAD))
		Expect(len(paths[2])).To(Equal(4))
		Expect(paths[2].Seed()).To(Equal(common.HexToHash("0x0")))
		Expect(paths[2].Address()).
			To(Equal(common.HexToHash("0x290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563")))

		// write array[0]
		Expect(paths[3].Opcode()).To(Equal(vm.SSTORE))
		Expect(len(paths[3])).To(Equal(4))
		Expect(paths[3].Seed()).To(Equal(common.HexToHash("0x0")))
		Expect(paths[3].Address()).
			To(Equal(common.HexToHash("0x290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563")))
	})

	It("should capture the mapping value addressing path", func() {
		var paths []AddressingPath
		analyzer := &StorageAddressingPathAnalyzer{
			OnStorageStoredOrLoaded: func(op vm.OpCode, addressingPathCandidates []AddressingPath) {
				paths = append(paths, addressingPathCandidates...)
			},
		}
		tracer := data_flow.NewDataFlowTracer(analyzer)
		exeVM.SetTracer(tracer)

		args, _ := contractAbi.Pack("setAddressValue", account, big.NewInt(100))
		r, _, err := exeVM.DebuggingCall(state, vmContext, account, &contract, nil, args)
		Expect(err).To(BeNil())
		Expect(r.Failed()).To(BeFalse())

		Expect(len(paths)).To(Equal(1))

		// write map[account]
		Expect(paths[0].Opcode()).To(Equal(vm.SSTORE))
		Expect(len(paths[0])).To(Equal(3))
		Expect(paths[0].Seed()).To(Equal(common.HexToHash("0x1")))
		Expect(paths[0].Address()).
			To(Equal(common.HexToHash("0xcc69885fda6bcc1a4ace058b4a62bf5e179ea78fd58a1ccd71c22cc9b688792f")))
	})

	It("should capture nested value addressing path", func() {
		var paths []AddressingPath
		analyzer := &StorageAddressingPathAnalyzer{
			OnStorageStoredOrLoaded: func(op vm.OpCode, addressingPathCandidates []AddressingPath) {
				paths = append(paths, addressingPathCandidates...)
			},
		}
		tracer := data_flow.NewDataFlowTracer(analyzer)
		exeVM.SetTracer(tracer)

		args, _ := contractAbi.Pack("setT", account, big.NewInt(100))
		r, _, err := exeVM.DebuggingCall(state, vmContext, account, &contract, nil, args)
		Expect(err).To(BeNil())
		Expect(r.Failed()).To(BeFalse())

		Expect(paths).To(HaveLen(14))
	})
})
