package data_flow

import (
	"math/big"
	"time"

	"github.com/holiman/uint256"

	"github.com/Troublor/erebus-redgiant/helpers"

	engine "github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/Troublor/erebus-redgiant/dyengine/tracers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

type AfterOperationHook func(scope *vm.ScopeContext)

type TrackerCollection struct {
	Stack     *stack
	Memory    *memory
	Storage   *storage
	Chain     *chain
	Call      *call
	ChildCall *call
}

type Data struct {
	// each analyzer has its own collection of trackers
	trackerCollections map[Analyzer]*TrackerCollection
	// each analyzer has its own callback which will be called after the operation is executed
	lastStateCallbacks map[Analyzer]AfterOperationHook
}

type DataFlowTracer struct {
	*tracers.BasicNestedCallTracer[*Data]

	analyzers map[Analyzer]FlowPolicy // each analyzer has its own policy

	globalStorages map[Analyzer]map[common.Address]*storage
	globalChains   map[Analyzer]*chain
}

func NewDataFlowTracer(analyzers ...Analyzer) *DataFlowTracer {
	policies := make(map[Analyzer]FlowPolicy)
	for _, analyzer := range analyzers {
		policies[analyzer] = analyzer.FlowPolicy()
	}
	return &DataFlowTracer{
		BasicNestedCallTracer: &tracers.BasicNestedCallTracer[*Data]{},
		analyzers:             policies,
	}
}

func (t *DataFlowTracer) CaptureStart(
	env *vm.EVM, from common.Address, to common.Address, create bool,
	input []byte, gas uint64, value *big.Int,
) {
	t.BasicNestedCallTracer.CaptureStart(env, from, to, create, input, gas, value)
}

func (t *DataFlowTracer) CaptureState(
	pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext,
	rData []byte, depth int, err error,
) {
	pops, pushes := helpers.GetStackEffects(op)
	currentMessageCall := t.CurrentMsgCall()

	// feed the stack results to the callbacks of last operation for all trackers.
	innerData := currentMessageCall.InnerData
	for analyzer := range t.analyzers {
		if innerData.lastStateCallbacks[analyzer] != nil {
			innerData.lastStateCallbacks[analyzer](scope)
			innerData.lastStateCallbacks[analyzer] = nil
		}
	}

	//helpers.SanityCheck(func() bool {
	//	for analyzer := range t.analyzers {
	//		assert := len(scope.Stack.Data()) == innerData.trackerCollections[analyzer].Stack.Depth()
	//		if !assert {
	//			return false
	//		}
	//	}
	//	return true
	//})

	// call super method
	t.BasicNestedCallTracer.CaptureState(pc, op, gas, cost, scope, rData, depth, err)

	// construct current operation
	args := make([]Operand, pops)
	for i := 0; i < pops; i++ {
		v := scope.Stack.Back(i)
		cpV := new(uint256.Int).SetBytes(v.Bytes())
		args[pops-i-1] = Operand{cpV}
	}
	operation := &Operation{
		TraceLocation: t.CurrentMsgCall().CurrentLocation,
		msgCall:       t.CurrentMsgCall(),
		args:          args,
	}

	for a, flowPolicy := range t.analyzers {
		analyzer := a
		// call trackers' BeforeOperation hook
		// pop arguments from mirroring stack
		stackArgs := innerData.trackerCollections[analyzer].Stack.pop(pops)
		stackResults := make([]FlowNode, pushes)

		policy := flowPolicy[op]
		callback := policy(analyzer, scope, innerData.trackerCollections[analyzer], operation, stackArgs, stackResults)

		// register callback which will be called after the operation is executed (before next operation)
		innerData.lastStateCallbacks[analyzer] = func(scope *vm.ScopeContext) {
			if callback != nil {
				callback(scope)
			}
			innerData.trackerCollections[analyzer].Stack.push(stackResults...)
		}
	}
}

func (t *DataFlowTracer) CaptureEnter(
	typ vm.OpCode, from common.Address, to common.Address,
	input []byte, gas uint64, value *big.Int,
) {
	t.BasicNestedCallTracer.CaptureEnter(typ, from, to, input, gas, value)

	collections := make(map[Analyzer]*TrackerCollection)
	for analyzer := range t.analyzers {
		callData := t.CurrentMsgCall().Parent().InnerData.trackerCollections[analyzer].ChildCall
		if callData == nil || !callData.matchMsgCall(t.CurrentMsgCall()) {
			callData = newCall(t.CurrentMsgCall().Parent().GenChildPosition(helpers.IsPrecompiledContract(to)))
		}
		var storage *storage
		var ok bool
		if storage, ok = t.globalStorages[analyzer][t.CurrentMsgCall().StateAddr]; !ok {
			storage = newStorage()
			t.globalStorages[analyzer][t.CurrentMsgCall().StateAddr] = storage
		}
		collections[analyzer] = &TrackerCollection{
			Call:    callData,
			Stack:   newStack(),
			Memory:  newMemory(),
			Storage: storage,
			Chain:   t.globalChains[analyzer],
		}
	}

	t.CurrentMsgCall().InnerData = &Data{
		trackerCollections: collections,
		lastStateCallbacks: map[Analyzer]AfterOperationHook{},
	}
}

func (t *DataFlowTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	t.BasicNestedCallTracer.CaptureExit(output, gasUsed, err)
}

func (t *DataFlowTracer) CaptureFault(
	pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error,
) {
	t.BasicNestedCallTracer.CaptureFault(pc, op, gas, cost, scope, depth, err)
}

func (t *DataFlowTracer) CaptureEnd(output []byte, gasUsed uint64, tt time.Duration, err error) {
	t.BasicNestedCallTracer.CaptureEnd(output, gasUsed, tt, err)
}

func (t *DataFlowTracer) TransactionStart(tx *engine.Tx, context *engine.VMContext, state engine.State) {
	t.globalStorages = make(map[Analyzer]map[common.Address]*storage)
	t.globalChains = make(map[Analyzer]*chain)
	collections := make(map[Analyzer]*TrackerCollection)
	for analyzer := range t.analyzers {
		t.globalStorages[analyzer] = make(map[common.Address]*storage)
		t.globalChains[analyzer] = newChain()
		// start a fresh new storage tracker for the callee contract of root call.
		contextContractStorage := newStorage()
		collections[analyzer] = &TrackerCollection{
			Stack:   newStack(),
			Memory:  newMemory(),
			Storage: contextContractStorage,
			Call:    newCall(tracers.CallPosition{}),
			Chain:   t.globalChains[analyzer],
		}
	}

	t.BasicNestedCallTracer.TransactionStart(tx, context, state)
	t.CurrentMsgCall().InnerData = &Data{
		trackerCollections: collections,
		lastStateCallbacks: map[Analyzer]AfterOperationHook{},
	}
}

func (t *DataFlowTracer) TransactionEnd(
	tx *engine.Tx, context *engine.VMContext, state engine.State,
	result *core.ExecutionResult, receipt *types.Receipt,
) {
	t.BasicNestedCallTracer.TransactionEnd(tx, context, state, result, receipt)
	t.CurrentMsgCall().InnerData = nil
}
