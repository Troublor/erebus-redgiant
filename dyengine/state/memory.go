package state

import (
	"github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
)

type MemoryState struct {
	// NOTE that MemoryState is only used for testing and debugging.
	// It does not have the concept of "blockchain", which means
	// historic block hash and are always zero hash.

	*state.StateDB
}

func (m *MemoryState) LastError() error {
	return nil
}

func (m *MemoryState) GetHashFn() func(uint64) common.Hash {
	return func(u uint64) common.Hash {
		return common.Hash{}
	}
}

func (m *MemoryState) Copy() dyengine.State {
	return &MemoryState{
		StateDB: m.StateDB.Copy(),
	}
}

func NewMemoryState() *MemoryState {
	db := rawdb.NewMemoryDatabase()
	sdb := state.NewDatabase(db)
	originalRoot := common.Hash{}
	statedb, err := state.New(originalRoot, sdb, nil)
	if err != nil {
		panic(err)
	}
	return &MemoryState{
		statedb,
	}
}
