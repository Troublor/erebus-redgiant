package tracers

import (
	"github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
)

type StructLogConfig = logger.Config
type StructLog = logger.StructLog

type StructLogTracer struct {
	*logger.StructLogger
	TxStart func(tx *types.Transaction, vmContext *dyengine.VMContext, state dyengine.State)
	TxEnd   func(structLogs []StructLog, tx *types.Transaction,
		vmContext *dyengine.VMContext, state dyengine.State, receipt *types.Receipt)
}

func NewStructLogTracer(cfg StructLogConfig) *StructLogTracer {
	return &StructLogTracer{
		StructLogger: logger.NewStructLogger(&cfg),
	}
}

func (t *StructLogTracer) TransactionStart(tx *dyengine.Tx, context *dyengine.VMContext, state dyengine.State) {
	t.Reset()
	if t.TxStart != nil {
		t.TxStart(tx.Transaction, context, state)
	}
}

func (t *StructLogTracer) TransactionEnd(
	tx *dyengine.Tx,
	context *dyengine.VMContext, state dyengine.State,
	result *core.ExecutionResult, receipt *types.Receipt,
) {
	if t.TxEnd != nil {
		t.TxEnd(t.StructLogs(), tx.Transaction, context, state, receipt)
	}
}
