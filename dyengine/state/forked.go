package state

import (
	"bytes"
	"context"
	"math/big"
	"sync"

	"github.com/Troublor/erebus-redgiant/chain"
	"github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

type accessList struct {
	addresses map[common.Address]int
	slots     []map[common.Hash]struct{}
}

// newAccessList creates a new accessList.
func newAccessList() *accessList {
	return &accessList{
		addresses: make(map[common.Address]int),
	}
}

type ForkedState struct {
	*state.StateDB
	mu sync.Mutex

	ctx context.Context

	stateProvider chain.BlockchainReader
	forkNumber    *big.Int

	stateInherited          map[common.Address]bool
	storageInherited        map[common.Address]map[common.Hash]bool
	dirtyClearedStorage     map[common.Address]map[common.Hash]bool
	committedClearedStorage map[common.Address]map[common.Hash]bool

	// The following three fields act as a copy of the corresponding ones in state.StateDB.
	// The purpose is to facilitate Copy.
	accessList *accessList
	thash      common.Hash
	txIndex    int

	snapshots []*ForkedState

	lastErr error
}

func NewForkedState(forkStatePivot chain.BlockchainReader, forkNumber *big.Int) (*ForkedState, error) {
	ctx := context.Background()

	db := rawdb.NewMemoryDatabase()
	sdb := state.NewDatabase(db)
	// TODO originalRoot is only a dummy one since we don't use the state.triePrefetcher at the moment.
	originalRoot := common.Hash{}
	statedb, err := state.New(originalRoot, sdb, nil)
	if err != nil {
		return nil, err
	}

	if forkNumber == nil {
		blockNumber, err := forkStatePivot.BlockNumber(ctx)
		forkNumber = big.NewInt(int64(blockNumber))
		if err != nil {
			return nil, err
		}
	}

	return &ForkedState{
		StateDB: statedb,
		ctx:     context.Background(),

		stateProvider: forkStatePivot,
		forkNumber:    forkNumber,

		stateInherited:          make(map[common.Address]bool),
		storageInherited:        make(map[common.Address]map[common.Hash]bool),
		dirtyClearedStorage:     make(map[common.Address]map[common.Hash]bool),
		committedClearedStorage: make(map[common.Address]map[common.Hash]bool),

		accessList: &accessList{
			addresses: make(map[common.Address]int),
			slots:     make([]map[common.Hash]struct{}, 0),
		},

		snapshots: make([]*ForkedState, 0),

		lastErr: nil,
	}, nil
}

func (s *ForkedState) SubBalance(addr common.Address, amount *big.Int) {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	s.StateDB.SubBalance(addr, amount)
}

func (s *ForkedState) AddBalance(addr common.Address, amount *big.Int) {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	s.StateDB.AddBalance(addr, amount)
}

func (s *ForkedState) GetBalance(addr common.Address) *big.Int {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	return s.StateDB.GetBalance(addr)
}

func (s *ForkedState) GetNonce(addr common.Address) uint64 {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	return s.StateDB.GetNonce(addr)
}

func (s *ForkedState) SetNonce(addr common.Address, nonce uint64) {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	s.StateDB.SetNonce(addr, nonce)
}

func (s *ForkedState) GetCodeHash(addr common.Address) common.Hash {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	return s.StateDB.GetCodeHash(addr)
}

func (s *ForkedState) GetCode(addr common.Address) []byte {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	return s.StateDB.GetCode(addr)
}

func (s *ForkedState) SetCode(addr common.Address, code []byte) {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	s.StateDB.SetCode(addr, code)
}

func (s *ForkedState) GetCodeSize(addr common.Address) int {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	return s.StateDB.GetCodeSize(addr)
}

func (s *ForkedState) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}

	err = s.inheritAccountStorage(addr, hash)
	if err != nil {
		s.lastErr = err
	}
	value := s.StateDB.GetCommittedState(addr, hash)
	// if value is non-zero, it means some previsous transactions have set the value.
	if s.emptyHash(value) {
		// Note that inheritAccountStorage only store inherited storage value
		// in the dirty/pending storage inside stateObject.
		// So, if the getted committed state is empty, it has two possibilities:
		// 1. the storage is not inherited, and it must not be set by previous
		// transactions, otherwise the storage must have been inherited.
		// 2. the storage has been inherited by the storage is set to zero by previous transactions.
		// In these cases, we need to distinguish.
		if stor, ok := s.committedClearedStorage[addr]; ok {
			if stor[hash] {
				return value
			}
		}

		// this is not cleared by any previous transactions, we fetch from the remote (previous block)
		v, err := s.stateProvider.StorageAt(s.ctx, addr, hash, s.forkNumber)
		if err != nil {
			s.lastErr = err
		}
		value = common.BytesToHash(v)
	}

	return value
}

func (s *ForkedState) GetState(addr common.Address, hash common.Hash) common.Hash {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	err = s.inheritAccountStorage(addr, hash)
	if err != nil {
		s.lastErr = err
	}
	return s.StateDB.GetState(addr, hash)
}

func (s *ForkedState) SetState(addr common.Address, hash common.Hash, value common.Hash) {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	err = s.inheritAccountStorage(addr, hash)
	if err != nil {
		s.lastErr = err
	}
	s.StateDB.SetState(addr, hash, value)

	if s.emptyHash(value) {
		s.dirtyClearedStorage[addr][hash] = true
	} else {
		s.dirtyClearedStorage[addr][hash] = false
	}
}

func (s *ForkedState) Suicide(addr common.Address) bool {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	return s.StateDB.Suicide(addr)
}

func (s *ForkedState) HasSuicided(addr common.Address) bool {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	return s.StateDB.HasSuicided(addr)
}

func (s *ForkedState) Exist(addr common.Address) bool {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	return s.StateDB.Exist(addr)
}

func (s *ForkedState) Empty(addr common.Address) bool {
	err := s.inheritAccountState(addr)
	if err != nil {
		s.lastErr = err
	}
	return s.StateDB.Empty(addr)
}

// PrepareAccessList is identical to state.StateDB.PrepareAccessList.
//
// We rewrite it here only to redirect the invocation of AddAddressToAccessList and AddSlotToAccessList
// to our own implementation.
func (s *ForkedState) PrepareAccessList(
	sender common.Address, dst *common.Address, precompiles []common.Address, list types.AccessList,
) {
	// this is essential to let super StateDB clear the leftovers
	s.StateDB.PrepareAccessList(sender, nil, nil, nil)

	// Clear out any leftover from previous executions
	s.accessList = newAccessList()

	s.AddAddressToAccessList(sender)
	if dst != nil {
		s.AddAddressToAccessList(*dst)
		// If it's a create-tx, the destination will be added inside evm.create
	}
	for _, addr := range precompiles {
		s.AddAddressToAccessList(addr)
	}
	for _, el := range list {
		s.AddAddressToAccessList(el.Address)
		for _, key := range el.StorageKeys {
			s.AddSlotToAccessList(el.Address, key)
		}
	}
}

// AddAddressToAccessList calls the state.StateDB's AddAddressToAccessList,
// but also saves the added address so that Copy can also copy the access list.
func (s *ForkedState) AddAddressToAccessList(addr common.Address) {
	if _, present := s.accessList.addresses[addr]; !present {
		s.accessList.addresses[addr] = -1
	}
	s.StateDB.AddAddressToAccessList(addr)
}

// AddSlotToAccessList calls the state.StateDB's AddSlotToAccessList,
// but also saves the added slots so that Copy can also copy the access list.
func (s *ForkedState) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	idx, addrPresent := s.accessList.addresses[addr]
	if !addrPresent || idx == -1 {
		// Account not present, or addr present but no slots there
		s.accessList.addresses[addr] = len(s.accessList.slots)
		slotmap := map[common.Hash]struct{}{slot: {}}
		s.accessList.slots = append(s.accessList.slots, slotmap)
	} else {
		// There is already an (address,slot) mapping
		slotmap := s.accessList.slots[idx]
		if _, ok := slotmap[slot]; !ok {
			slotmap[slot] = struct{}{}
		}
	}
	s.StateDB.AddSlotToAccessList(addr, slot)
}

// Prepare calls the same method of state.StateDB,
// but also saves the tx hash and index so that Copy can also copy them.
func (s *ForkedState) Prepare(thash common.Hash, ti int) {
	s.thash = thash
	s.txIndex = ti
	s.StateDB.Prepare(thash, ti)
}

// Copy copies the content of ForkedState.
//
// The field journal in the state.StateDB is not well copied,
// because currently we don't use journal to implement RevertToSnapshot.
// If the mechanism of RevertToSnapshot is subject to change in the future,
// the journal should also be re-considered here.
func (s *ForkedState) Copy() dyengine.State {
	s.mu.Lock()
	defer s.mu.Unlock()

	ss := s.StateDB.Copy()
	// In addition to the copy of internal StateDB, we also need to copy fields of ForkedState
	stateInherited := make(map[common.Address]bool)
	for k, v := range s.stateInherited {
		stateInherited[k] = v
	}
	storageInherited := make(map[common.Address]map[common.Hash]bool)
	for k, v := range s.storageInherited {
		storageInherited[k] = make(map[common.Hash]bool)
		for kk, vv := range v {
			storageInherited[k][kk] = vv
		}
	}
	dirtyClearedStorage := make(map[common.Address]map[common.Hash]bool)
	for k, v := range s.dirtyClearedStorage {
		dirtyClearedStorage[k] = make(map[common.Hash]bool)
		for kk, vv := range v {
			dirtyClearedStorage[k][kk] = vv
		}
	}
	committedClearedStorage := make(map[common.Address]map[common.Hash]bool)
	for k, v := range s.committedClearedStorage {
		committedClearedStorage[k] = make(map[common.Hash]bool)
		for kk, vv := range v {
			committedClearedStorage[k][kk] = vv
		}
	}

	cpSnapshots := make([]*ForkedState, len(s.snapshots))
	copy(cpSnapshots, s.snapshots)
	cp := &ForkedState{
		StateDB:                 ss,
		ctx:                     s.ctx,
		stateProvider:           s.stateProvider,
		forkNumber:              s.forkNumber,
		stateInherited:          stateInherited,
		storageInherited:        storageInherited,
		dirtyClearedStorage:     dirtyClearedStorage,
		committedClearedStorage: committedClearedStorage,

		accessList: &accessList{
			addresses: make(map[common.Address]int),
			slots:     make([]map[common.Hash]struct{}, 0),
		},

		snapshots: cpSnapshots, // don't deep copy snapshots to save memory
		// snapshots:        make([]*ForkedState, 0), // don't copy snapshots to save memory
		lastErr: s.lastErr,
	}

	// We need to Prepare the state again,
	// because the thash and txIndex field of state.StateDB is not copied yet.
	cp.Prepare(s.thash, s.txIndex)
	// The access list is copied by the Copy method of state.StateDB,
	// but when we invoke Prepare again in previous statement, the access list is cleared.
	// So, we need to manually copy the access list.
	for addr, index := range s.accessList.addresses {
		cp.AddAddressToAccessList(addr)
		if index < 0 {
			continue
		}
		for slot := range s.accessList.slots[index] {
			cp.AddSlotToAccessList(addr, slot)
		}
	}

	return cp
}

// Snapshot implements the functionality in a different way from StateDB.Snapshot.
// This is because the procedure of StateDB.RevertToSnapshot is private and not exposed to public.
// We implement Snapshot here by copying the whole object.
// Snapshots are only shadow copied.
// This design choice is meant to save memory.
// TODO a better design is needed.
func (s *ForkedState) Snapshot() int {
	snapshot := s.Copy().(*ForkedState)
	id := len(s.snapshots)
	s.snapshots = append(s.snapshots, snapshot)
	return id
}

// RevertToSnapshot is used to revert state.
func (s *ForkedState) RevertToSnapshot(snapshotId int) {
	snapshot := s.snapshots[snapshotId]
	s.StateDB = snapshot.StateDB
	s.ctx = snapshot.ctx
	s.stateProvider = snapshot.stateProvider
	s.forkNumber = snapshot.forkNumber
	s.stateInherited = snapshot.stateInherited
	s.storageInherited = snapshot.storageInherited
	s.dirtyClearedStorage = snapshot.dirtyClearedStorage
	s.committedClearedStorage = snapshot.committedClearedStorage
	s.snapshots = snapshot.snapshots
	s.lastErr = snapshot.lastErr
}

func (s *ForkedState) Finalise(deleteEmptyObjects bool) {
	s.StateDB.Finalise(deleteEmptyObjects)
	for acc, stor := range s.dirtyClearedStorage {
		if _, ok := s.committedClearedStorage[acc]; !ok {
			s.committedClearedStorage[acc] = make(map[common.Hash]bool)
		}
		for key, cleared := range stor {
			if cleared {
				s.committedClearedStorage[acc][key] = true
			} else {
				delete(s.committedClearedStorage[acc], key)
			}
		}
	}
}

// inheritAccountState tries to inherit account state from the remote forked blockchain.
//
// The inherited properties are as follows:
// - balance
// - code
// - nonce
//
// Account storage are inherited lazily with inheritAccountStorage function.
func (s *ForkedState) inheritAccountState(addr common.Address) error {
	if !s.stateInherited[addr] && !s.StateDB.Exist(addr) {
		// if the account does not exist, we retrieve balance from remote

		// inherit account balance
		balance, err := s.stateProvider.BalanceAt(s.ctx, addr, s.forkNumber)
		if err != nil {
			return err
		}
		if balance.Cmp(big.NewInt(0)) > 0 {
			s.StateDB.AddBalance(addr, balance)
		}

		// inherit account code
		code, err := s.stateProvider.CodeAt(s.ctx, addr, s.forkNumber)
		if err != nil {
			return err
		}
		if len(code) > 0 {
			s.StateDB.SetCode(addr, code)
		}

		// inherit account nonce
		nonce, err := s.stateProvider.NonceAt(s.ctx, addr, s.forkNumber)
		if err != nil {
			return err
		}
		if nonce > 0 {
			s.StateDB.SetNonce(addr, nonce)
		}

		// inherit account storage
		// Since the total storage address used by an account is unknown,
		// we inherit account storage lazily with inheritAccountStorage.
		s.storageInherited[addr] = make(map[common.Hash]bool)
		s.dirtyClearedStorage[addr] = make(map[common.Hash]bool)

		s.stateInherited[addr] = true
	}
	return nil
}

// inheritAccountStorage lazily inherit storage value from remote blockchain,
// assuming inheritAccountState has been called.
// inheritAccountStorage must be called whenever an account storage is involved,
// i.e., GetState, SetState, GetCommittedState, etc
// If the value of the storage key is set remotely, we will also set it in the memory state.
func (s *ForkedState) inheritAccountStorage(addr common.Address, key common.Hash) error {
	if !s.storageInherited[addr][key] {
		// the storage is not inherited from remote (from previous block),
		// that is to say, the storage is not used previously.
		// We don't need to worry that the storage has been overrided by previous execution,
		// since otherwise inheritAccountStorage would be called in the first place.
		// we need to inherit it
		v, err := s.stateProvider.StorageAt(s.ctx, addr, key, s.forkNumber)
		if err != nil {
			return err
		}
		value := common.BytesToHash(v)
		// no need to set it in memory if the value is zero
		if !s.emptyHash(value) {
			// one limitation here is that the inherited storage should be set
			// in the original storage inside the StateDB,
			// but that API is not exposed to the by go-ethereum.
			// So the original storage is mixed with dirty storage of the first transaction that uses it.
			// This is fine if GetState is called since GetState will don't distinguish origin or dirty storage.
			// But we are in trouble when GetCommittedState is called.
			s.StateDB.SetState(addr, key, value)
		}

		if storage, ok := s.storageInherited[addr]; ok {
			storage[key] = true
		} else {
			s.storageInherited[addr] = map[common.Hash]bool{key: true}
		}
	}
	return nil
}

func (s *ForkedState) emptyHash(hash common.Hash) bool {
	emptyHash := common.Hash{}
	if bytes.Equal(hash.Bytes(), emptyHash.Bytes()) {
		return true
	} else {
		return false
	}
}

func (s *ForkedState) LastError() error {
	return s.lastErr
}

func (s *ForkedState) GetHashFn() func(uint64) common.Hash {
	return func(num uint64) common.Hash {
		hash, err := s.stateProvider.BlockHashByNumber(s.ctx, big.NewInt(int64(num)))
		if err != nil {
			s.lastErr = err
		}
		return hash
	}
}
