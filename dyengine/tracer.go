package dyengine

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

type TxTracer interface {
	TransactionStart(tx *Tx, vmContext *VMContext, state State)
	TransactionEnd(tx *Tx, vmContext *VMContext, state State, result *core.ExecutionResult, receipt *types.Receipt)

	CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte,
		gas uint64, value *big.Int)
	CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error)

	CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int)
	CaptureExit(output []byte, gasUsed uint64, err error)

	CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error)
	CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error)
}

type wrappedEVMLogger struct {
	inner TxTracer
}

func (l *wrappedEVMLogger) CaptureTxStart(gasLimit uint64) {
}

func (l *wrappedEVMLogger) CaptureTxEnd(restGas uint64) {
}

func (l *wrappedEVMLogger) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte,
	gas uint64, value *big.Int) {
	l.inner.CaptureStart(env, from, to, create, input, gas, value)
}

func (l *wrappedEVMLogger) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {
	l.inner.CaptureEnd(output, gasUsed, t, err)
}

func (l *wrappedEVMLogger) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64,
	value *big.Int) {
	l.inner.CaptureEnter(typ, from, to, input, gas, value)
}

func (l *wrappedEVMLogger) CaptureExit(output []byte, gasUsed uint64, err error) {
	l.inner.CaptureExit(output, gasUsed, err)
}

func (l *wrappedEVMLogger) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte,
	depth int, err error) {
	l.inner.CaptureState(pc, op, gas, cost, scope, rData, depth, err)
}

func (l *wrappedEVMLogger) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int,
	err error) {
	l.inner.CaptureFault(pc, op, gas, cost, scope, depth, err)
}
