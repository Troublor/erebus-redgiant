package tracers

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/samber/lo"
)

type MsgCallCaller struct {
	CodeAddr  *common.Address // nil if the caller is the EOA, i.e., caller of the root message call
	StateAddr common.Address
	CallSite  TraceLocation // nil if the caller is the EOA, i.e., caller of the root message call
}

type IMsgCallData interface {
}

type MsgCall[D IMsgCallData] struct {
	Position CallPosition
	Result   *core.ExecutionResult
	Receipt  *types.Receipt

	OpCode      vm.OpCode
	Caller      MsgCallCaller
	Precompiled bool
	CodeAddr    common.Address
	StateAddr   common.Address
	Input       []byte
	Value       *big.Int

	// back reference to the parent MsgCall, ugly but convenient
	parent *MsgCall[D]

	NestedCalls []*MsgCall[D]

	// The current location, which is only valid during tracing.
	CurrentLocation TraceLocation

	// InnerData is a placeholder for any struct that composites with BasicNestedCallTracer.
	// It is useful to store any customized data.
	InnerData D
}

func (c *MsgCall[_]) IsRoot() bool {
	return c.Caller.CodeAddr == nil
}

func (c *MsgCall[_]) OutOfGas() bool {
	if errors.Is(c.Result.Unwrap(), vm.ErrOutOfGas) {
		return true
	}
	for _, nestedCall := range c.NestedCalls {
		if nestedCall.OutOfGas() {
			return true
		}
	}
	return false
}

func (c *MsgCall[D]) Parent() *MsgCall[D] {
	return c.parent
}

func (c *MsgCall[D]) GenChildPosition(isPrecompiled bool) CallPosition {
	parentCall := c
	childPosition := CallPosition{
		position:        make(position, len(parentCall.Position.position)+1),
		compactPosition: make(position, len(parentCall.Position.compactPosition)+1),
	}
	copy(childPosition.position, parentCall.Position.position)
	copy(childPosition.compactPosition, parentCall.Position.compactPosition)
	childPosition.position[len(parentCall.Position.position)] = len(parentCall.NestedCalls)
	if isPrecompiled {
		// the compact position of a call to precompiled contract is the same as its parent msg call
		childPosition.compactPosition = childPosition.compactPosition[:len(childPosition.compactPosition)-1]
	} else {
		childPosition.compactPosition[len(parentCall.Position.compactPosition)] =
			len(lo.Filter(parentCall.NestedCalls, func(c *MsgCall[D], _ int) bool {
				return !c.Precompiled
			}))
	}
	return childPosition
}

type Iterator[D IMsgCallData] func(call *MsgCall[D], segmentIndex int) bool

func (c *MsgCall[D]) Iterate(cb Iterator[D]) {
	stop := cb(c, 0)
	if stop {
		return
	}
	for i, nestedCall := range c.NestedCalls {
		nestedCall.Iterate(cb)
		stop = cb(nestedCall, i+1)
		if stop {
			return
		}
	}
}

func (c *MsgCall[D]) FindMsgCall(pos CallPosition) *MsgCall[D] {
	var call *MsgCall[D]
	c.Iterate(func(cc *MsgCall[D], _ int) bool {
		if cc.Position.Equal(pos) {
			call = cc
			return true
		}
		return false
	})
	return call
}
