package dyengine

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

// State defines the interface of a state which EVM modifies when executing a transaction.
type State interface {
	vm.StateDB
	GetHashFn() func(uint64) common.Hash
	Prepare(common.Hash, int)
	Finalise(bool)
	Commit(bool) (common.Hash, error)
	IntermediateRoot(bool) common.Hash
	GetLogs(common.Hash, common.Hash) []*types.Log
	Copy() State
	LastError() error
}
