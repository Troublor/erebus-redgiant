package summary

import (
	_ "embed"
	"math/big"
	"time"

	"github.com/Troublor/erebus-redgiant/dyengine/tracers"
	"github.com/Troublor/erebus-redgiant/helpers"

	engine "github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

// TxSummaryTracer collects a TxSummary of the transaction execution.
// The TxSummary includes:
// 1. Defs and Uses
// 2. profits.
type TxSummaryTracer struct {
	tracers.INestedCallTracer[*CallSummary]

	// input
	Config Config

	// output
	Summary *CallSummary

	state     engine.State
	currentTx *types.Transaction
	defs      map[string]StateVariable // the map of defs that is used to check whether a use is def-clear
}

func NewTxSummaryTracer(config Config) *TxSummaryTracer {
	var tracer tracers.INestedCallTracer[*CallSummary]
	if config.IncludeTrace {
		tracer = &tracers.NestedCallTracerWithTrace[*CallSummary]{}
	} else {
		tracer = &tracers.BasicNestedCallTracer[*CallSummary]{}
	}
	t := &TxSummaryTracer{
		INestedCallTracer: tracer,
		Config:            config,

		defs: make(map[string]StateVariable),
	}
	return t
}

func (t *TxSummaryTracer) CurrentMsgCall() *tracers.MsgCall[*CallSummary] {
	return t.INestedCallTracer.CurrentMsgCall()
}

func (t *TxSummaryTracer) CaptureStart(
	env *vm.EVM,
	from common.Address,
	to common.Address,
	create bool,
	input []byte,
	gas uint64,
	value *big.Int,
) {
	t.INestedCallTracer.CaptureStart(env, from, to, create, input, gas, value)
}

func (t *TxSummaryTracer) CaptureState(
	pc uint64,
	op vm.OpCode,
	gas, cost uint64,
	scope *vm.ScopeContext,
	rData []byte,
	depth int,
	err error,
) {
	t.INestedCallTracer.CaptureState(pc, op, gas, cost, scope, rData, depth, err)
	summary := t.CurrentMsgCall().InnerData

	if t.Config.IncludeDef {
		switch op {
		case vm.SSTORE:
			contract := scope.Contract.Address()
			key := common.BigToHash(scope.Stack.Back(0).ToBig())
			value := common.BigToHash(scope.Stack.Back(1).ToBig())
			def := StorageVariable{
				Address: contract,
				Storage: key,
				Value:   value,
				L:       t.CurrentMsgCall().CurrentLocation,
			}
			summary.currentData().addDefs(def)
			t.defs[def.ID()] = def
		}
	}
	if t.Config.IncludeUse {
		switch op {
		case vm.SLOAD:
			contract := scope.Contract.Address()
			key := common.BigToHash(scope.Stack.Back(0).ToBig())
			value := t.state.GetState(contract, key)
			use := StorageVariable{
				Address: contract,
				Storage: key,
				Value:   value,
				L:       t.CurrentMsgCall().CurrentLocation,
			}
			if _, defined := t.defs[use.ID()]; !defined {
				// only add def-clear uses
				summary.currentData().addUses(use)
			}
		case vm.BALANCE:
			account := common.BigToAddress(scope.Stack.Back(0).ToBig())
			balance := t.state.GetBalance(account)
			use := BalanceVariable{
				Address: account,
				Value:   balance,
				L:       t.CurrentMsgCall().CurrentLocation,
			}
			if _, defined := t.defs[use.ID()]; !defined {
				// only add def-clear uses
				summary.currentData().addUses(use)
			}
		case vm.SELFBALANCE:
			balance := t.state.GetBalance(scope.Contract.Address())
			use := BalanceVariable{
				Address: scope.Contract.Address(),
				Value:   balance,
				L:       t.CurrentMsgCall().CurrentLocation,
			}
			if _, defined := t.defs[use.ID()]; !defined {
				// only add def-clear uses
				summary.currentData().addUses(use)
			}
		case vm.CODESIZE:
			contract := *scope.Contract.CodeAddr
			size := t.state.GetCodeSize(contract)
			summary.currentData().addUses(CodeVariable{
				Address:  contract,
				Op:       op,
				CodeSize: size,
				L:        t.CurrentMsgCall().CurrentLocation,
			})
		case vm.CODECOPY:
			contract := *scope.Contract.CodeAddr
			code := t.state.GetCode(contract)
			use := CodeVariable{
				Address: contract,
				Op:      op,
				Code:    code,
				L:       t.CurrentMsgCall().CurrentLocation,
			}
			if _, defined := t.defs[use.ID()]; !defined {
				// only add def-clear uses
				summary.currentData().addUses(use)
			}
		case vm.EXTCODESIZE:
			contract := common.BigToAddress(scope.Stack.Back(0).ToBig())
			size := t.state.GetCodeSize(contract)
			use := CodeVariable{
				Address:  contract,
				Op:       op,
				CodeSize: size,
				L:        t.CurrentMsgCall().CurrentLocation,
			}
			if _, defined := t.defs[use.ID()]; !defined {
				// only add def-clear uses
				summary.currentData().addUses(use)
			}
		case vm.EXTCODECOPY:
			contract := common.BigToAddress(scope.Stack.Back(0).ToBig())
			code := t.state.GetCode(contract)
			use := CodeVariable{
				Address: contract,
				Op:      op,
				Code:    code,
				L:       t.CurrentMsgCall().CurrentLocation,
			}
			if _, defined := t.defs[use.ID()]; !defined {
				// only add def-clear uses
				summary.currentData().addUses(use)
			}
		case vm.EXTCODEHASH:
			contract := common.BigToAddress(scope.Stack.Back(0).ToBig())
			hash := t.state.GetCodeHash(contract)
			use := CodeVariable{
				Address:  contract,
				Op:       op,
				CodeHash: hash,
				L:        t.CurrentMsgCall().CurrentLocation,
			}
			if _, defined := t.defs[use.ID()]; !defined {
				// only add def-clear uses
				summary.currentData().addUses(use)
			}
		case vm.CALL:
			address := scope.Contract.Address()
			if scope.Stack.Back(2).ToBig().Sign() > 0 {
				// ether transfers reads the balance of the sender
				use := BalanceVariable{
					Address: address,
					Value:   t.state.GetBalance(address),
					L:       t.CurrentMsgCall().CurrentLocation,
				}
				if _, defined := t.defs[use.ID()]; !defined {
					// only add def-clear uses
					summary.currentData().addUses(use)
				}
			}
		}
	}
	if t.Config.IncludeTransfer || t.Config.IncludeProfit {
		switch op {
		case vm.LOG0, vm.LOG1, vm.LOG2, vm.LOG3, vm.LOG4:
			topics := make([]common.Hash, op-vm.LOG0)
			for i := 0; i < len(topics); i++ {
				topics[i] = common.BigToHash(scope.Stack.Back(2 + i).ToBig())
			}
			dataOffset := scope.Stack.Back(0).ToBig().Int64()
			dataLength := scope.Stack.Back(1).ToBig().Int64()
			log := &types.Log{
				TxHash:  t.currentTx.Hash(),
				Topics:  topics,
				Address: scope.Contract.Address(),
				Data:    helpers.GetMemoryCopyWithPadding(scope.Memory, dataOffset, dataLength),
			}
			location := t.CurrentMsgCall().CurrentLocation
			transfers, err := LogToTransfers(log, location)
			if err == nil {
				summary.currentData().addTransfers(transfers...)
				if t.Config.IncludeProfit {
					for _, transfer := range transfers {
						summary.currentData().addProfits(transfer.Profits()...)
					}
				}
			}
		}
	}
}

func (t *TxSummaryTracer) CaptureEnter(
	typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int,
) {
	parentMsgCall := t.CurrentMsgCall()
	parentSummary := parentMsgCall.InnerData
	t.INestedCallTracer.CaptureEnter(typ, from, to, input, gas, value)
	currentMsgCall := t.CurrentMsgCall()
	summary := newCallSummary(t.state.Copy(), currentMsgCall)
	parentSummary.addNestedSummary()
	currentMsgCall.InnerData = summary
	if value != nil && value.Cmp(big.NewInt(0)) > 0 {
		if t.Config.IncludeDef {
			// value transferGraphNode  defines
			def0 := BalanceVariable{
				Address: to,
				Value:   new(big.Int).Add(t.state.GetBalance(to), value),
				L:       parentMsgCall.CurrentLocation,
			}
			summary.currentData().addDefs(def0)
			t.defs[def0.ID()] = def0
			def1 := BalanceVariable{
				Address: from,
				Value:   new(big.Int).Sub(t.state.GetBalance(from), value),
				L:       parentMsgCall.CurrentLocation,
			}
			summary.currentData().addDefs(def1)
			t.defs[def1.ID()] = def1
		}

		var transfer ITransfer
		if t.Config.IncludeTransfer || t.Config.IncludeProfit {
			// collect Transfer
			transfer = EtherTransfer{
				fungibleTransfer{
					FromAccount:      from,
					ToAccount:        to,
					Amount:           value,
					ContractLocation: parentMsgCall.CurrentLocation,
				},
			}
			summary.currentData().addTransfers(transfer)
		}

		if t.Config.IncludeProfit {
			// if there is ether transferGraphNode, record a positive profit for receiver
			// and a negative profit for sender
			summary.currentData().addProfits(transfer.Profits()...)
		}
	}
}

func (t *TxSummaryTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	callSummary := t.CurrentMsgCall().InnerData
	if t.Config.IncludeTrace {
		for i, trace := range callSummary.BlockPaths() {
			callSummary.Data[i].ExecutionPath = trace
		}
	}
	if err != nil {
		// if the message call reverts, clear all Defs, Transfers and Profits
		callSummary.currentData().clearChanges()
	}
	t.INestedCallTracer.CaptureExit(output, gasUsed, err)
}

func (t *TxSummaryTracer) CaptureEnd(output []byte, gasUsed uint64, tt time.Duration, err error) {
	t.INestedCallTracer.CaptureEnd(output, gasUsed, tt, err)
}

func (t *TxSummaryTracer) TransactionStart(
	tx *engine.Tx,
	vmContext *engine.VMContext,
	state engine.State,
) {
	t.INestedCallTracer.TransactionStart(tx, vmContext, state)
	t.state = state
	t.currentTx = tx.Transaction
	summary := newCallSummary(state.Copy(), t.CurrentMsgCall())
	t.CurrentMsgCall().InnerData = summary
}

func (t *TxSummaryTracer) TransactionEnd(
	tx *engine.Tx, vmContext *engine.VMContext, state engine.State,
	result *core.ExecutionResult, receipt *types.Receipt,
) {
	t.Summary = t.CurrentMsgCall().InnerData
	if t.Config.IncludeTrace {
		for i, trace := range t.Summary.BlockPaths() {
			t.Summary.Data[i].ExecutionPath = trace
		}
	}
	if receipt.Status <= 0 {
		// if the transaction reverts, we only preserve Uses in the summary
		t.Summary.currentData().clearChanges()
	} else {
		if tx.Value().Cmp(big.NewInt(0)) > 0 {
			var to common.Address
			if tx.To() == nil {
				to = receipt.ContractAddress
			} else {
				to = *tx.To()
			}

			if t.Config.IncludeUse {
				// ether transfer also reads the balance of the sender
				use := BalanceVariable{
					Address: tx.From(),
					Value:   new(big.Int).Sub(t.state.GetBalance(tx.From()), tx.Value()),
				}
				if _, defined := t.defs[use.ID()]; !defined {
					// only add def-clear uses
					t.Summary.currentData().addUses(use)
				}
			}
			if t.Config.IncludeDef {
				// ether transferGraphNode is also Defs
				def0 := BalanceVariable{
					Address: tx.From(),
					Value:   state.GetBalance(tx.From()),
					L:       nil,
				}
				t.Summary.currentData().addDefs(def0)
				t.defs[def0.ID()] = def0
				def1 := BalanceVariable{
					Address: to,
					Value:   state.GetBalance(to),
					L:       nil,
				}
				t.Summary.currentData().addDefs(def1)
				t.defs[def1.ID()] = def1
			}

			var transfer ITransfer
			if t.Config.IncludeTransfer || t.Config.IncludeProfit {
				// collect Transfer
				transfer = EtherTransfer{
					fungibleTransfer{
						FromAccount:      tx.From(),
						ToAccount:        to,
						Amount:           tx.Value(),
						ContractLocation: nil,
					},
				}
				t.Summary.currentData().addTransfers(transfer)
			}

			if t.Config.IncludeProfit {
				// collect Ether profit
				t.Summary.currentData().addProfits(transfer.Profits()...)
			}
		}
	}
	t.INestedCallTracer.TransactionEnd(tx, vmContext, state, result, receipt)
}
