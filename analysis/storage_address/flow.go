package storage_address

import (
	"math/big"

	"github.com/Troublor/erebus-redgiant/analysis/data_flow"
	"github.com/Troublor/erebus-redgiant/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

func genFlowPolicy() data_flow.FlowPolicy {
	policy := data_flow.GenDefaultFlowPolicy()
	policy[vm.ADD] = pureOpFlowFn
	//policy[vm.SUB] = pureOpFlowFn
	//policy[vm.MUL] = pureOpFlowFn
	//policy[vm.DIV] = pureOpFlowFn
	//policy[vm.SDIV] = pureOpFlowFn
	policy[vm.SLOAD] = storageOpFlowFn
	policy[vm.SSTORE] = storageOpFlowFn
	policy[vm.KECCAK256] = hashOpFlowFn
	return policy
}

// storageOpFlowFn is tailored to the Storage Addressing Path analysis.
// It will add stackArgs to the StorageAddressingNode of the storage operation, with arguments order preserved.
var storageOpFlowFn data_flow.FlowFn = func(
	analyzer data_flow.Analyzer,
	scope *vm.ScopeContext, collection *data_flow.TrackerCollection,
	operation *data_flow.Operation, stackArgs, stackResults data_flow.FlowNodeList,
) data_flow.AfterOperationHook {
	_, sink := analyzer.CheckOperation(operation)
	address := stackArgs.Back(0)
	if address != data_flow.UntaintedValue {
		node := analyzer.NewFlowNode(operation).(*StorageAddressingNode)
		node.AddUpstream(address)
		node.Operand = operation.Arg(0)
		if sink {
			analyzer.SinkTainted(collection, node)
		}
	}
	return nil
}

// pureOpFlowFn is tailored to the Storage Addressing Path analysis.
// It will add stackArgs to the StorageAddressingNode of the operation, with arguments order preserved.
var pureOpFlowFn data_flow.FlowFn = func(
	analyzer data_flow.Analyzer,
	scope *vm.ScopeContext, collection *data_flow.TrackerCollection,
	operation *data_flow.Operation, stackArgs, stackResults data_flow.FlowNodeList,
) data_flow.AfterOperationHook {
	if stackArgs.IsTainted() {
		stackResults[0] = analyzer.NewFlowNode(operation)
		stackResults[0].AddUpstream(stackArgs...)
	}
	return nil
}

// hashOpFlowFn is tailored to the Storage Addressing Path analysis.
// It will check if the hash is consistent with the storage addressing operation.
var hashOpFlowFn data_flow.FlowFn = func(
	analyzer data_flow.Analyzer,
	scope *vm.ScopeContext, collection *data_flow.TrackerCollection,
	operation *data_flow.Operation, stackArgs, stackResults data_flow.FlowNodeList,
) data_flow.AfterOperationHook {
	offset := operation.Arg(0).Uint64()
	size := operation.Arg(1).Uint64()
	if size == 2*common.HashLength {
		// hash operation in addressing path is always hashing 2 words
		//ingredient := collection.Memory.Load(offset, common.HashLength)
		operand := collection.Memory.Load(offset+common.HashLength, common.HashLength)
		node := analyzer.NewFlowNode(operation).(*StorageAddressingNode)
		node.AddUpstream(operand...)
		ingredientBytes := helpers.GetMemoryCopyWithPadding(
			scope.Memory,
			int64(offset),
			common.HashLength,
		)
		ingredientValue, _ := uint256.FromBig(new(big.Int).SetBytes(ingredientBytes))
		node.Ingredient = &data_flow.Operand{Int: ingredientValue}
		operandBytes := helpers.GetMemoryCopyWithPadding(
			scope.Memory,
			int64(offset+common.HashLength),
			common.HashLength,
		)
		operandValue, _ := uint256.FromBig(new(big.Int).SetBytes(operandBytes))
		node.Operand = data_flow.Operand{Int: operandValue}
		if operand.IsTainted() {
			stackResults[0] = node
		}
	} else if size == common.HashLength {
		operand := collection.Memory.Load(offset, common.HashLength)
		node := analyzer.NewFlowNode(operation).(*StorageAddressingNode)
		node.AddUpstream(operand...)
		operandBytes := helpers.GetMemoryCopyWithPadding(scope.Memory, int64(offset), common.HashLength)
		operandValue, _ := uint256.FromBig(new(big.Int).SetBytes(operandBytes))
		node.Operand = data_flow.Operand{Int: operandValue}
		if operand.IsTainted() {
			stackResults[0] = node
		}
	}
	return nil
}
