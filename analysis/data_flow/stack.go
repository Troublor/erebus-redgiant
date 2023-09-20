package data_flow

import (
	"github.com/Troublor/erebus-redgiant/helpers"
	"github.com/ethereum/go-ethereum/params"
)

type stack struct {
	stack []FlowNode
}

func newStack() *stack {
	return &stack{stack: make([]FlowNode, 0, 32)}
}

func (s *stack) Depth() int {
	return len(s.stack)
}

// push mimics the EVM stack push operation.
// Top-most element in after the push is on the last index of elements.
func (s *stack) push(elements ...FlowNode) {
	helpers.SanityCheck(func() bool {
		// If the execution of contracts indeed causes overflow
		// then this sanity check must be violated since it is done before that of EVM.
		return uint64(s.Depth()+len(elements)) <= params.StackLimit
	}, "stack overflow")
	s.stack = append(s.stack, elements...)
}

func (s *stack) pop(n int) FlowNodeList {
	if n == 0 {
		return nil
	}
	helpers.SanityCheck(func() bool {
		return s.Depth() >= n
	}, "stack underflow")
	elements := make([]FlowNode, n)
	copy(elements, s.stack[s.Depth()-n:])
	s.stack = s.stack[:s.Depth()-n]
	return elements
}

func (s *stack) Get(i int) FlowNode {
	helpers.SanityCheck(func() bool {
		return s.Depth() > i
	}, "stack underflow")
	return s.stack[s.Depth()-i-1]
}

func (s *stack) Tail(n int) FlowNodeList {
	helpers.SanityCheck(func() bool {
		return n >= 0 && s.Depth() >= n
	}, "stack underflow")
	return s.stack[s.Depth()-n:]
}

func (s *stack) swap(pos int) {
	helpers.SanityCheck(func() bool {
		return pos > 0 && s.Depth() > pos
	}, "stack underflow")
	top := s.stack[s.Depth()-1]
	swap := s.stack[s.Depth()-pos-1]
	s.stack[s.Depth()-1] = swap
	s.stack[s.Depth()-pos-1] = top
}
