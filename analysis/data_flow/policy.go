package data_flow

import (
	"github.com/Troublor/erebus-redgiant/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// FlowFn must set the stackResults for tracked new values in stack
// FlowFn must set flowFrom for tracked values
// FlowFn must not manipulate stack.
type FlowFn func(
	analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
	operation *Operation, stackArgs,
	stackResults FlowNodeList,
) AfterOperationHook

type FlowPolicy map[vm.OpCode]FlowFn

func GenDefaultFlowPolicy() FlowPolicy {
	var defaultFlowPolicy = FlowPolicy{
		// 0x0 range
		vm.STOP:       noopFlow,
		vm.ADD:        pureOpFlow,
		vm.MUL:        pureOpFlow,
		vm.SUB:        pureOpFlow,
		vm.DIV:        pureOpFlow,
		vm.SDIV:       pureOpFlow,
		vm.MOD:        pureOpFlow,
		vm.SMOD:       pureOpFlow,
		vm.ADDMOD:     pureOpFlow,
		vm.MULMOD:     pureOpFlow,
		vm.EXP:        pureOpFlow,
		vm.SIGNEXTEND: pureOpFlow,

		// 0x10 range
		vm.LT:     pureOpFlow,
		vm.GT:     pureOpFlow,
		vm.SLT:    pureOpFlow,
		vm.SGT:    pureOpFlow,
		vm.EQ:     pureOpFlow,
		vm.ISZERO: pureOpFlow,
		vm.AND:    pureOpFlow,
		vm.OR:     pureOpFlow,
		vm.XOR:    pureOpFlow,
		vm.NOT:    pureOpFlow,
		vm.BYTE:   pureOpFlow,
		vm.SHL:    pureOpFlow,
		vm.SHR:    pureOpFlow,
		vm.SAR:    pureOpFlow,

		// 0x20 range
		vm.KECCAK256: hashOpFlow,

		// 0x30 range
		vm.ADDRESS:        txConstReadToStackOpFlow,
		vm.BALANCE:        balanceFlow,
		vm.ORIGIN:         txConstReadToStackOpFlow,
		vm.CALLER:         txConstReadToStackOpFlow,
		vm.CALLVALUE:      callValueFlow,
		vm.CALLDATALOAD:   callDataLoadFlow,
		vm.CALLDATASIZE:   txConstReadToStackOpFlow,
		vm.CALLDATACOPY:   callDataCopyFlow,
		vm.CODESIZE:       codeSizeFlow,
		vm.CODECOPY:       codeCopyFlow,
		vm.GASPRICE:       txConstReadToStackOpFlow,
		vm.EXTCODESIZE:    extCodeSizeFlow,
		vm.EXTCODECOPY:    extCodeCopyFlow,
		vm.RETURNDATASIZE: txConstReadToStackOpFlow,
		vm.RETURNDATACOPY: returnDataCopyFlow,
		vm.EXTCODEHASH:    extCodeHashFlow,

		// 0x40 range
		vm.BLOCKHASH:   txConstReadToStackOpFlow,
		vm.COINBASE:    txConstReadToStackOpFlow,
		vm.TIMESTAMP:   txConstReadToStackOpFlow,
		vm.NUMBER:      txConstReadToStackOpFlow,
		vm.DIFFICULTY:  txConstReadToStackOpFlow,
		vm.GASLIMIT:    txConstReadToStackOpFlow,
		vm.CHAINID:     txConstReadToStackOpFlow,
		vm.SELFBALANCE: selfBalanceFlow,
		vm.BASEFEE:     txConstReadToStackOpFlow,

		// 0x50 range
		vm.POP:      noopFlow,
		vm.MLOAD:    mloadFlow,
		vm.MSTORE:   mstoreFlow,
		vm.MSTORE8:  mstore8Flow,
		vm.SLOAD:    sloadFlow,
		vm.SSTORE:   sstoreFlow,
		vm.JUMP:     jumpFlow,
		vm.JUMPI:    jumpiFlow,
		vm.PC:       txConstReadToStackOpFlow,
		vm.MSIZE:    txConstReadToStackOpFlow,
		vm.GAS:      txConstReadToStackOpFlow,
		vm.JUMPDEST: noopFlow,

		// 0x60 range
		vm.PUSH1:  pushFlow,
		vm.PUSH2:  pushFlow,
		vm.PUSH3:  pushFlow,
		vm.PUSH4:  pushFlow,
		vm.PUSH5:  pushFlow,
		vm.PUSH6:  pushFlow,
		vm.PUSH7:  pushFlow,
		vm.PUSH8:  pushFlow,
		vm.PUSH9:  pushFlow,
		vm.PUSH10: pushFlow,
		vm.PUSH11: pushFlow,
		vm.PUSH12: pushFlow,
		vm.PUSH13: pushFlow,
		vm.PUSH14: pushFlow,
		vm.PUSH15: pushFlow,
		vm.PUSH16: pushFlow,
		vm.PUSH17: pushFlow,
		vm.PUSH18: pushFlow,
		vm.PUSH19: pushFlow,
		vm.PUSH20: pushFlow,
		vm.PUSH21: pushFlow,
		vm.PUSH22: pushFlow,
		vm.PUSH23: pushFlow,
		vm.PUSH24: pushFlow,
		vm.PUSH25: pushFlow,
		vm.PUSH26: pushFlow,
		vm.PUSH27: pushFlow,
		vm.PUSH28: pushFlow,
		vm.PUSH29: pushFlow,
		vm.PUSH30: pushFlow,
		vm.PUSH31: pushFlow,
		vm.PUSH32: pushFlow,

		// 0x80 range
		vm.DUP1:  dupFlow,
		vm.DUP2:  dupFlow,
		vm.DUP3:  dupFlow,
		vm.DUP4:  dupFlow,
		vm.DUP5:  dupFlow,
		vm.DUP6:  dupFlow,
		vm.DUP7:  dupFlow,
		vm.DUP8:  dupFlow,
		vm.DUP9:  dupFlow,
		vm.DUP10: dupFlow,
		vm.DUP11: dupFlow,
		vm.DUP12: dupFlow,
		vm.DUP13: dupFlow,
		vm.DUP14: dupFlow,
		vm.DUP15: dupFlow,
		vm.DUP16: dupFlow,

		// 0x90 range
		vm.SWAP1:  swapFlow,
		vm.SWAP2:  swapFlow,
		vm.SWAP3:  swapFlow,
		vm.SWAP4:  swapFlow,
		vm.SWAP5:  swapFlow,
		vm.SWAP6:  swapFlow,
		vm.SWAP7:  swapFlow,
		vm.SWAP8:  swapFlow,
		vm.SWAP9:  swapFlow,
		vm.SWAP10: swapFlow,
		vm.SWAP11: swapFlow,
		vm.SWAP12: swapFlow,
		vm.SWAP13: swapFlow,
		vm.SWAP14: swapFlow,
		vm.SWAP15: swapFlow,
		vm.SWAP16: swapFlow,

		// 0xa0 range
		vm.LOG0: logFlow,
		vm.LOG1: logFlow,
		vm.LOG2: logFlow,
		vm.LOG3: logFlow,
		vm.LOG4: logFlow,

		// 0xf0 range
		vm.CREATE:       createFlow,
		vm.CALL:         callFlow,
		vm.CALLCODE:     callCodeFlow,
		vm.RETURN:       returnFlow,
		vm.DELEGATECALL: delegateCallFlow,
		vm.CREATE2:      createFlow,
		vm.STATICCALL:   staticCallFlow,
		vm.REVERT:       revertFlow,
		vm.INVALID:      invalidFlow,
		vm.SELFDESTRUCT: selfDestructFlow,
	}
	return defaultFlowPolicy
}

var (
	noopFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		return nil
	}

	pushFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		if source, _ := analyzer.CheckOperation(operation); source {
			stackResults[0] = analyzer.NewFlowNode(operation)
		}
		return nil
	}

	dupFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		pos := int(operation.OpCode() - 0x80)
		v := collection.Stack.Get(pos)
		stackResults[0] = v
		if source, _ := analyzer.CheckOperation(operation); source && v == UntaintedValue {
			stackResults[0] = analyzer.NewFlowNode(operation)
		}
		return nil
	}

	swapFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		pos := int(operation.OpCode() - 0x90 + 1)
		collection.Stack.swap(pos)
		return nil
	}

	logFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		if _, sink := analyzer.CheckOperation(operation); sink {
			// check if any topic or data is being tracked (tainted)
			offset := scope.Stack.Back(0).Uint64()
			length := scope.Stack.Back(1).Uint64()
			data := collection.Memory.Load(offset, length)
			if data.IsTainted() || stackArgs.BackSlice(2).IsTainted() {
				node := analyzer.NewFlowNode(operation)
				node.AddUpstream(stackArgs.BackSlice(2)...)
				node.AddUpstream(data...)
				analyzer.SinkTainted(collection, node)
			}
		}
		return nil
	}

	pureOpFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		source, sink := analyzer.CheckOperation(operation)
		// takes multiple but only produce one stack value
		// if any of the stackArgs are tracked, we need to track the result
		if stackArgs.IsTainted() {
			stackResults[0] = analyzer.NewFlowNode(operation)
			stackResults[0].AddUpstream(stackArgs...)
			if sink {
				analyzer.SinkTainted(collection, stackResults[0])
			}
		} else if source {
			stackResults[0] = analyzer.NewFlowNode(operation)
		}
		return nil
	}

	hashOpFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		// Hash operation value flows from the memory slice
		source, sink := analyzer.CheckOperation(operation)
		payload := collection.Memory.Load(operation.Arg(0).Uint64(), operation.Arg(1).Uint64())
		if payload.IsTainted() {
			stackResults[0] = analyzer.NewFlowNode(operation)
			stackResults[0].AddUpstream(payload...)
			if sink {
				analyzer.SinkTainted(collection, stackResults[0])
			}
		} else if source {
			stackResults[0] = analyzer.NewFlowNode(operation)
		}
		return nil
	}

	balanceFlowFactory = func(addr common.Address) FlowFn {
		return func(
			analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
			operation *Operation, stackArgs,
			stackResults FlowNodeList,
		) AfterOperationHook {
			source, sink := analyzer.CheckOperation(operation)
			bal := collection.Chain.GetBalance(addr)
			if bal != UntaintedValue || stackArgs.IsTainted() {
				stackResults[0] = analyzer.NewFlowNode(operation)
				stackResults[0].AddUpstream(stackArgs...)
				stackResults[0].AddUpstream(bal)
				if sink {
					analyzer.SinkTainted(collection, stackResults[0])
				}
			} else if source {
				flowableResult := analyzer.NewFlowNode(operation)
				stackResults[0] = flowableResult
			}
			return nil
		}
	}

	txConstReadToStackOpFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		if source, _ := analyzer.CheckOperation(operation); source {
			stackResults[0] = analyzer.NewFlowNode(operation)
		}
		return nil
	}

	balanceFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		return balanceFlowFactory(scope.Stack.Back(0).Bytes20())(
			analyzer, scope, collection,
			operation, stackArgs,
			stackResults,
		)
	}

	codeReadToStackOpFlowFactory = func(addr common.Address) FlowFn {
		return func(
			analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
			operation *Operation, stackArgs,
			stackResults FlowNodeList,
		) AfterOperationHook {
			source, sink := analyzer.CheckOperation(operation)
			code := collection.Chain.GetCode(addr)
			if code != UntaintedValue || stackArgs.IsTainted() {
				stackResults[0] = analyzer.NewFlowNode(operation)
				stackResults[0].AddUpstream(stackArgs...)
				stackResults[0].AddUpstream(code)
				if sink {
					analyzer.SinkTainted(collection, stackResults[0])
				}
			} else if source {
				flowableResult := analyzer.NewFlowNode(operation)
				stackResults[0] = flowableResult
			}
			return nil
		}
	}

	codeReadToMemoryOpFlowFactory = func(addr common.Address, destOffset, offset, length uint64) FlowFn {
		return func(
			analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
			operation *Operation, stackArgs,
			stackResults FlowNodeList,
		) AfterOperationHook {
			code := collection.Chain.GetCode(addr)
			if stackArgs.IsTainted() {
				taintedNode := analyzer.NewFlowNode(operation)
				taintedNode.AddUpstream(stackArgs...)
				taintedNode.AddUpstream(code)
				collection.Memory.Store(destOffset, length, taintedNode)
			} else if code != UntaintedValue {
				collection.Memory.Store(destOffset, length, code)
			} else if source, _ := analyzer.CheckOperation(operation); source {
				flowableResult := analyzer.NewFlowNode(operation)
				collection.Memory.Store(destOffset, length, flowableResult)
			} else {
				// clear the memory if the written value is not tainted
				collection.Memory.Clear(destOffset, length)
			}
			return nil
		}
	}

	callValueFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		value := collection.Call.value
		if value != UntaintedValue {
			stackResults[0] = value
		} else if source, _ := analyzer.CheckOperation(operation); source {
			stackResults[0] = analyzer.NewFlowNode(operation)
		}
		return nil
	}

	callDataLoadFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		start := scope.Stack.Back(0).Uint64()
		data := collection.Call.GetData(start, 32)
		if data.IsTainted() || stackArgs.IsTainted() {
			flowableResult := analyzer.NewFlowNode(operation)
			flowableResult.AddUpstream(data...)
			flowableResult.AddUpstream(stackArgs...)
			stackResults[0] = flowableResult
		} else if source, _ := analyzer.CheckOperation(operation); source {
			flowableResult := analyzer.NewFlowNode(operation)
			stackResults[0] = flowableResult
		}
		return nil
	}

	callDataCopyFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		destOffset := scope.Stack.Back(0).Uint64()
		offset := scope.Stack.Back(1).Uint64()
		length := scope.Stack.Back(2).Uint64()
		collection.Memory.StoreCallData(collection.Call, destOffset, offset, length)

		if source, _ := analyzer.CheckOperation(operation); source {
			mem := collection.Memory.Load(destOffset, length)
			if !mem.IsTainted() {
				node := analyzer.NewFlowNode(operation)
				collection.Memory.Store(destOffset, length, node)
			}
		}
		return nil
	}

	codeSizeFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		addr := *scope.Contract.CodeAddr
		return codeReadToStackOpFlowFactory(
			addr,
		)(
			analyzer,
			scope,
			collection,
			operation,
			stackArgs,
			stackResults,
		)
	}

	codeCopyFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		addr := *scope.Contract.CodeAddr
		destOffset := scope.Stack.Back(0).Uint64()
		offset := scope.Stack.Back(1).Uint64()
		length := scope.Stack.Back(2).Uint64()
		return codeReadToMemoryOpFlowFactory(addr, destOffset, offset, length)(
			analyzer, scope, collection,
			operation, stackArgs,
			stackResults,
		)
	}

	extCodeSizeFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		addr := scope.Stack.Back(0).Bytes20()
		return codeReadToStackOpFlowFactory(
			addr,
		)(
			analyzer,
			scope,
			collection,
			operation,
			stackArgs,
			stackResults,
		)
	}

	extCodeCopyFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		addr := scope.Stack.Back(0).Bytes20()
		destOffset := scope.Stack.Back(1).Uint64()
		offset := scope.Stack.Back(2).Uint64()
		length := scope.Stack.Back(3).Uint64()
		return codeReadToMemoryOpFlowFactory(addr, destOffset, offset, length)(
			analyzer, scope, collection,
			operation, stackArgs,
			stackResults,
		)
	}

	returnDataCopyFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		destOffset := scope.Stack.Back(0).Uint64()
		offset := scope.Stack.Back(1).Uint64()
		length := scope.Stack.Back(2).Uint64()
		collection.Memory.StoreCallReturnData(collection.ChildCall, destOffset, offset, length)

		if source, _ := analyzer.CheckOperation(operation); source {
			mem := collection.Memory.Load(destOffset, length)
			if !mem.IsTainted() {
				node := analyzer.NewFlowNode(operation)
				collection.Memory.Store(destOffset, length, node)
			}
		}
		return nil
	}

	extCodeHashFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		addr := scope.Stack.Back(0).Bytes20()
		return codeReadToStackOpFlowFactory(
			addr,
		)(
			analyzer,
			scope,
			collection,
			operation,
			stackArgs,
			stackResults,
		)
	}

	selfBalanceFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		addr := scope.Contract.Address()
		return balanceFlowFactory(
			addr,
		)(
			analyzer,
			scope,
			collection,
			operation,
			stackArgs,
			stackResults,
		)
	}

	mloadFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		source, sink := analyzer.CheckOperation(operation)
		start := scope.Stack.Back(0).Uint64()
		data := collection.Memory.Load(start, 32)
		if data.IsTainted() || stackArgs.IsTainted() {
			flowableResult := analyzer.NewFlowNode(operation)
			flowableResult.AddUpstream(data...)
			flowableResult.AddUpstream(stackArgs...)
			stackResults[0] = flowableResult
			if sink {
				analyzer.SinkTainted(collection, flowableResult)
			}
		} else if source {
			stackResults[0] = analyzer.NewFlowNode(operation)
		}
		return nil
	}

	mstoreFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		start := scope.Stack.Back(0).Uint64()
		collection.Memory.Store(start, 32, stackArgs.Back(1))

		if source, _ := analyzer.CheckOperation(operation); source {
			if stackArgs.Back(1) == UntaintedValue {
				node := analyzer.NewFlowNode(operation)
				collection.Memory.Store(start, 32, node)
			}
		}
		return nil
	}

	mstore8Flow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		start := scope.Stack.Back(0).Uint64()
		collection.Memory.Store(start, 1, stackArgs.Back(1))

		if source, _ := analyzer.CheckOperation(operation); source {
			if stackArgs.Back(1) == UntaintedValue {
				node := analyzer.NewFlowNode(operation)
				collection.Memory.Store(start, 1, node)
			}
		}
		return nil
	}

	sloadFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		source, sink := analyzer.CheckOperation(operation)
		key := scope.Stack.Back(0).Bytes32()
		value := collection.Storage.Load(key)
		if value != UntaintedValue || stackArgs.IsTainted() {
			node := analyzer.NewFlowNode(operation)
			node.AddUpstream(collection.Storage.Load(key))
			node.AddUpstream(stackArgs...)
			stackResults[0] = node
			if sink {
				analyzer.SinkTainted(collection, stackResults[0])
			}
		} else if source {
			stackResults[0] = analyzer.NewFlowNode(operation)
		}
		return nil
	}

	sstoreFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		source, sink := analyzer.CheckOperation(operation)
		key := scope.Stack.Back(0).Bytes32()
		if stackArgs.IsTainted() {
			node := analyzer.NewFlowNode(operation)
			node.AddUpstream(stackArgs...)
			collection.Storage.Store(key, node)
			if sink {
				analyzer.SinkTainted(collection, node)
			}
		} else if source {
			node := analyzer.NewFlowNode(operation)
			collection.Storage.Store(key, node)
		} else {
			// clear the storage if the written value is untainted
			collection.Storage.Clear(key)
		}
		return nil
	}

	jumpFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		_, sink := analyzer.CheckOperation(operation)
		dest := stackArgs.Back(0)
		if dest != UntaintedValue {
			if sink {
				node := analyzer.NewFlowNode(operation)
				node.AddUpstream(dest)
				analyzer.SinkTainted(collection, node)
			}
		}
		return nil
	}

	jumpiFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		_, sink := analyzer.CheckOperation(operation)
		cond := stackArgs.Back(1)
		dest := stackArgs.Back(0)
		if cond != UntaintedValue || dest != UntaintedValue {
			if sink {
				node := analyzer.NewFlowNode(operation)
				node.AddUpstream(cond, dest)
				analyzer.SinkTainted(collection, node)
			}
		}
		return nil
	}

	createFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		source, sink := analyzer.CheckOperation(operation)
		// 1. value flows to the balance of the new contract
		// 2. code (in memory) flows to the code of the new contract
		offset := scope.Stack.Back(1).Uint64()
		length := scope.Stack.Back(2).Uint64()
		data := collection.Memory.Load(offset, length)

		collection.ChildCall = newCall(operation.msgCall.GenChildPosition(false))
		if source {
			collection.ChildCall.setFlowSource(analyzer.NewFlowNode(operation))
		}
		collection.ChildCall.StoreData(collection.Memory, offset, length)

		if data.IsTainted() || stackArgs.BackSlice(0, 3).IsTainted() {
			flowableResult := analyzer.NewFlowNode(operation)
			flowableResult.AddUpstream(data...)
			flowableResult.AddUpstream(stackArgs.BackSlice(0, 3)...)
			if sink &&
				(stackArgs.Back(0) != UntaintedValue || data.IsTainted()) {
				analyzer.SinkTainted(collection, flowableResult)
			}
		}

		return func(scope *vm.ScopeContext) {
			address := scope.Stack.Back(0).Bytes20()
			returnData := collection.ChildCall.GetAllReturnData()
			if returnData.IsTainted() {
				node := analyzer.NewFlowNode(operation)
				node.AddUpstream(returnData...)
				collection.Chain.SetCode(address, node)
			} else {
				collection.Chain.SetCode(address, UntaintedValue)
			}
			collection.Chain.SetBalance(address, stackArgs.Back(0))
		}
	}

	callFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		source, sink := analyzer.CheckOperation(operation)
		// 1. value flows to the balance of account being called as well as the caller
		// 2. stackArgs (in memory) flows to the data of call data in child call
		// 3. call status (in call data after call finishes) flows to the stack
		// 4. return data (in child call) flows to the memory
		addr := scope.Stack.Back(1).Bytes20()
		arg := operation.Arg(2)
		if arg.Sign() > 0 {
			collection.Chain.SetBalance(addr, stackArgs.Back(2))
			collection.Chain.SetBalance(scope.Contract.Address(), stackArgs.Back(2))
		}

		argsOffset := scope.Stack.Back(3).Uint64()
		argsLength := scope.Stack.Back(4).Uint64()
		args := collection.Memory.Load(argsOffset, argsLength)
		returnDataOffset := scope.Stack.Back(5).Uint64()
		returnDataLength := scope.Stack.Back(6).Uint64()
		collection.ChildCall = newCall(operation.msgCall.GenChildPosition(helpers.IsPrecompiledContract(addr)))
		if source {
			collection.ChildCall.setFlowSource(analyzer.NewFlowNode(operation))
		}
		collection.ChildCall.StoreData(collection.Memory, argsOffset, argsLength)

		if sink && (stackArgs.BackSlice(1, 3).IsTainted() || args.IsTainted()) {
			node := analyzer.NewFlowNode(operation)
			node.AddUpstream(stackArgs.BackSlice(1, 3)...)
			node.AddUpstream(args...)
			analyzer.SinkTainted(collection, node)
		}

		return func(scope *vm.ScopeContext) {
			if source {
				// CALL reads the balance of the account if value > 0, so it can be the source.
				node := analyzer.NewFlowNode(operation)
				node.AddUpstream(collection.ChildCall.GetSuccess())
				stackResults[0] = node
			} else {
				stackResults[0] = collection.ChildCall.GetSuccess()
			}
			collection.Memory.StoreCallReturnData(
				collection.ChildCall,
				returnDataOffset,
				0,
				returnDataLength,
			)
		}
	}

	callCodeFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		source, sink := analyzer.CheckOperation(operation)
		// 1. value flows to the balance of current account
		// 2. stackArgs (in memory) flows to the data of call data in child call
		// 3. call status (in call data after call finishes) flows to the stack
		// 4. return data (in child call) flows to the memory
		arg := operation.Arg(2)
		if arg.Sign() > 0 {
			collection.Chain.SetBalance(scope.Contract.Address(), stackArgs.Back(2))
		}

		addr := scope.Stack.Back(1).Bytes20()
		argsOffset := scope.Stack.Back(3).Uint64()
		argsLength := scope.Stack.Back(4).Uint64()
		args := collection.Memory.Load(argsOffset, argsLength)
		returnDataOffset := scope.Stack.Back(5).Uint64()
		returnDataLength := scope.Stack.Back(6).Uint64()
		collection.ChildCall = newCall(operation.msgCall.GenChildPosition(helpers.IsPrecompiledContract(addr)))
		if source {
			collection.ChildCall.setFlowSource(analyzer.NewFlowNode(operation))
		}
		collection.ChildCall.StoreData(collection.Memory, argsOffset, argsLength)

		if sink &&
			(stackArgs.BackSlice(1, 3).IsTainted() || args.IsTainted()) {
			node := analyzer.NewFlowNode(operation)
			node.AddUpstream(stackArgs.BackSlice(1, 3)...)
			node.AddUpstream(args...)
			analyzer.SinkTainted(collection, node)
		}

		return func(scope *vm.ScopeContext) {
			stackResults[0] = collection.ChildCall.GetSuccess()
			collection.Memory.StoreCallReturnData(
				collection.ChildCall,
				returnDataOffset,
				0,
				returnDataLength,
			)
		}
	}

	returnFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		// 1. return data (in memory) flows to the returnData of current call
		// 2. (if source) new flowable value flows to the status of current call
		offset := scope.Stack.Back(0).Uint64()
		length := scope.Stack.Back(1).Uint64()
		collection.Call.StoreReturnData(collection.Memory, offset, length)

		if source, sink := analyzer.CheckOperation(operation); source {
			flowableResult := analyzer.NewFlowNode(operation)
			collection.Call.success = flowableResult
		} else if sink && collection.Call.ReturnDataIsTainted(analyzer) {
			node := analyzer.NewFlowNode(operation)
			node.AddUpstream(collection.Call.GetAllReturnData()...)
			analyzer.SinkTainted(collection, node)
		}
		return nil
	}

	delegateCallFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		source, sink := analyzer.CheckOperation(operation)
		// 1. stackArgs (in memory) flows to the data in child call
		// 2. call status (in call data after call finishes) flows to the stack
		// 3. return data (in child call) flows to the memory
		addr := scope.Stack.Back(1).Bytes20()
		argsOffset := scope.Stack.Back(2).Uint64()
		argsLength := scope.Stack.Back(3).Uint64()
		args := collection.Memory.Load(argsOffset, argsLength)
		returnDataOffset := scope.Stack.Back(4).Uint64()
		returnDataLength := scope.Stack.Back(5).Uint64()
		collection.ChildCall = newCall(operation.msgCall.GenChildPosition(helpers.IsPrecompiledContract(addr)))
		if source {
			collection.ChildCall.setFlowSource(analyzer.NewFlowNode(operation))
		}
		collection.ChildCall.StoreData(collection.Memory, argsOffset, argsLength)

		if sink &&
			(stackArgs.Back(1) != UntaintedValue || args.IsTainted()) {
			node := analyzer.NewFlowNode(operation)
			node.AddUpstream(stackArgs.Back(1))
			node.AddUpstream(args...)
			analyzer.SinkTainted(collection, node)
		}

		return func(scope *vm.ScopeContext) {
			stackResults[0] = collection.ChildCall.GetSuccess()
			collection.Memory.StoreCallReturnData(
				collection.ChildCall,
				returnDataOffset,
				0,
				returnDataLength,
			)
		}
	}

	staticCallFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		source, sink := analyzer.CheckOperation(operation)
		// 1. value flows to the balance of the new contract
		// 2. code (in memory) flows to the code of the new contract
		addr := scope.Stack.Back(1).Bytes20()
		offset := scope.Stack.Back(2).Uint64()
		length := scope.Stack.Back(3).Uint64()
		args := collection.Memory.Load(offset, length)
		returnDataOffset := scope.Stack.Back(4).Uint64()
		returnDataLength := scope.Stack.Back(5).Uint64()

		collection.ChildCall = newCall(operation.msgCall.GenChildPosition(helpers.IsPrecompiledContract(addr)))
		if source {
			collection.ChildCall.setFlowSource(analyzer.NewFlowNode(operation))
		}
		collection.ChildCall.StoreData(collection.Memory, offset, length)

		if sink &&
			(stackArgs.Back(1) != UntaintedValue || args.IsTainted()) {
			node := analyzer.NewFlowNode(operation)
			node.AddUpstream(stackArgs.Back(1))
			node.AddUpstream(args...)
			analyzer.SinkTainted(collection, node)
		}

		return func(scope *vm.ScopeContext) {
			stackResults[0] = collection.ChildCall.GetSuccess()
			collection.Memory.StoreCallReturnData(
				collection.ChildCall,
				returnDataOffset,
				0,
				returnDataLength,
			)
		}
	}

	revertFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		// 1. return data (in memory) flows to the returnData of current call
		// 2. (if source) new flowable value flows to the status of current call
		offset := scope.Stack.Back(0).Uint64()
		length := scope.Stack.Back(1).Uint64()
		collection.Call.StoreReturnData(collection.Memory, offset, length)

		if source, sink := analyzer.CheckOperation(operation); source {
			flowableResult := analyzer.NewFlowNode(operation)
			collection.Call.success = flowableResult
		} else if sink && collection.Call.ReturnDataIsTainted(analyzer) {
			node := analyzer.NewFlowNode(operation)
			node.AddUpstream(collection.Call.GetAllReturnData()...)
			analyzer.SinkTainted(collection, node)
		}
		return nil
	}

	invalidFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		if source, sink := analyzer.CheckOperation(operation); sink {
			node := analyzer.NewFlowNode(operation)
			analyzer.SinkTainted(collection, node)
		} else if source {
			node := analyzer.NewFlowNode(operation)
			collection.Call.success = node
		}
		return nil
	}

	selfDestructFlow FlowFn = func(
		analyzer Analyzer, scope *vm.ScopeContext, collection *TrackerCollection,
		operation *Operation, stackArgs,
		stackResults FlowNodeList,
	) AfterOperationHook {
		// 1. (if source) new value flows to the code of current contract
		// 2. balance of current contract flows to the balance of the given address/current contract
		// 3. (if source) new flowable value flows to the status of current call
		if source, sink := analyzer.CheckOperation(operation); source {
			collection.Chain.SetCode(scope.Contract.Address(), analyzer.NewFlowNode(operation))
			collection.Call.success = analyzer.NewFlowNode(operation)
		} else if sink && stackArgs.Back(0) != UntaintedValue {
			node := analyzer.NewFlowNode(operation)
			node.AddUpstream(stackArgs.Back(0))
			analyzer.SinkTainted(collection, node)
		}
		addr := scope.Stack.Back(0).Bytes20()
		collection.Chain.SetBalance(addr, collection.Chain.GetBalance(scope.Contract.Address()))
		collection.Chain.SetBalance(scope.Contract.Address(), analyzer.NewFlowNode(operation))
		return nil
	}
)
