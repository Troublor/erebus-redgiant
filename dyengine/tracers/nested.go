package tracers

import (
	"math/big"
	"time"

	"github.com/Troublor/erebus-redgiant/helpers"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/samber/lo"
)

type INestedCallTracer[D IMsgCallData] interface {
	dyengine.TxTracer
	CurrentMsgCall() *MsgCall[D]
	RootMsgCall() *MsgCall[D]
}

type BasicNestedCallTracer[D IMsgCallData] struct {
	msgCallStack    []*MsgCall[D]
	totalOperations uint

	MsgCallDataFactory func(*MsgCall[D]) D
}

func (t *BasicNestedCallTracer[D]) CurrentMsgCall() *MsgCall[D] {
	if len(t.msgCallStack) == 0 {
		return nil
	}
	return t.msgCallStack[len(t.msgCallStack)-1]
}

func (t *BasicNestedCallTracer[D]) RootMsgCall() *MsgCall[D] {
	if len(t.msgCallStack) == 0 {
		return nil
	}
	return t.msgCallStack[0]
}

func (t *BasicNestedCallTracer[_]) CaptureStart(
	*vm.EVM,
	common.Address,
	common.Address,
	bool,
	[]byte,
	uint64,
	*big.Int,
) {
}

func (t *BasicNestedCallTracer[D]) CaptureState(
	pc uint64, op vm.OpCode, gas uint64, cost uint64, _ *vm.ScopeContext, _ []byte, _ int, _ error,
) {
	currentCall := t.CurrentMsgCall()
	t.CurrentMsgCall().CurrentLocation = location{
		pos:          currentCall.Position,
		codeAddr:     currentCall.CodeAddr,
		pc:           pc,
		opcode:       op,
		gasAvailable: gas,
		gasUsed:      cost,
		index:        t.totalOperations,
	}
	t.totalOperations++
}

func (t *BasicNestedCallTracer[D]) CaptureEnter(
	typ vm.OpCode, from common.Address, to common.Address,
	input []byte, _ uint64, value *big.Int,
) {
	var contextAddr common.Address
	switch typ {
	case vm.CALL, vm.CREATE, vm.CREATE2, vm.STATICCALL:
		contextAddr = to
	case vm.CALLCODE, vm.DELEGATECALL:
		contextAddr = from
	}
	cpInput := make([]byte, len(input))
	copy(cpInput, input)
	cpValue := new(big.Int)
	if value != nil {
		cpValue.Set(value)
	}
	isPrecompiled := helpers.IsPrecompiledContract(to)
	parentCall := t.msgCallStack[len(t.msgCallStack)-1]
	callPosition := t.CurrentMsgCall().GenChildPosition(isPrecompiled)
	msgCall := &MsgCall[D]{
		Position: callPosition,

		OpCode: typ,
		Caller: MsgCallCaller{
			CodeAddr:  lo.ToPtr(parentCall.CodeAddr),
			StateAddr: parentCall.StateAddr,
			CallSite:  parentCall.CurrentLocation,
		},
		Precompiled: isPrecompiled,
		CodeAddr:    to,
		StateAddr:   contextAddr,
		Input:       cpInput,
		Value:       cpValue,

		parent: parentCall,
	}
	if t.MsgCallDataFactory != nil {
		msgCall.InnerData = t.MsgCallDataFactory(msgCall)
	}
	parentCall.NestedCalls = append(parentCall.NestedCalls, msgCall)
	t.msgCallStack = append(t.msgCallStack, msgCall)
}

func (t *BasicNestedCallTracer[_]) CaptureExit(output []byte, gasUsed uint64, err error) {
	call := t.msgCallStack[len(t.msgCallStack)-1]
	call.Result = &core.ExecutionResult{
		Err:        err,
		UsedGas:    gasUsed,
		ReturnData: output,
	}
	t.msgCallStack = t.msgCallStack[:len(t.msgCallStack)-1]
}

func (t *BasicNestedCallTracer[_]) CaptureFault(
	uint64,
	vm.OpCode,
	uint64,
	uint64,
	*vm.ScopeContext,
	int,
	error,
) {
}

func (t *BasicNestedCallTracer[_]) CaptureEnd([]byte, uint64, time.Duration, error) {
}

func (t *BasicNestedCallTracer[D]) TransactionStart(
	tx *dyengine.Tx,
	_ *dyengine.VMContext,
	state dyengine.State,
) {
	// root message call
	from := tx.From()
	to := tx.To()
	var contract common.Address
	var opcode vm.OpCode
	if to != nil {
		contract = *to
		opcode = vm.CALL
	} else {
		// we compute the created contract address in advance
		contract = crypto.CreateAddress(from, state.GetNonce(from))
		opcode = vm.CREATE
	}
	call := &MsgCall[D]{
		OpCode: opcode,
		Caller: MsgCallCaller{
			CodeAddr:  nil,
			StateAddr: from,
			CallSite:  nil,
		},
		Precompiled: helpers.IsPrecompiledContract(contract),
		CodeAddr:    contract,
		StateAddr:   contract,
		Input:       tx.Data(),
		Value:       tx.Value(),
	}
	if t.MsgCallDataFactory != nil {
		call.InnerData = t.MsgCallDataFactory(call)
	}
	t.msgCallStack = append(t.msgCallStack, call)
}

func (t *BasicNestedCallTracer[_]) TransactionEnd(
	_ *dyengine.Tx, _ *dyengine.VMContext, _ dyengine.State,
	result *core.ExecutionResult, receipt *types.Receipt,
) {
	call := t.msgCallStack[len(t.msgCallStack)-1]
	call.Result = result
	call.Receipt = receipt
}

type IDataWithTrace interface {
	IMsgCallData
	lastLocation() TraceLocation
	setLastLocation(TraceLocation)
	currentBasicBlock() *traceBlock
	setCurrentBasicBlock(*traceBlock)
	appendBasicBlock(IBasicBlock[TraceLocation])
	nextTrace()

	BlockPaths() []Trace
}

type DataWithTrace struct {
	loc TraceLocation
	blk *traceBlock

	// the execution trace is split to n+1 parts, where n is the number of nested calls
	// each part is a Blocks which is a slice of IBasicBlock
	paths []Trace
}

func NewDataWithTrace() *DataWithTrace {
	return &DataWithTrace{
		paths: []Trace{{}},
	}
}

func (d *DataWithTrace) lastLocation() TraceLocation {
	return d.loc
}

func (d *DataWithTrace) setLastLocation(l TraceLocation) {
	d.loc = l
}

func (d *DataWithTrace) currentBasicBlock() *traceBlock {
	return d.blk
}

func (d *DataWithTrace) setCurrentBasicBlock(b *traceBlock) {
	d.blk = b
}

func (d *DataWithTrace) BlockPaths() []Trace {
	return d.paths
}

func (d *DataWithTrace) appendBasicBlock(b IBasicBlock[TraceLocation]) {
	d.paths[len(d.paths)-1] = append(d.paths[len(d.paths)-1], b)
}

func (d *DataWithTrace) nextTrace() {
	d.paths = append(d.paths, Trace{})
}

type NestedCallTracerWithTrace[D IDataWithTrace] struct {
	BasicNestedCallTracer[D]
}

func (t *NestedCallTracerWithTrace[D]) CaptureState(
	pc uint64,
	op vm.OpCode,
	gas uint64,
	cost uint64,
	scope *vm.ScopeContext,
	rData []byte,
	depth int,
	err error,
) {
	t.BasicNestedCallTracer.CaptureState(pc, op, gas, cost, scope, rData, depth, err)
	currentCall := t.CurrentMsgCall()
	currentLoc := currentCall.CurrentLocation
	if err != nil {
		// if some errors happens before the opcode is executed.
		// such errors can be ErrOutOfGas, ErrStackOverflow, etc.
		// the current msgCall will terminate, we need to close our execution basic block.
		if currentCall.InnerData.currentBasicBlock() != nil {
			// complete the current basic block
			currentCall.InnerData.currentBasicBlock().content = append(
				currentCall.InnerData.currentBasicBlock().content,
				currentLoc,
			)
			currentCall.InnerData.appendBasicBlock(currentCall.InnerData.currentBasicBlock())
			// set the next block to nil
			currentCall.InnerData.setCurrentBasicBlock(nil)
		}

		// if err happens before the opcode is executed,
		// def, use, transfer, profit do not matter anymore, we directly return.
		return
	}

	defer func() {
		currentCall.InnerData.setLastLocation(currentLoc)
	}()

	if IsBasicBlockHead(currentCall.InnerData.lastLocation(), currentLoc) {
		nextBlock := &traceBlock{
			content:   []TraceLocation{currentLoc},
			previous:  currentCall.InnerData.currentBasicBlock(),
			stateAddr: scope.Contract.Address(),
			codeAddr:  *scope.Contract.CodeAddr,
		}

		if currentCall.InnerData.currentBasicBlock() != nil {
			// complete the current basic block
			currentCall.InnerData.appendBasicBlock(currentCall.InnerData.currentBasicBlock())
			currentCall.InnerData.currentBasicBlock().next = nextBlock
		}

		// and start a new basic block
		currentCall.InnerData.setCurrentBasicBlock(nextBlock)
	} else if IsBasicBlockTail(currentLoc) {
		if currentCall.InnerData.currentBasicBlock() != nil {
			// complete the current basic block
			currentCall.InnerData.currentBasicBlock().content = append(
				currentCall.InnerData.currentBasicBlock().content,
				currentLoc,
			)
			currentCall.InnerData.appendBasicBlock(currentCall.InnerData.currentBasicBlock())
			// set the next block to nil
			currentCall.InnerData.setCurrentBasicBlock(nil)
		} else {
			panic("unexpected basic block tail")
		}
	} else {
		if currentCall.InnerData.currentBasicBlock() == nil {
			panic("unexpected opcode outside of basic block")
		}
		currentCall.InnerData.currentBasicBlock().content = append(
			currentCall.InnerData.currentBasicBlock().content,
			currentLoc,
		)
	}
}

func (t *NestedCallTracerWithTrace[D]) CaptureEnter(
	typ vm.OpCode, from common.Address, to common.Address,
	input []byte, gas uint64, value *big.Int,
) {
	t.CurrentMsgCall().InnerData.nextTrace()
	t.BasicNestedCallTracer.CaptureEnter(typ, from, to, input, gas, value)
}
