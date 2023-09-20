package tracers

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core"

	"github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

type NoopTracer struct{}

func (t *NoopTracer) CaptureStart(*vm.EVM, common.Address, common.Address, bool, []byte, uint64, *big.Int) {
}

func (t *NoopTracer) CaptureState(*vm.EVM, uint64, vm.OpCode, uint64, uint64, *vm.ScopeContext, []byte, int, error) {
}

func (t *NoopTracer) CaptureFault(*vm.EVM, uint64, vm.OpCode, uint64, uint64, *vm.ScopeContext, int, error) {
}

func (t *NoopTracer) CaptureEnter(vm.OpCode, common.Address, common.Address, []byte, uint64, *big.Int) {
}

func (t *NoopTracer) CaptureExit([]byte, uint64, error) {
}

func (t *NoopTracer) CaptureEnd([]byte, uint64, time.Duration, error) {
}

func (t *NoopTracer) TransactionStart(*dyengine.Tx, *dyengine.VMContext, dyengine.State) {
}

func (t *NoopTracer) TransactionEnd(
	*dyengine.Tx, *dyengine.VMContext, dyengine.State, *core.ExecutionResult, *types.Receipt,
) {
}
