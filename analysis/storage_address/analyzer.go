package storage_address

import (
	"github.com/Troublor/erebus-redgiant/analysis/data_flow"
	"github.com/Troublor/erebus-redgiant/analysis/summary"
	"github.com/ethereum/go-ethereum/core/vm"
)

type StorageAddressingPathAnalyzer struct {
	StorageVariable         *summary.StorageVariable // if not nil, all storage accessing operation will be monitored.
	OnStorageStoredOrLoaded func(op vm.OpCode, addressingPathCandidates []AddressingPath)
}

func (a *StorageAddressingPathAnalyzer) IsMyFlowNode(node data_flow.FlowNode) bool {
	if node == nil {
		return false
	}
	if _, ok := node.(*StorageAddressingNode); ok {
		return true
	}
	return false
}

func (a *StorageAddressingPathAnalyzer) NewFlowNode(operation *data_flow.Operation) data_flow.FlowNode {
	return &StorageAddressingNode{operation: operation}
}

func (a *StorageAddressingPathAnalyzer) CheckOperation(operation *data_flow.Operation) (isSource, isSink bool) {
	if operation.OpCode().IsPush() {
		if a.StorageVariable == nil {
			isSource = true
		} else {
			// we assume that the storage addressing path is not across multiple message calls
			isSource = a.StorageVariable.Address == operation.MsgCall().StateAddr
		}
	}
	switch operation.OpCode() {
	case vm.SLOAD, vm.SSTORE:
		if a.StorageVariable == nil {
			isSink = true
		} else {
			// we assume that the storage addressing path is not across multiple message calls
			isSink = a.StorageVariable.Address == operation.MsgCall().StateAddr &&
				a.StorageVariable.Storage == operation.Arg(0).Hash()
		}
	}
	return isSource, isSink
}

func (a *StorageAddressingPathAnalyzer) SinkTainted(_ *data_flow.TrackerCollection, flowedValue data_flow.FlowNode) {
	a.OnStorageStoredOrLoaded(flowedValue.Operation().OpCode(), flowedValue.(*StorageAddressingNode).AddressingPaths())
}

func (a *StorageAddressingPathAnalyzer) FlowPolicy() data_flow.FlowPolicy {
	return genFlowPolicy()
}
