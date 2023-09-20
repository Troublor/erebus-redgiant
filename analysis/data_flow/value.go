package data_flow

import (
	"fmt"
	"math/big"

	"github.com/holiman/uint256"

	"github.com/Troublor/erebus-redgiant/dyengine/tracers"

	"github.com/ethereum/go-ethereum/common"
)

var UntaintedValue FlowNode = nil

type Operand struct {
	*uint256.Int
}

func (o Operand) Nil() bool {
	return o.Int == nil
}

func (o Operand) Uint64() uint64 {
	return o.Int.Uint64()
}

func (o Operand) Big() *big.Int {
	return o.ToBig()
}

func (o Operand) Hash() common.Hash {
	return o.Bytes32()
}

func (o Operand) Address() common.Address {
	return o.Bytes20()
}

func (o Operand) Bool() bool {
	return o.Int.Cmp(new(uint256.Int)) != 0
}

type Operation struct {
	tracers.TraceLocation
	msgCall *tracers.MsgCall[*Data]

	args []Operand // top-most element on the last place of the array
}

func (o Operation) CodeAddr() common.Address {
	return o.msgCall.CodeAddr
}

func (o Operation) MsgCall() *tracers.MsgCall[*Data] {
	return o.msgCall
}

func (o Operation) Arg(i int) Operand {
	return o.args[len(o.args)-i-1]
}

type IOperationBlock = tracers.IBasicBlock[*Operation]

type OperationTrace = tracers.Blocks[*Operation, IOperationBlock]

type NodeID string

type FlowNode interface {
	ID() NodeID

	// Operation returns the operation that corresponds to this node.
	Operation() *Operation

	// From returns a list of FlowNode where the data flows from.
	From() FlowNodeList

	// AddUpstream adds a list of FlowNode where the data flows from.
	AddUpstream(...FlowNode)
}

type RawFlowNode struct {
	label string // label to identify this a group of RawFlowNode

	operation *Operation
	from      map[NodeID]FlowNode
}

func NewRawFlowNode(operation *Operation, label string) FlowNode {
	n := &RawFlowNode{
		label: label,

		operation: operation,
		from:      make(map[NodeID]FlowNode),
	}
	return n
}

func (n *RawFlowNode) Label() string {
	return n.label
}

func (n *RawFlowNode) HasUpstream() bool {
	return len(n.from) > 0
}

func (n *RawFlowNode) ID() NodeID {
	return NodeID(fmt.Sprintf("%s-%s", n.label, n.operation.ID()))
}

func (n *RawFlowNode) Operation() *Operation {
	return n.operation
}

func (n *RawFlowNode) From() FlowNodeList {
	var from []FlowNode
	for _, v := range n.from {
		if v != nil {
			from = append(from, v)
		}
	}
	return from
}

func (n *RawFlowNode) AddUpstream(node ...FlowNode) {
	if n == nil {
		return
	}
	for _, flowNode := range node {
		if flowNode != nil {
			n.from[flowNode.ID()] = flowNode
		}
	}
}

type FlowNodeList []FlowNode

func (l FlowNodeList) Back(index int) FlowNode {
	return l[len(l)-index-1]
}

func (l FlowNodeList) BackSlice(from int, to ...int) FlowNodeList {
	if len(to) > 1 {
		panic("too many arguments")
	}
	if len(to) == 1 {
		return l[len(l)-to[0] : len(l)-from]
	} else {
		return l[0 : len(l)-from]
	}
}

func (l FlowNodeList) IsTainted() bool {
	for _, node := range l {
		if node != UntaintedValue {
			return true
		}
	}
	return false
}
