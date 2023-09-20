package summary

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/samber/lo"

	"github.com/Troublor/erebus-redgiant/dyengine/tracers"

	engine "github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/ethereum/go-ethereum/common"
)

type Data struct {
	Defs          StateVariables
	Uses          StateVariables
	Transfers     []ITransfer
	Profits       Profits
	ExecutionPath tracers.Trace
}

func (d *Data) clearChanges() {
	d.Defs = nil
	d.Transfers = nil
	d.Profits = nil
}

func (d *Data) addDefs(defs ...StateVariable) {
	d.Defs = d.Defs.AddWithOverride(defs...)
}

func (d *Data) addUses(uses ...StateVariable) {
	d.Uses = d.Uses.AddIfAbsent(uses...)
}

func (d *Data) addProfits(profits ...Profit) {
	d.Profits = d.Profits.Add(profits...)
}

func (d *Data) addTransfers(transfers ...ITransfer) {
	d.Transfers = append(d.Transfers, transfers...)
}

type CallPosition []int

func (p CallPosition) String() string {
	if len(p) == 0 {
		return "root"
	}
	var sections []string
	for _, i := range p {
		sections = append(sections, fmt.Sprintf("%d", i))
	}
	return strings.Join(sections, "_")
}

func (p CallPosition) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

type TxSummary = CallSummary

// CallSummary summarize the structured call execution information.
// It is a tree structure, where Data is only available for the leaf nodes.
// msgCall and Receipt is non-nil only for non-leaf nodes.
type CallSummary struct {
	*tracers.DataWithTrace
	// The length of Data field should be len(NestedSummaries) + 1
	Data []*Data

	// back reference to the msgCall this summary belongs to, ugly but convenient
	msgCall *tracers.MsgCall[*CallSummary]

	//// Utility fields
	//parent       *CallSummary
	//pc           uint64 // the current program counter during execution
	//gasAvailable uint64
	//gasCost      uint64
	// preState is the state snapshot based on which current msgCall executes.
	preState engine.State

	// lastLocation is only used in TxSummaryTracer at runtime.
	// It is used to determine whether current opcode is a basic block head or not.
	//lastLocation ProgramLocation
	// currentBasicBlock is
	//currentBasicBlock *basicBlock

	// cache
	cacheMu                sync.Mutex
	overallTransfers       []ITransfer
	overallProfits         Profits
	overallDefs            StateVariables
	overallUses            StateVariables
	flattenedExecutionPath tracers.Trace
	allInvokedAddresses    []common.Address
}

func newCallSummary(prestate engine.State, call *tracers.MsgCall[*CallSummary]) *CallSummary {
	return &CallSummary{
		DataWithTrace: tracers.NewDataWithTrace(),
		msgCall:       call,
		preState:      prestate,
		Data:          []*Data{{}},
	}
}

func (s *CallSummary) addNestedSummary() {
	s.Data = append(s.Data, &Data{})
}

func (s *CallSummary) currentData() *Data {
	return s.Data[len(s.Data)-1]
}

func (s *CallSummary) IsRoot() bool {
	return s.msgCall.IsRoot()
}

func (s *CallSummary) IsLeaf() bool {
	return len(s.msgCall.NestedCalls) == 0
}

func (s *CallSummary) CallFailed() bool {
	return s.msgCall.Result.Failed()
}

func (s *CallSummary) MsgCall() *tracers.MsgCall[*CallSummary] {
	return s.msgCall
}

func (s *CallSummary) InvalidateCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.overallTransfers = nil
	s.overallProfits = nil
	s.overallDefs = nil
	s.overallUses = nil
}

func (s *CallSummary) NestedSummaries() []*CallSummary {
	return lo.Map(
		s.msgCall.NestedCalls,
		func(call *tracers.MsgCall[*CallSummary], _ int) *CallSummary {
			return call.InnerData
		},
	)
}

func (s *CallSummary) ForEach(summaryCB func(*CallSummary), dataCB func(*CallSummary, *Data)) {
	if summaryCB != nil {
		summaryCB(s)
	}
	if dataCB != nil {
		dataCB(s, s.Data[0])
	}
	for i, summary := range s.NestedSummaries() {
		summary.ForEach(summaryCB, dataCB)
		if dataCB != nil {
			dataCB(s, s.Data[i+1])
		}
	}
}

func (s *CallSummary) NestedSummaryByCallPosition(position CallPosition) *CallSummary {
	if !s.IsRoot() {
		panic("NestedSummaryByCallPosition should be called on root node")
	}

	if len(position) == 0 {
		return s
	} else {
		next := position[0]
		if next >= len(s.NestedSummaries()) {
			return nil
		} else {
			return s.NestedSummaries()[next].NestedSummaryByCallPosition(position[1:])
		}
	}
}

// OverallDefs summarizes the overall defined StateVariable variables by all nested summaries.
// Unchanged variables are not included.
func (s *CallSummary) OverallDefs() StateVariables {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

returnHere:
	if s.overallDefs != nil {
		return s.overallDefs
	}

	s.overallDefs = make(StateVariables, 0)
	if s.CallFailed() {
		// failed call doesn't expose any defined variables
		goto returnHere
	}
	for i := 0; i < len(s.Data)+len(s.NestedSummaries()); i++ {
		if i%2 == 0 {
			// in current call
			index := i / 2
			data := s.Data[index]
			s.overallDefs = s.overallDefs.AddWithOverride(data.Defs...)
		} else {
			// in nested call
			index := (i - 1) / 2
			s.overallDefs = s.overallDefs.AddWithOverride(s.NestedSummaries()[index].OverallDefs()...)
		}
	}

	if !s.IsRoot() {
		goto returnHere
	}

	// only clear unchanged stateVar on the root (top-most overall Defs)
	originalStateGetter := func(account common.Address, key common.Hash) common.Hash {
		// Since CallSummary.preState is the snapshot of the state before the call executes,
		// we can use GetState to get the original state value.
		return s.preState.GetState(account, key)
	}
	originalBalanceGetter := func(account common.Address) *big.Int {
		return new(big.Int).SetBytes(s.preState.GetBalance(account).Bytes())
	}
	cp := make(StateVariables, 0, len(s.overallDefs))
	for _, def := range s.overallDefs {
		switch def.Type() {
		case StorageVar:
			storageDef := def.(StorageVariable)
			originalValue := originalStateGetter(storageDef.Address, storageDef.Storage)
			if originalValue != storageDef.Value {
				cp = append(cp, storageDef)
			}
		case BalanceVar:
			balanceDef := def.(BalanceVariable)
			originalValue := originalBalanceGetter(balanceDef.Address)
			if originalValue.Cmp(balanceDef.Value) != 0 {
				cp = append(cp, def)
			}
		case CodeVar:
			cp = append(cp, def)
		}
	}
	s.overallDefs = cp

	goto returnHere
}

// OverallUses returns the overall def-clear read StateVariable variables by all nested summaries.
// The Uses stored in Data should already be def-clear uses.
func (s *CallSummary) OverallUses() StateVariables {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

returnHere:
	if s.overallUses != nil {
		return s.overallUses
	}

	s.overallUses = make(StateVariables, 0)

	for i := 0; i < len(s.Data)+len(s.NestedSummaries()); i++ {
		if i%2 == 0 {
			// in current call
			index := i / 2
			data := s.Data[index]
			s.overallUses = s.overallUses.AddIfAbsent(data.Uses...)
		} else {
			// in nested call
			index := (i - 1) / 2
			s.overallUses = s.overallUses.AddIfAbsent(s.NestedSummaries()[index].OverallUses()...)
		}
	}

	goto returnHere
}

func (s *CallSummary) OverallProfits() Profits {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

returnHere:
	if s.overallProfits != nil {
		return s.overallProfits
	}

	s.overallProfits = make(Profits, 0)
	if s.CallFailed() {
		// failed call doesn't expose any Profits
		goto returnHere
	}

	for i := 0; i < len(s.Data)+len(s.NestedSummaries()); i++ {
		if i%2 == 0 {
			// in current call
			index := i / 2
			data := s.Data[index]
			s.overallProfits = s.overallProfits.Add(data.Profits...)
		} else {
			// in nested call
			index := (i - 1) / 2
			s.overallProfits = s.overallProfits.Add(s.NestedSummaries()[index].OverallProfits()...)
		}
	}
	s.overallProfits = s.overallProfits.Compact()

	goto returnHere
}

func (s *CallSummary) OverallTransfers() []ITransfer {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

returnHere:
	if s.overallTransfers != nil {
		return s.overallTransfers
	}

	s.overallTransfers = make([]ITransfer, 0)
	if s.CallFailed() {
		// failed call doesn't expose any Transfers
		goto returnHere
	}

	for i := 0; i < len(s.Data)+len(s.NestedSummaries()); i++ {
		if i%2 == 0 {
			// in current call
			index := i / 2
			data := s.Data[index]
			s.overallTransfers = append(s.overallTransfers, data.Transfers...)
		} else {
			// in nested call
			index := (i - 1) / 2
			s.overallTransfers = append(s.overallTransfers, s.NestedSummaries()[index].OverallTransfers()...)
		}
	}

	goto returnHere
}

func (s *CallSummary) FlattenedExecutionPath() tracers.Trace {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

returnHere:
	if s.flattenedExecutionPath != nil {
		return s.flattenedExecutionPath
	}

	s.flattenedExecutionPath = make(tracers.Trace, 0)

	for i := 0; i < len(s.Data)+len(s.NestedSummaries()); i++ {
		if i%2 == 0 {
			// in current call
			index := i / 2
			data := s.Data[index]
			s.flattenedExecutionPath = append(s.flattenedExecutionPath, data.ExecutionPath...)
		} else {
			// in nested call
			index := (i - 1) / 2
			s.flattenedExecutionPath = append(
				s.flattenedExecutionPath,
				s.NestedSummaries()[index].FlattenedExecutionPath()...,
			)
		}
	}

	goto returnHere
}

func (s *CallSummary) GetMsgCallByGasAvailable(gasLeft uint64) *tracers.MsgCall[*CallSummary] {
	lastData := s.Data[len(s.Data)-1]
	lastBlock := lastData.ExecutionPath[len(lastData.ExecutionPath)-1]
	if gasLeft < lastBlock.Tail().GasAvailable() {
		return nil
	}

	for i := 0; i < len(s.Data)+len(s.NestedSummaries()); i++ {
		if i%2 == 0 {
			// in current call
			index := i / 2
			data := s.Data[index]
			startGas := data.ExecutionPath[len(data.ExecutionPath)-1].Head().GasAvailable()
			endGas := data.ExecutionPath[len(data.ExecutionPath)-1].Tail().GasAvailable()
			if startGas >= gasLeft && gasLeft >= endGas {
				return s.msgCall
			}
		} else {
			// in nested call
			index := (i - 1) / 2
			found := s.NestedSummaries()[index].GetMsgCallByGasAvailable(gasLeft)
			if found != nil {
				return found
			}
		}
	}
	return nil
}

func (s *CallSummary) GetNestedSummaryByPosition(position CallPosition) *CallSummary {
	if len(position) == 0 {
		return s
	} else {
		nextIdx := position[0]
		next := s.NestedSummaries()[nextIdx]
		return next.GetNestedSummaryByPosition(position[1:])
	}
}

func (s *CallSummary) AllInvokedAddresses() []common.Address {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

returnHere:
	if s.allInvokedAddresses != nil {
		return s.allInvokedAddresses
	}

	allInvokedAddresses := make(map[common.Address]bool)
	allInvokedAddresses[s.msgCall.CodeAddr] = true

	for _, nestedSummary := range s.NestedSummaries() {
		for _, address := range nestedSummary.AllInvokedAddresses() {
			allInvokedAddresses[address] = true
		}
	}

	s.allInvokedAddresses = make([]common.Address, 0, len(allInvokedAddresses))
	for address, b := range allInvokedAddresses {
		if b {
			s.allInvokedAddresses = append(s.allInvokedAddresses, address)
		}
	}

	goto returnHere
}
