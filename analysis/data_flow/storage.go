package data_flow

import "github.com/ethereum/go-ethereum/common"

type storage struct {
	storage map[common.Hash]FlowNode
}

func newStorage() *storage {
	return &storage{
		storage: make(map[common.Hash]FlowNode),
	}
}

func (s *storage) Load(key common.Hash) FlowNode {
	return s.storage[key]
}

func (s *storage) Store(key common.Hash, value FlowNode) {
	s.storage[key] = value
}

func (s *storage) Clear(key common.Hash) {
	s.Store(key, UntaintedValue)
}
