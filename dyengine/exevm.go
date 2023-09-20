package dyengine

import (
	"fmt"
	"math/big"
	"runtime/debug"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type VMConfig struct {
	*params.ChainConfig

	NoBaseFee                 bool `json:"noBaseFee"`
	BypassNonceAndSenderCheck bool `json:"bypassNonceAndSenderCheck"`
	CapGasToBlockLimit        bool `json:"capGasToBlockLimit"`
	ForceZeroGasPrice         bool `json:"forceZeroGasPrice"`
	RegulateBaseFee           bool `json:"regulateBaseFee"` // adjust tx base fee if it is less than block base fee

	ExtraEips []int `json:"extraEips"`
}

type VMContext struct {
	vm.BlockContext
	*core.GasPool
	BlockHash        common.Hash
	GasUsed          uint64
	TransactionIndex uint
}

func (c *VMContext) Copy() *VMContext {
	var gasPool core.GasPool
	gasPool.AddGas(c.GasPool.Gas())
	return &VMContext{
		BlockContext:     c.BlockContext,
		GasPool:          &gasPool,
		BlockHash:        c.BlockHash,
		GasUsed:          c.GasUsed,
		TransactionIndex: c.TransactionIndex,
	}
}

type ExeVM struct {
	Config VMConfig
	Tracer TxTracer
}

func NewExeVM(config VMConfig) *ExeVM {
	return &ExeVM{
		Config: config,
	}
}

func (m *ExeVM) SetTracer(tracer TxTracer) {
	m.Tracer = tracer
}

func (m *ExeVM) BypassNonceAndSenderCheck(bypass bool) {
	m.Config.BypassNonceAndSenderCheck = bypass
}

func (m *ExeVM) ApplyTransaction(
	state State, tx *types.Transaction, vmContext *VMContext, commit bool, genReceipt bool,
) (*core.ExecutionResult, *types.Receipt, error) {
	signer := types.MakeSigner(m.Config.ChainConfig, vmContext.BlockNumber)
	t, err := TxFromTransactionWithSigner(tx, signer)
	if err != nil {
		return nil, nil, err
	}
	return m.ApplyTx(state, t, vmContext, commit, genReceipt)
}

type txExecutionError struct {
	msg   string
	stack []byte
}

func (e txExecutionError) Error() string {
	return fmt.Sprintf("TxExecutionError: %v: %s", e.msg, e.stack)
}

func (e txExecutionError) Msg() string {
	return e.msg
}

func (e txExecutionError) Stack() string {
	return string(e.stack)
}

func (m *ExeVM) ApplyTx(
	state State, tx *Tx, vmContext *VMContext, commit bool, genReceipt bool,
) (msgResult *core.ExecutionResult, receipt *types.Receipt, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = txExecutionError{
				msg:   fmt.Sprintf("%v", r),
				stack: debug.Stack(),
			}
		}
	}()

	// execute pseudo transaction
	if tx.IsPseudo() {
		var returnData []byte
		var err error
		if tx.PseudoExec != nil {
			returnData, err = tx.PseudoExec(state)
		}
		return &core.ExecutionResult{
			ReturnData: returnData,
			Err:        err,
		}, nil, nil
	}

	// prepare BlockContext
	vmContext.BlockContext.GetHash = state.GetHashFn()

	// prepare vm.Config
	evmConfig := vm.Config{
		Tracer: &wrappedEVMLogger{m.Tracer},
		Debug:  m.Tracer != nil,

		NoBaseFee: m.Config.NoBaseFee,

		ExtraEips: m.Config.ExtraEips,
	}

	// prepare TxContext
	var msg types.Message
	if tx.Signed() {
		// if the transaction is signed
		signer := types.MakeSigner(m.Config.ChainConfig, vmContext.BlockNumber)
		oriMsg, err := tx.AsMessage(signer, vmContext.BaseFee)
		if err != nil {
			return nil, nil, err
		}
		// create a fake message based on original message.
		// this is meant to bypass nonce check.
		gas := oriMsg.Gas()
		if m.Config.CapGasToBlockLimit && gas > vmContext.GasPool.Gas() {
			gas = vmContext.GasPool.Gas()
		}
		gasPrice := oriMsg.GasPrice()
		if m.Config.ForceZeroGasPrice {
			gasPrice = big.NewInt(0)
		}
		feeCap := oriMsg.GasFeeCap()
		tipCap := oriMsg.GasTipCap()
		if m.Config.NoBaseFee {
			feeCap = big.NewInt(0)
			tipCap = big.NewInt(0)
		}
		if m.Config.RegulateBaseFee && vmContext.BaseFee != nil {
			if feeCap.Cmp(vmContext.BaseFee) < 0 {
				feeCap = vmContext.BaseFee
			}
		}
		msg = types.NewMessage(
			oriMsg.From(),
			oriMsg.To(),
			oriMsg.Nonce(),
			oriMsg.Value(),
			gas,
			gasPrice,
			feeCap,
			tipCap,
			oriMsg.Data(),
			oriMsg.AccessList(),
			m.Config.BypassNonceAndSenderCheck,
		)
	} else {
		// for unsigned transactions, we still accept it but set its 'From' as zero address.
		gas := tx.Gas()
		if m.Config.CapGasToBlockLimit && gas > vmContext.GasPool.Gas() {
			gas = vmContext.GasPool.Gas()
		}
		gasPrice := tx.GasPrice()
		if m.Config.ForceZeroGasPrice {
			gasPrice = big.NewInt(0)
		}
		feeCap := tx.GasFeeCap()
		if m.Config.NoBaseFee {
			feeCap = big.NewInt(0)
		}
		tipCap := tx.GasTipCap()
		if m.Config.NoBaseFee {
			tipCap = big.NewInt(0)
		}
		if m.Config.RegulateBaseFee && vmContext.BaseFee != nil {
			if feeCap.Cmp(vmContext.BaseFee) < 0 {
				feeCap = vmContext.BaseFee
			}
		}
		msg = types.NewMessage(
			tx.From(),
			tx.To(),
			tx.Nonce(),
			tx.Value(),
			gas,
			gasPrice,
			feeCap,
			tipCap,
			tx.Data(),
			tx.AccessList(),
			m.Config.BypassNonceAndSenderCheck,
		)
	}
	state.Prepare(tx.Hash(), int(vmContext.TransactionIndex))
	txContext := core.NewEVMTxContext(msg)

	// initiate EVM
	evm := vm.NewEVM(vmContext.BlockContext, txContext, state, m.Config.ChainConfig, evmConfig)

	// apply transaction
	snapshot := state.Snapshot()
	if m.Tracer != nil {
		m.Tracer.TransactionStart(tx, vmContext, state)
	}
	msgResult, err = core.ApplyMessage(evm, msg, vmContext.GasPool)
	if err != nil {
		state.RevertToSnapshot(snapshot)
		return msgResult, nil, err
	}
	// check whether there are errors in ForkedState
	// FIXME this is very imprecise since we don't know where the error is thrown
	if err := state.LastError(); err != nil {
		state.RevertToSnapshot(snapshot)
		return msgResult, nil, err
	}

	vmContext.GasUsed += msgResult.UsedGas

	var root []byte
	if m.Config.ChainConfig.IsByzantium(vmContext.BlockNumber) {
		state.Finalise(true)
	} else {
		root = state.IntermediateRoot(m.Config.ChainConfig.IsEIP158(vmContext.BlockNumber)).Bytes()
	}

	// short circuit if we don't need to generate receipt
	if genReceipt {
		// Create a new receipt for the transaction, storing the intermediate root and
		// gas used by the tx.
		receipt = &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: vmContext.GasUsed}
		if msgResult.Failed() {
			receipt.Status = types.ReceiptStatusFailed
		} else {
			receipt.Status = types.ReceiptStatusSuccessful
		}
		receipt.TxHash = tx.Hash()
		receipt.GasUsed = msgResult.UsedGas

		// If the transaction created a contract, store the creation address in the receipt.
		if msg.To() == nil {
			receipt.ContractAddress = crypto.CreateAddress(txContext.Origin, tx.Nonce())
		}

		// Set the receipt logs and create the bloom filter.
		receipt.Logs = state.GetLogs(tx.Hash(), vmContext.BlockHash)
		if receipt.Logs == nil {
			receipt.Logs = make([]*types.Log, 0)
		}
		receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
		// These three are non-consensus fields:
		// receipt.BlockHash
		// receipt.BlockNumber
		receipt.TransactionIndex = vmContext.TransactionIndex

		receipt.BlockHash = vmContext.BlockHash
		receipt.BlockNumber = vmContext.BlockNumber
		for _, l := range receipt.Logs {
			if l.Data == nil {
				l.Data = []byte{}
			}
		}
	}
	vmContext.TransactionIndex += 1

	if commit {
		// Commit block
		_, err = state.Commit(m.Config.ChainConfig.IsEIP158(vmContext.BlockNumber))
		if err != nil {
			return nil, nil, fmt.Errorf("could not commit state: %v", err)
		}
	}

	if m.Tracer != nil {
		m.Tracer.TransactionEnd(tx, vmContext, state, msgResult, receipt)
	}

	return msgResult, receipt, nil
}

func (m *ExeVM) ApplyTransactions(
	state State, txs types.Transactions, vmContext *VMContext, commit bool, genReceipt bool,
) (includedTxs types.Transactions, receipts types.Receipts, rejectedTxs []*rejectedTx, err error) {
	for i, tx := range txs {
		signer := types.MakeSigner(m.Config.ChainConfig, vmContext.BlockNumber)
		msg, err := tx.AsMessage(signer, vmContext.BaseFee)
		if err != nil {
			rejectedTxs = append(rejectedTxs, &rejectedTx{i, err.Error()})
			log.Info("rejected tx", "index", i, "hash", tx.Hash(), "error", err)
			continue
		}
		_, receipt, err := m.ApplyTransaction(state, tx, vmContext, false, genReceipt)
		if err != nil {
			rejectedTxs = append(rejectedTxs, &rejectedTx{i, err.Error()})
			log.Info("rejected tx", "index", i, "hash", tx.Hash(), "from", msg.From(), "error", err)
			continue
		}
		includedTxs = append(includedTxs, tx)
		if genReceipt {
			receipts = append(receipts, receipt)
		}
	}
	state.IntermediateRoot(m.Config.ChainConfig.IsEIP158(vmContext.BlockNumber))
	// TODO mining reward is not implemented. See go-ethereum/cmd/evm/internal/t8ntool/execution.go:213

	if commit {
		// Commit block
		_, err = state.Commit(m.Config.ChainConfig.IsEIP158(vmContext.BlockNumber))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("could not commit state: %v", err)
		}
	}
	return includedTxs, receipts, rejectedTxs, nil
}

// DebuggingCall execute EVM with minimal information (from, to, value, data) and get the execution result..
// DebuggingCall is mainly used in testing to prepare the VM state by deploy/invoking contracts in the state.
// DebuggingCall usually used together with MemoryState.
func (m *ExeVM) DebuggingCall(
	state State, vmContext *VMContext, from common.Address, to *common.Address, value *big.Int, data []byte,
) (*core.ExecutionResult, *types.Receipt, error) {
	tx := NewTx(from, &types.LegacyTx{
		To:    to,
		Value: value,
		Data:  data,
		Gas:   vmContext.GasPool.Gas(),
		Nonce: state.GetNonce(from),
	})
	return m.ApplyTx(state, tx, vmContext, true, true)
}

type rejectedTx struct {
	Index int    `json:"index"`
	Err   string `json:"error"`
}
