package summary

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/Troublor/erebus-redgiant/dyengine/tracers"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type StateVarType string

const (
	BalanceVar = "BALANCE"
	StorageVar = "STORAGE"
	CodeVar    = "CODE"
)

type StateVariable interface {
	// Type returns the type of the variable
	Type() StateVarType

	// ID returns the string representation of the variable (value does not matter)
	ID() string

	// Same returns true if the variable is the same as the other variable (value does not matter)
	Same(other StateVariable) bool

	// Equal returns true if the variable is the same as the other variable and values are also the same.
	Equal(other StateVariable) bool

	// Copy returns a copy of the variable
	Copy() StateVariable

	// Location returns the location of the variable
	// Location could be nil if the StateVariable is defined/used by the transaction itself.
	// E.g., the balance of the sender of the transaction.
	Location() tracers.TraceLocation
}

type StateVariables []StateVariable

// IntersectWith returns the list of StateVariable values
// in current StateVariables that are also in other StateVariables.
func (vs StateVariables) IntersectWith(other StateVariables) (StateVariables, StateVariables) {
	var r StateVariables
	var ro StateVariables
	for _, v := range vs {
		for _, variable := range other {
			if v.Same(variable) {
				r = append(r, v)
				ro = append(ro, variable)
			}
		}
	}
	return r, ro
}

func (vs StateVariables) AddWithOverride(elems ...StateVariable) StateVariables {
	slice := vs
elemsLoop:
	for _, elem := range elems {
		elem = elem.Copy()
		for i, e := range slice {
			if e.Same(elem) {
				slice[i] = elem
				continue elemsLoop
			}
		}
		slice = append(slice, elem)
	}
	return slice
}

func (vs StateVariables) AddIfAbsent(elems ...StateVariable) StateVariables {
	slice := vs
elemsLoop:
	for _, elem := range elems {
		elem = elem.Copy()
		for _, e := range slice {
			if e.Same(elem) {
				continue elemsLoop
			}
		}
		slice = append(slice, elem)
	}
	return slice
}

type StorageVariable struct {
	// the Address that this stateVar belongs to
	Address common.Address

	// this is the key of Storage slot
	Storage common.Hash

	// the Value of the stateVar
	Value common.Hash

	L tracers.TraceLocation
}

func (s StorageVariable) Location() tracers.TraceLocation {
	return s.L
}

func (s StorageVariable) Type() StateVarType {
	return StorageVar
}

func (s StorageVariable) ID() string {
	return fmt.Sprintf("%s:%s", s.Address, s.Storage)
}

func (s StorageVariable) Same(other StateVariable) bool {
	if sv, ok := other.(StorageVariable); ok {
		return s.Address == sv.Address && s.Storage == sv.Storage
	} else {
		return false
	}
}

func (s StorageVariable) Equal(other StateVariable) bool {
	if sv, ok := other.(StorageVariable); ok {
		return s.Same(other) && s.Value == sv.Value
	} else {
		return false
	}
}

func (s StorageVariable) Copy() StateVariable {
	return StorageVariable{
		Address: s.Address,
		Storage: s.Storage,
		Value:   s.Value, // common.Hash is an array thus copied in assignment.
		L:       s.L,
	}
}

type BalanceVariable struct {
	// the Address that this stateVar belongs to
	Address common.Address

	// the Value of the stateVar
	Value *big.Int

	L tracers.TraceLocation
}

func (b BalanceVariable) Location() tracers.TraceLocation {
	return b.L
}

func (b BalanceVariable) Type() StateVarType {
	return BalanceVar
}

func (b BalanceVariable) ID() string {
	return fmt.Sprintf("%s:balance", b.Address)
}

func (b BalanceVariable) Same(other StateVariable) bool {
	if bv, ok := other.(BalanceVariable); ok {
		return b.Address == bv.Address
	} else {
		return false
	}
}

func (b BalanceVariable) Equal(other StateVariable) bool {
	if bv, ok := other.(BalanceVariable); ok {
		return b.Same(other) && b.Value.Cmp(bv.Value) == 0
	} else {
		return false
	}
}

func (b BalanceVariable) Copy() StateVariable {
	return BalanceVariable{
		Address: b.Address,
		Value:   new(big.Int).Set(b.Value),
		L:       b.L,
	}
}

type CodeVariable struct {
	// the Address that this stateVar belongs to
	Address common.Address

	Op vm.OpCode

	// the Code of the stateVar
	Code     []byte
	CodeSize int
	CodeHash common.Hash

	L tracers.TraceLocation
}

func (c CodeVariable) Location() tracers.TraceLocation {
	return c.L
}

func (c CodeVariable) Type() StateVarType {
	return CodeVar
}

func (c CodeVariable) ID() string {
	return fmt.Sprintf("%s:code", c.Address)
}

func (c CodeVariable) Same(other StateVariable) bool {
	if cv, ok := other.(CodeVariable); ok {
		return c.Address == cv.Address
	} else {
		return false
	}
}

func (c CodeVariable) Equal(other StateVariable) bool {
	if cv, ok := other.(CodeVariable); ok {
		return c.Same(other) && bytes.Equal(c.Code, cv.Code) && c.CodeSize == cv.CodeSize &&
			c.CodeHash == cv.CodeHash
	} else {
		return false
	}
}

func (c CodeVariable) Copy() StateVariable {
	v := make([]byte, len(c.Code))
	copy(v, c.Code)
	return CodeVariable{
		Address:  c.Address,
		Code:     v,
		CodeSize: c.CodeSize,
		CodeHash: c.CodeHash,
		L:        c.L,
	}
}
