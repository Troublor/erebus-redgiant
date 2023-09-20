package dataset

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Troublor/erebus-redgiant/chain"
	"github.com/samber/lo"

	"github.com/Troublor/erebus-redgiant/analysis/data_flow"
	"github.com/Troublor/erebus-redgiant/analysis/storage_address"
	"github.com/Troublor/erebus-redgiant/analysis/summary"
	engine "github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/Troublor/erebus-redgiant/dyengine/tracers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"
)

type AttackPattern string

const (
	PathConditionAlteration AttackPattern = "PathConditionAlteration"
	ComputationAlteration   AttackPattern = "ComputationAlteration"
	GasEstimationGriefing   AttackPattern = "GasEstimationGriefing"
)

type StorageVariableWithAddressingPaths struct {
	summary.StorageVariable

	// all possible addressing paths of this storage variable.
	// Note that this field is not set by the TxSummaryTracer.
	// It can be computed using the storage_address Tracer.
	AddressingPaths []storage_address.AddressingPath
}

type AttackAnalysis struct {
	Pattern        AttackPattern
	SharedVariable summary.StateVariable
	OriginalValue  fmt.Stringer
	AlteredValue   fmt.Stringer
	// WritePoint could be nil if the SharedVariable's write location is nil
	WritePoint tracers.TraceLocation
	// ReadPoint could be nil if the SharedVariable's write location is nil
	ReadPoint        tracers.TraceLocation
	ConsequencePoint tracers.TraceLocation

	// InfluenceTrace is the tracers.Trace which
	// 1. starting from the ReadPoint
	// 2. ending at the ConsequencePoint
	//    2.1  for PathConditionAlteration attacks, ConsequencePoint is the nearest
	//          conditional statement (JUMP, JUMPI) that controls the victim profits
	//    2.2  for ComputationAlteration attacks, ConsequencePoint is the transfer operation
	//       2.2.1 if the transfer operation is via a token contract, the ConsequencePoint is
	//          resolved to the CALL statement to the token contract.
	//       2.2.2 if the transfer operation is ether transfer, the ConsequencePoint is the
	//          CALL statement that transfer ethers.
	//    2.3 for GasEstimationGriefing attacks, ConsequencePoint is first operation that throws out-of-gas exception
	InfluenceTrace []*data_flow.Operation

	Attack *Attack

	// cache
	// msgCallLevelInfluenceTrace is the InfluenceTrace aggreated to MsgCall level
	msgCallLevelInfluenceTrace []*tracers.MsgCall[*data_flow.Data]
	// influenceString is used to identify the vulnerability.
	// Same influenceString means same vulnerability.
	influenceString *string
	// hash of this attack analysis
	hash               *common.Hash
	attackAnalysisBSON *AttackAnalysisBSON
}

// Hash is deterministic, given the same attack and shared variable.
func (a *AttackAnalysis) Hash() common.Hash {
	if a.hash == nil {
		h := crypto.Keccak256Hash(
			a.Attack.Hash().Bytes(),
			[]byte(a.SharedVariable.ID()),
		)
		a.hash = &h
	}
	return *a.hash
}

func (a *AttackAnalysis) MsgCallLevelInfluenceTrace() []*tracers.MsgCall[*data_flow.Data] {
	return a.msgCallLevelInfluenceTrace
}

// InfluenceString is a string to identify the vulnerability exploited by the attack.
// InfluenceString is computed in the following way:
//  1. Excluding tailing token transfers, if this is a ComputationAlteration attack,
//     and the attack consequence is altered token transfer.
//     1.1. Find the last CALL operation, which results in a message call that transfers tokens.
//     1.2. Excluding all operations under this message call.
//  2. Aggregate operations in each message call.
//     2.1. Split the InfluenceTrace into segments by the code hash of the contract that each operation executes on.
//     2.2. Convert each segment to string: <codeHash>:<pc>,<pc>,<pc>,...
//     where pc is the program counter of each operation in the segment and duplicate ones are excluded..
//  3. Concatenate all segments in the order of their appearance in the InfluenceTrace.
func (a *AttackAnalysis) InfluenceString(getCodeHash func(common.Address) common.Hash) string {
	if a.influenceString == nil {
		type Segment []*data_flow.Operation
		var segments []Segment
		var lastCall *tracers.MsgCall[*data_flow.Data]
		var currentSegment Segment = Segment{}
		for _, op := range a.InfluenceTrace {
			if lastCall != nil && !lastCall.Position.Equal(op.MsgCallPosition()) {
				segments = append(segments, currentSegment)
				currentSegment = Segment{}
			}
			currentSegment = append(currentSegment, op)
			lastCall = op.MsgCall()
		}
		segments = append(segments, currentSegment)

		lastOp := a.InfluenceTrace[len(a.InfluenceTrace)-1]
		if a.Pattern == ComputationAlteration && lastOp.OpCode() >= vm.LOG0 &&
			lastOp.OpCode() <= vm.LOG4 {
			// Exclude tailing token transfers
			for i := len(segments) - 1; i >= 0; i-- {
				segment := segments[i]
				msgCall := segment[0].MsgCall()
				if msgCall.OpCode == vm.CALL && IsTokenTransfer(msgCall) {
					segments = segments[:i]
					break
				}
			}
		}

		a.influenceString = lo.ToPtr(
			strings.Join(lo.Map(segments, func(seg Segment, _ int) string {
				pcs := strings.Join(
					lo.Uniq(lo.Map(seg, func(op *data_flow.Operation, _ int) string {
						return strconv.FormatInt(int64(op.PC()), 10)
					})), ",")
				return fmt.Sprintf("%s:%s", getCodeHash(seg[0].CodeAddr()).Hex(), pcs)
			}), "->"),
		)
	}
	return *a.influenceString
}

// Analyze analyzes the attack and the AttackAnalysis structs are stored in Attack.Analysis.
func (a *Attack) Analyze(
	chainReader chain.BlockchainReader,
	session *TxHistorySession,
) (attackAnalysisSet []*AttackAnalysis, err error) {
	defer func() {
		a.Analysis = attackAnalysisSet
	}()

	victimState, victimVMContext :=
		a.VictimTxRecord.State.Copy(), a.VictimTxRecord.VmContext.Copy()
	victimAsIfState, victimAsIfVMContext :=
		a.AttackTxRecord.State.Copy(), a.AttackTxRecord.VmContext.Copy()

	// instance of that defined in attack transaction
	sharedWrites, sharedReads := a.AttackTxRecord.TxSummary.OverallDefs().IntersectWith(
		a.VictimTxRecord.TxSummary.OverallUses(),
	)

	log.Debug().
		Str("attackHash", a.Hash().Hex()).
		Str("attackTx", a.AttackTxRecord.Tx.Hash().Hex()).
		Str("victimTx", a.VictimTxRecord.Tx.Hash().Hex()).
		Msg("Start processing attack case")

	exeVM := engine.NewExeVM(AttackSearchVMConfig())

	// 1: construct reference execution path, which is the path of victim transaction in AF scenario.
	summaryTracerAF := summary.NewTxSummaryTracer(
		summary.Config{IncludeTransfer: true, IncludeTrace: true},
	)
	exeVM.SetTracer(nil)
	// 1.1: Replay the nonce prerequisites of victim transaction, so that victim transaction is not rejected.
	prerequisites := session.SlicePrerequisites(a.VictimTxRecord, a.AttackTxRecord)
	for _, pre := range prerequisites {
		_, _, err = exeVM.ApplyTx(victimAsIfState, pre.Tx, victimAsIfVMContext, false, false)
		if err != nil {
			return nil, err
		}
	}
	// 1.2: Replay the victim transaction in attack-free scenario
	exeVM.SetTracer(summaryTracerAF)
	_, _, err = exeVM.ApplyTx(
		victimAsIfState,
		a.VictimTxRecord.Tx,
		victimAsIfVMContext,
		false,
		true,
	)
	if err != nil {
		return nil, err
	}

	// 2: Construct taint analyzers for all shared variables.
	// The taint analyzers will be used later for victim transaction in attack scenario.
	var analyzersA []data_flow.Analyzer // data flow analyzers for all shared variables in attack scenario
	type sharedVarAnalyzerPair struct {
		// shared variable is the instance of def
		write, read                 summary.StateVariable
		sharedVariable              summary.StateVariable
		originalValue, alteredValue fmt.Stringer
		taintAnalyzer               *TaintAnalyzer
	}
	analysisPairs := make([]*sharedVarAnalyzerPair, len(sharedWrites))
	for i, write := range sharedWrites {
		read := sharedReads[i]
		pair := &sharedVarAnalyzerPair{write: write, read: read}
		analysisPairs[i] = pair
		pair.sharedVariable = read

		// obtain the original and altered value of the shared variable
		switch sv := write.(type) {
		case summary.BalanceVariable:
			pair.originalValue = a.AttackTxRecord.State.GetBalance(sv.Address)
			pair.alteredValue = a.VictimTxRecord.State.GetBalance(sv.Address)
		case summary.StorageVariable:
			pair.originalValue = a.AttackTxRecord.State.GetState(sv.Address, sv.Storage)
			pair.alteredValue = a.VictimTxRecord.State.GetState(sv.Address, sv.Storage)
		case summary.CodeVariable:
			originalBuilder := &strings.Builder{}
			_, _ = originalBuilder.WriteString(hexutil.Encode(a.AttackTxRecord.State.GetCode(sv.Address)))
			pair.originalValue = originalBuilder
			alteredBuilder := &strings.Builder{}
			_, _ = alteredBuilder.WriteString(hexutil.Encode(a.VictimTxRecord.State.GetCode(sv.Address)))
			pair.alteredValue = alteredBuilder
		default:
			panic("unexpected shared variable type")
		}

		// construct taint analysis analyzer
		pair.taintAnalyzer = NewTaintAnalyzer(
			victimState,
			read,
			summaryTracerAF.Summary.FlattenedExecutionPath(),
		)
		analyzersA = append(analyzersA, pair.taintAnalyzer)

		// construct addressing path analyzer
		if sharedStorageVariable, ok := read.(summary.StorageVariable); ok {
			sharedStorageVariableWithPaths := &StorageVariableWithAddressingPaths{
				StorageVariable: sharedStorageVariable,
			}
			pair.sharedVariable = sharedStorageVariableWithPaths
			addressingPathAnalyzer := &storage_address.StorageAddressingPathAnalyzer{
				StorageVariable: &sharedStorageVariable,
				OnStorageStoredOrLoaded: func(op vm.OpCode, addressingPathCandidates []storage_address.AddressingPath) {
				nextCandidate:
					for _, candidate := range addressingPathCandidates {
						if candidate.Opcode() == vm.SLOAD {
							// save LOAD addressing path of the sharedVar
							for _, existing := range sharedStorageVariableWithPaths.AddressingPaths {
								if existing.Equal(candidate) {
									continue nextCandidate
								}
							}
							sharedStorageVariableWithPaths.AddressingPaths = append(
								sharedStorageVariableWithPaths.AddressingPaths,
								candidate,
							)
						}
					}
				},
			}
			analyzersA = append(analyzersA, addressingPathAnalyzer)
		}
	}

	// 2: Construct tracers for victim transaction in attack scenario
	summaryTracerA := summary.NewTxSummaryTracer(summary.Config{
		IncludeTransfer: true, IncludeTrace: true,
	})
	tracerA := tracers.CombineTracers(
		summaryTracerA,
		data_flow.NewDataFlowTracer(analyzersA...),
	)

	// 3: Replay the victim transaction in attack scenario
	exeVM.SetTracer(tracerA)
	_, _, err = exeVM.ApplyTx(victimState, a.VictimTxRecord.Tx, victimVMContext, false, true)
	if err != nil {
		return nil, err
	}

	// 4: Analyze each shared variable and generate AttackAnalysis result.
	// 4.1: find the consequence point between the execution path of A and AF.
	// the consequence point is the nearest conditional statement that
	// is control-depended by the victim profit operation.
	// if there is no such statement, the consequence point is the victim profit operation.
	// 4.1.1: find all transfers that are related to victim in victim transaction
	// in both A and AF scenario.
	var transfersA, transfersAF []summary.ITransfer
	for _, transfer := range summaryTracerAF.Summary.OverallTransfers() {
		if transfer.From() == a.Victim || transfer.To() == a.Victim {
			transfersAF = append(transfersAF, transfer)
		}
	}
	for _, transfer := range summaryTracerA.Summary.OverallTransfers() {
		if transfer.From() == a.Victim || transfer.To() == a.Victim {
			transfersA = append(transfersA, transfer)
		}
	}
	// 4.1.2: build the consequenceInside function which returns the victim profits inside the basic block.
	buildConsequenceInsideFn := func(transfers []summary.ITransfer) consequenceProfitsGetter {
		return func(block tracers.ITraceBlock) summary.Profits {
			var consequence summary.Profits = nil
			for _, transfer := range transfers {
				if transfer.Location() != nil && blockContains(block, transfer.Location()) {
					ps := summary.Profits(transfer.Profits())
					consequence = consequence.Add(ps.ProfitsOf(a.Victim)...)
				}
			}
			return consequence
		}
	}
	// 4.1.2: find the consequence point
	consequencePoints, _, diverge := a.locateConsequencePoint(
		summaryTracerA.Summary.FlattenedExecutionPath(),
		summaryTracerAF.Summary.FlattenedExecutionPath(),
		buildConsequenceInsideFn(transfersA),
		buildConsequenceInsideFn(transfersAF),
		summaryTracerA.RootMsgCall().FindMsgCall,
		summaryTracerAF.RootMsgCall().FindMsgCall,
	)
	if len(consequencePoints) <= 0 {
		// usualy this should not happen
		return attackAnalysisSet, nil
	}

	var pattern AttackPattern
	if diverge {
		if a.VictimTxRecord.TxSummary.MsgCall().
			OutOfGas() !=
			a.VictimTxAsIfSummary.MsgCall().
				OutOfGas() {
			pattern = GasEstimationGriefing
		} else {
			pattern = PathConditionAlteration
		}
	} else {
		pattern = ComputationAlteration
	}

analysisPairLoop:
	for _, pair := range analysisPairs {
		if pair == nil {
			continue
		}
		// 4.2: check if the consequence point is tainted.
		// If it is, find the influence trace, which is the data flow call trace
		// from where the shared variable is loaded to the consequence point.
		var taintSource data_flow.FlowNode
		var influenceTrace []*data_flow.Operation
		var callLevelInfluenceTrace []*tracers.MsgCall[*data_flow.Data]
		var taintedSinks map[uint]data_flow.FlowNode

		// find the last consequence block that is tainted
		var consequenceLocation tracers.TraceLocation
		var sink data_flow.FlowNode
	nextConsequencePoint:
		for i := len(consequencePoints) - 1; i >= 0; i-- {
			consequencePoint := consequencePoints[i]
			consequenceBlock := summaryTracerA.Summary.FlattenedExecutionPath()[consequencePoint]
			if diverge {
				consequenceLocation = consequenceBlock.Tail()
			} else {
				// search for consequence opcode
				for _, location := range consequenceBlock.Content() {
					for _, transfer := range transfersA {
						if transfer.Location() != nil && transfer.Location().Index() == location.Index() {
							consequenceLocation = location
							goto checkTaint
						}
					}
				}
				continue
			}
		checkTaint:
			switch consequenceLocation.OpCode() {
			case vm.JUMP, vm.JUMPI:
				taintedSinks = pair.taintAnalyzer.Jumps
			case vm.CALL:
				taintedSinks = pair.taintAnalyzer.Calls
			case vm.LOG0, vm.LOG1, vm.LOG2, vm.LOG3, vm.LOG4:
				taintedSinks = pair.taintAnalyzer.Logs
			default:
				log.Debug().
					Str("op", consequenceLocation.OpCode().String()).
					Msg("unexpected consequence opcode")
				continue nextConsequencePoint
			}
			var exist bool
			if sink, exist = taintedSinks[consequenceLocation.Index()]; exist {
				goto locateInfluenceTrace
			}
		}
		// if there is no tainted consequence block, skip this pair
		// no sink found, meaning that this consequence is not tainted by the altered shared variable
		continue analysisPairLoop

	locateInfluenceTrace:
		// find which sink is corresponding to the consequence
		taintSource, influenceTrace, callLevelInfluenceTrace = a.locateVarReadAndInfluenceTrace(
			sink,
			func(node data_flow.FlowNode) bool {
				return pair.taintAnalyzer.Sources[node.Operation().Index()]
			},
		)
		if taintSource == nil {
			continue analysisPairLoop
		}

		// 5: get the read/write location of the shared variable
		var writeLocation = pair.write.Location()
		// We use the taint source instead of pair.read since the shared variable can be read multiple times.
		// Only the taint source is the actual read location.
		var readLocation = taintSource.Operation()

		// 7: construct the attack analysis result
		analysis := &AttackAnalysis{
			Pattern:          pattern,
			SharedVariable:   pair.sharedVariable,
			OriginalValue:    pair.originalValue,
			AlteredValue:     pair.alteredValue,
			WritePoint:       writeLocation,
			ReadPoint:        readLocation,
			ConsequencePoint: consequenceLocation,
			InfluenceTrace:   influenceTrace,

			Attack: a,

			msgCallLevelInfluenceTrace: callLevelInfluenceTrace,
		}
		attackAnalysisSet = append(attackAnalysisSet, analysis)
	}
	return attackAnalysisSet, nil
}

func (a *Attack) locateVarReadAndInfluenceTrace(
	sink data_flow.FlowNode,
	isSource func(data_flow.FlowNode) bool,
) (
	source data_flow.FlowNode,
	influenceTrace []*data_flow.Operation,
	msgCallLevelInfluenceTrace []*tracers.MsgCall[*data_flow.Data],
) {
	dupNode := make(map[data_flow.NodeID]bool)
	var innerFind func(
		[]*data_flow.Operation,
		[]*tracers.MsgCall[*data_flow.Data],
		data_flow.FlowNode,
	) (data_flow.FlowNode, []*data_flow.Operation, []*tracers.MsgCall[*data_flow.Data])
	innerFind = func(
		trace []*data_flow.Operation, cTrace []*tracers.MsgCall[*data_flow.Data], node data_flow.FlowNode,
	) (data_flow.FlowNode, []*data_flow.Operation, []*tracers.MsgCall[*data_flow.Data]) {
		if _, ok := dupNode[node.ID()]; ok {
			return nil, nil, nil
		}
		dupNode[node.ID()] = true

		trace = append(trace, node.Operation())
		// append trace when entered a new message call
		if len(cTrace) == 0 ||
			node.Operation().MsgCall().Position.Cmp(cTrace[len(cTrace)-1].Position) != 0 {
			cTrace = append(cTrace, node.Operation().MsgCall())
		}

		fromNodes := node.From()
		if isSource(node) {
			// no flow from, this is the taint source
			oTrace := make([]*data_flow.Operation, len(trace))
			copy(oTrace, trace)
			oTrace = lo.Reverse(oTrace)
			callTrace := make([]*tracers.MsgCall[*data_flow.Data], len(cTrace))
			copy(callTrace, cTrace)
			callTrace = lo.Reverse(callTrace)
			return node, oTrace, callTrace
		} else {
			// check the upstream nodes
			for _, from := range fromNodes {
				var cpOTrace = make([]*data_flow.Operation, len(trace))
				copy(cpOTrace, trace)
				var cpTrace = make([]*tracers.MsgCall[*data_flow.Data], len(cTrace))
				copy(cpTrace, cTrace)
				n, opTrace, callTrace := innerFind(cpOTrace, cpTrace, from)
				if n != nil && callTrace != nil {
					return n, opTrace, callTrace
				}
			}
		}

		return nil, nil, nil
	}

	if sink.Operation().OpCode() == vm.CALL && sink.Operation().Arg(2).Int.Sign() > 0 {
		// if the sink is an ether transfer
		// find child call
		var call *tracers.MsgCall[*data_flow.Data]
		for _, child := range sink.Operation().MsgCall().NestedCalls {
			if child.Caller.CallSite.Index() == sink.Operation().Index() {
				call = child
				break
			}
		}
		return innerFind(
			[]*data_flow.Operation{},
			[]*tracers.MsgCall[*data_flow.Data]{call},
			sink,
		)
	} else {
		return innerFind(
			[]*data_flow.Operation{},
			[]*tracers.MsgCall[*data_flow.Data]{},
			sink,
		)
	}
}

type msgCallFinder[D any] func(position1 tracers.CallPosition) *tracers.MsgCall[D]

type consequenceProfitsGetter func(block tracers.ITraceBlock) summary.Profits

func (a *Attack) locateConsequencePoint(
	path1, path2 tracers.Trace,
	consequenceInside1, consequenceInside2 consequenceProfitsGetter,
	getMsgCall1, getMsgCall2 msgCallFinder[*summary.CallSummary],
) (path1Indices []int, path2Indices []int, divergePoint bool) {
	var divergeIndices1, divergeIndices2 []int
	var profitDifferIndices1, profitDifferIndices2 []int

	var profits1, profits2 summary.Profits

	sameProfits := func() bool {
		cmp, err := profits2.Cmp(profits1)
		if err != nil {
			return false
		}
		return cmp == 0
	}
	updateProfits := func(i1, j1, i2, j2 int) (updated bool) {
		reverts1 := func(b tracers.ITraceBlock) bool {
			call := getMsgCall1(b.Tail().MsgCallPosition())
			return call.Result.Unwrap() != nil
		}
		reverts2 := func(b tracers.ITraceBlock) bool {
			call := getMsgCall2(b.Tail().MsgCallPosition())
			return call.Result.Unwrap() != nil
		}
		var lastProfitsIndex1, lastProfitsIndex2 = -1, -1
		var localProfits1, localProfits2 summary.Profits
		for k1 := i1; k1 < j1; k1++ {
			if reverts1(path1[k1]) {
				updated = true
				profits1 = nil
			}
			if p := consequenceInside1(path1[k1]); p != nil {
				updated = true
				lastProfitsIndex1 = k1
				localProfits1 = localProfits1.Add(p...)
			}
		}
		for k2 := i2; k2 < j2; k2++ {
			if reverts2(path2[k2]) {
				updated = true
				profits2 = nil
			}
			if p := consequenceInside2(path2[k2]); p != nil {
				updated = true
				lastProfitsIndex2 = k2
				localProfits2 = localProfits2.Add(p...)
			}
		}
		profits1 = profits1.Add(localProfits1...)
		profits2 = profits2.Add(localProfits2...)
		if cmp, err := localProfits2.Cmp(localProfits1); err != nil || cmp != 0 {
			if lastProfitsIndex1 >= 0 {
				profitDifferIndices1 = append(profitDifferIndices1, lastProfitsIndex1)
			}
			if lastProfitsIndex2 >= 0 {
				profitDifferIndices2 = append(profitDifferIndices2, lastProfitsIndex2)
			}
		}
		return updated
	}
	// diverge point are the last basic block that both appear in the path1 and path2
proceedTogether:
	for i1, i2 := 0, 0; i1 < len(path1) && i2 < len(path2); i1, i2 = i1+1, i2+1 {
		if blocksEqual(path1[i1], path2[i2]) {
			updateProfits(i1, i1+1, i2, i2+1)
			continue proceedTogether
		}
		// path diverges here, find next merge point
		divergeIndices1 = append(divergeIndices1, i1-1)
		divergeIndices2 = append(divergeIndices2, i2-1)
		for j1 := i1; j1 < len(path1); j1++ {
			for j2 := i2; j2 < len(path2); j2++ {
				if !blocksEqual(path1[j1], path2[j2]) {
					continue
				}
				// check if any profit occurs in the branch
				updated := updateProfits(i1, j1, i2, j2)
				// if the profit becomes different after diverge
				if updated && !sameProfits() {
					return divergeIndices1, divergeIndices2, true
				}
				i1, i2 = j1-1, j2-1
				continue proceedTogether
			}
		}
		// two paths never merge after this
		updated := updateProfits(i1, len(path1), i2, len(path2))
		// profits must be different (since this is a real attack)
		if updated {
			return divergeIndices1, divergeIndices2, true
		} else {
			return profitDifferIndices1, profitDifferIndices2, false
		}
	}
	// exact same execution path (different profits)
	return profitDifferIndices1, profitDifferIndices2, false
}

func blocksEqual(b1, b2 tracers.ITraceBlock) bool {
	return b1.CodeAddr() == b2.CodeAddr() &&
		b1.Head().PC() == b2.Head().PC() &&
		b1.Head().MsgCallPosition().Cmp(b2.Head().MsgCallPosition()) == 0
}

func blockContains(b tracers.ITraceBlock, loc tracers.TraceLocation) bool {
	return b.CodeAddr() == loc.CodeAddr() &&
		b.Head().PC() <= loc.PC() && loc.PC() <= b.Tail().PC() &&
		b.Head().
			GasAvailable() >=
			loc.GasAvailable() && loc.GasAvailable() >= b.Tail().GasAvailable()
}
