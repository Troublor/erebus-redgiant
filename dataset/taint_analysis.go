package dataset

import (
	"fmt"
	"math"

	"github.com/Troublor/erebus-redgiant/dyengine/tracers"
	"github.com/Troublor/erebus-redgiant/helpers"

	. "github.com/Troublor/erebus-redgiant/analysis/data_flow"
	"github.com/Troublor/erebus-redgiant/analysis/summary"
	engine "github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

type attackTaintNode struct {
	*RawFlowNode
	controlFrom map[NodeID]FlowNode
	controlTo   map[NodeID]FlowNode
}

func newAttackTaintNode(operation *Operation) *attackTaintNode {
	return &attackTaintNode{
		RawFlowNode: NewRawFlowNode(operation, "attackTaintNode").(*RawFlowNode),
		controlFrom: make(map[NodeID]FlowNode),
		controlTo:   make(map[NodeID]FlowNode),
	}
}

// From returns a list of FlowNode where the data flows from.
func (n *attackTaintNode) From() FlowNodeList {
	from := n.RawFlowNode.From()
	for _, node := range n.controlFrom {
		from = append(from, node)
	}
	return from
}

// TaintAnalyzer analyze the data flow of the given shared variable, and
// additionally, it will also extend to the data flow with control dependency propagation.
// The control dependency is referencing the given refPath, i.e., if the current
// execution path diverges from the refPath, we consider control flow taint the variables defined in the branch.
type TaintAnalyzer struct {
	// shared variable, which is loaded in the victim transaction with attack scenario
	// the shared variable should be the variable loaded in the transaction where this analyzer is applied.
	v summary.StateVariable

	// reference execution path
	refPath tracers.Trace

	state engine.State

	// out
	Sources map[uint]bool // map from source location Index() to bool
	Jumps   map[uint]FlowNode
	Logs    map[uint]FlowNode
	Calls   map[uint]FlowNode

	// making sure that same operation in same message call always corresponding to the same FlowNode
	flowNodes map[string]FlowNode

	currentBlockIndex int

	isDiverging         bool
	divergingNode       *attackTaintNode
	potentialMergeBlock map[string]int

	lastTaintedSink FlowNode
}

func NewTaintAnalyzer(
	state engine.State,
	readSharedVariable summary.StateVariable,
	refPath tracers.Trace,
) *TaintAnalyzer {
	return &TaintAnalyzer{
		state:   state,
		v:       readSharedVariable,
		refPath: refPath,

		Sources: make(map[uint]bool),
		Jumps:   make(map[uint]FlowNode),
		Logs:    make(map[uint]FlowNode),
		Calls:   make(map[uint]FlowNode),

		flowNodes: make(map[string]FlowNode),
	}
}

func (a *TaintAnalyzer) NewFlowNode(operation *Operation) FlowNode {
	if n, ok := a.flowNodes[operation.ID()]; ok {
		return n
	} else {
		node := newAttackTaintNode(operation)
		if a.isDiverging && a.divergingNode != nil {
			node.controlFrom[a.divergingNode.ID()] = a.divergingNode
			a.divergingNode.controlTo[node.ID()] = node
		}
		a.flowNodes[operation.ID()] = node
		return node
	}
}

func (a *TaintAnalyzer) CheckOperation(
	operation *Operation,
) (isSource, isSink bool) {
	defer func() {
		if a.isDiverging && a.divergingNode != nil {
			// fake source operation, for control dependency propagation
			isSource = true
		}
	}()
	if a.sharedVariableLoaded(operation) {
		isSource = true
		// if current operation is the true taint source (not fake source for control dependency propagation)
		a.Sources[operation.Index()] = true
	}
	switch operation.OpCode() {
	// consequence
	case vm.JUMP, vm.JUMPI:
		isSink = true
	case vm.CALL:
		isSink = true
	case vm.LOG0, vm.LOG1, vm.LOG2, vm.LOG3, vm.LOG4:
		isSink = true
	}

	// control divergence
	switch operation.OpCode() {
	case vm.JUMP, vm.JUMPI, vm.RETURN, vm.STOP, vm.INVALID, vm.REVERT, vm.SELFDESTRUCT,
		vm.CREATE, vm.CREATE2, vm.CALL, vm.CALLCODE, vm.DELEGATECALL, vm.STATICCALL:
		isSink = true
	}
	return
}

func (a *TaintAnalyzer) SinkTainted(
	collection *TrackerCollection, flowedValue FlowNode,
) {
	// consequence
	switch flowedValue.Operation().OpCode() {
	case vm.JUMP, vm.JUMPI:
		a.Jumps[flowedValue.Operation().Index()] = flowedValue
	case vm.LOG0, vm.LOG1, vm.LOG2, vm.LOG3, vm.LOG4:
		a.Logs[flowedValue.Operation().Index()] = flowedValue
	case vm.CALL:
		a.Calls[flowedValue.Operation().Index()] = flowedValue
	}

	a.lastTaintedSink = flowedValue
}

func (a *TaintAnalyzer) FlowPolicy() FlowPolicy {
	policy := GenDefaultFlowPolicy()
	a.attachBasicBlockTailHook(policy, vm.JUMP)
	a.attachBasicBlockTailHook(policy, vm.JUMPI)
	a.attachBasicBlockTailHook(policy, vm.RETURN)
	a.attachBasicBlockTailHook(policy, vm.STOP)
	a.attachBasicBlockTailHook(policy, vm.INVALID)
	a.attachBasicBlockTailHook(policy, vm.REVERT)
	a.attachBasicBlockTailHook(policy, vm.SELFDESTRUCT)
	a.attachBasicBlockTailHook(policy, vm.CREATE)
	a.attachBasicBlockTailHook(policy, vm.CREATE2)
	a.attachBasicBlockTailHook(policy, vm.CALL)
	a.attachBasicBlockTailHook(policy, vm.CALLCODE)
	a.attachBasicBlockTailHook(policy, vm.DELEGATECALL)
	a.attachBasicBlockTailHook(policy, vm.STATICCALL)
	return policy
}

func (a *TaintAnalyzer) sharedVariableLoaded(operation *Operation) bool {
	switch v := a.v.(type) {
	case summary.StorageVariable:
		return operation.OpCode() == vm.SLOAD &&
			operation.MsgCall().StateAddr == v.Address &&
			operation.Arg(0).Hash() == v.Storage
	case summary.BalanceVariable:
		return operation.OpCode() == vm.BALANCE && operation.Arg(0).Address() == v.Address ||
			operation.OpCode() == vm.SELFBALANCE && operation.MsgCall().StateAddr == v.Address ||
			operation.OpCode() == vm.CALL && operation.Arg(2).Int.Sign() > 0 && operation.MsgCall().StateAddr == v.Address
	case summary.CodeVariable:
		switch operation.OpCode() {
		case vm.CODESIZE, vm.CODECOPY:
			return operation.MsgCall().CodeAddr == v.Address
		case vm.EXTCODESIZE, vm.EXTCODECOPY, vm.EXTCODEHASH:
			arg := operation.Arg(0)
			return common.BigToAddress(arg.ToBig()) == v.Address
		default:
			return false
		}
	default:
		return false
	}
}

func (a *TaintAnalyzer) getNextBlock(
	scope *vm.ScopeContext, operation *Operation,
) (codeAddr common.Address, pc uint64) {
	switch operation.OpCode() {
	case vm.JUMP:
		codeAddr = *scope.Contract.CodeAddr
		pc = scope.Stack.Back(0).Uint64()
	case vm.JUMPI:
		codeAddr = *scope.Contract.CodeAddr
		dest := scope.Stack.Back(0).Uint64()
		cond := scope.Stack.Back(1).ToBig()
		if cond.Sign() == 0 {
			pc = operation.PC() + 1
		} else {
			pc = dest
		}
	case vm.RETURN, vm.STOP, vm.INVALID, vm.REVERT, vm.SELFDESTRUCT:
		parent := operation.MsgCall().Parent()
		if parent != nil {
			codeAddr = parent.CodeAddr
			pc = parent.CurrentLocation.PC() + 1
		} else {
			codeAddr = common.Address{}
			pc = math.MaxUint64
		}
	case vm.CREATE:
		caller := scope.Contract.Address()
		codeAddr = crypto.CreateAddress(caller, a.state.GetNonce(caller))
		pc = 0
	case vm.CREATE2:
		caller := scope.Contract.Address()
		salt := scope.Stack.Back(3).Bytes32()
		codeOffset, codeSize := scope.Stack.Back(1), scope.Stack.Back(2)
		code := helpers.GetMemoryCopyWithPadding(
			scope.Memory,
			int64(codeOffset.Uint64()),
			int64(codeSize.Uint64()),
		)
		codeHash := crypto.Keccak256Hash(code)
		codeAddr = crypto.CreateAddress2(caller, salt, codeHash.Bytes())
		pc = 0
	case vm.CALL, vm.CALLCODE, vm.DELEGATECALL, vm.STATICCALL:
		codeAddr = scope.Stack.Back(1).Bytes20()
		pc = 0
	}
	return codeAddr, pc
}

func (a *TaintAnalyzer) attachBasicBlockTailHook(policy FlowPolicy, op vm.OpCode) {
	id := func(codeAddr common.Address, pc uint64) string {
		return fmt.Sprintf("%s:%d", codeAddr.Hex(), pc)
	}
	flowFn := policy[op]
	policy[op] = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		a.lastTaintedSink = nil
		hook := flowFn(analyzer, scope, collection, operation, stackArgs, stackResults)
		nextCodeAddr, nextPC := a.getNextBlock(scope, operation)
		if a.isDiverging {
			// check if the divergence merge
			if index, exist := a.potentialMergeBlock[id(nextCodeAddr, nextPC)]; exist {
				// merge in next block
				a.currentBlockIndex = index
				a.isDiverging = false
				a.divergingNode = nil
				a.potentialMergeBlock = nil
			}
		} else {
			if a.currentBlockIndex >= len(a.refPath)-1 {
				// we have reached the end of the path
				// we consider this as a divergence in case there are still following blocks in the tracing.
				a.isDiverging = true
				a.potentialMergeBlock = make(map[string]int)
			} else {
				expectedNext := a.refPath[a.currentBlockIndex+1]
				if expectedNext.CodeAddr() == nextCodeAddr && expectedNext.Head().PC() == nextPC {
					// no diverge
					a.currentBlockIndex++
				} else {
					// diverge
					a.isDiverging = true
					a.potentialMergeBlock = make(map[string]int)
					if a.lastTaintedSink != nil {
						a.divergingNode = a.lastTaintedSink.(*attackTaintNode)
					}
					for i := a.currentBlockIndex + 1; i < len(a.refPath); i++ {
						b := a.refPath[i]
						a.potentialMergeBlock[id(b.CodeAddr(), b.Head().PC())] = i
					}
				}
			}
		}
		return hook
	}
}
