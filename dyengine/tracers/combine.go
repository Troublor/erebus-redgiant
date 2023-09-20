package tracers

import (
	"math/big"
	"time"

	"github.com/Troublor/erebus-redgiant/dyengine"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

type CombinedTracer struct {
	Tracers []dyengine.TxTracer
}

func CombineTracers(tracers ...dyengine.TxTracer) *CombinedTracer {
	return &CombinedTracer{Tracers: tracers}
}

func (t *CombinedTracer) CaptureEnter(
	typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int,
) {
	for _, tr := range t.Tracers {
		tr.CaptureEnter(typ, from, to, input, gas, value)
	}
}

func (t *CombinedTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	for _, tr := range t.Tracers {
		tr.CaptureExit(output, gasUsed, err)
	}
}

func (t *CombinedTracer) CaptureStart(
	env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int,
) {
	for _, tr := range t.Tracers {
		tr.CaptureStart(env, from, to, create, input, gas, value)
	}
}

func (t *CombinedTracer) CaptureState(
	pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error,
) {
	for _, tr := range t.Tracers {
		tr.CaptureState(pc, op, gas, cost, scope, rData, depth, err)
	}
}

func (t *CombinedTracer) CaptureFault(
	pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error,
) {
	for _, tr := range t.Tracers {
		tr.CaptureFault(pc, op, gas, cost, scope, depth, err)
	}
}

func (t *CombinedTracer) CaptureEnd(output []byte, gasUsed uint64, tt time.Duration, err error) {
	for _, tr := range t.Tracers {
		tr.CaptureEnd(output, gasUsed, tt, err)
	}
}

func (t *CombinedTracer) TransactionStart(transaction *dyengine.Tx, context *dyengine.VMContext, state dyengine.State) {
	for _, tr := range t.Tracers {
		tr.TransactionStart(transaction, context, state)
	}
}

func (t *CombinedTracer) TransactionEnd(
	transaction *dyengine.Tx, context *dyengine.VMContext, state dyengine.State,
	result *core.ExecutionResult, receipt *types.Receipt,
) {
	for _, tr := range t.Tracers {
		tr.TransactionEnd(transaction, context, state, result, receipt)
	}
}
