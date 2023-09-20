package data_flow

import (
	"github.com/ethereum/go-ethereum/common"
)

type chain struct {
	balances map[common.Address]FlowNode
	codes    map[common.Address]FlowNode
}

func newChain() *chain {
	return &chain{
		balances: make(map[common.Address]FlowNode),
		codes:    make(map[common.Address]FlowNode),
	}
}

func (c *chain) GetBalance(addr common.Address) FlowNode {
	return c.balances[addr]
}

func (c *chain) SetBalance(addr common.Address, node FlowNode) {
	c.balances[addr] = node
}

func (c *chain) GetCode(addr common.Address) FlowNode {
	return c.codes[addr]
}

func (c *chain) SetCode(addr common.Address, node FlowNode) {
	c.codes[addr] = node
}
