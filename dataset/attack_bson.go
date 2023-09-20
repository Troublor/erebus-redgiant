package dataset

import (
	"bytes"

	"github.com/Troublor/erebus-redgiant/analysis/data_flow"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/Troublor/erebus-redgiant/analysis/storage_address"
	"github.com/Troublor/erebus-redgiant/analysis/summary"
	"github.com/Troublor/erebus-redgiant/contract"
	"github.com/Troublor/erebus-redgiant/dyengine/tracers"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
)

func (a *Attack) AsAttackBSON() *AttackBSON {
	if a.attackBSON == nil {
		var profitTx *string
		if a.ProfitTxRecord != nil {
			tx := a.ProfitTxRecord.Tx.Hash().Hex()
			profitTx = &tx
		}
		analysisSet := lo.Map(a.Analysis, func(analysis *AttackAnalysis, _ int) AttackAnalysisBSON {
			return *analysis.AsAttackAnalysisBSON()
		})
		a.attackBSON = &AttackBSON{
			Hash:  a.Hash().Hex(),
			Block: a.AttackTxRecord.TxSummary.MsgCall().Receipt.BlockNumber.Uint64(),

			Attacker: a.Attacker.Hex(),
			Victim:   a.Victim.Hex(),
			AttackTx: a.AttackTxRecord.Tx.Hash().Hex(),
			VictimTx: a.VictimTxRecord.Tx.Hash().Hex(),
			ProfitTx: profitTx,
			AttackerProfits: TwoScenarioProfits{
				Attack:     a.attackerProfits,
				AttackFree: a.attackerAsIfProfits,
			},
			VictimProfits: TwoScenarioProfits{
				Attack:     a.victimProfits,
				AttackFree: a.victimAsIfProfits,
			},
			//OutOfGas:       a.OutOfGas,
			//MismatchOracle: a.MismatchOracle,

			Analysis: analysisSet,

			attack: a,
		}
	}
	return a.attackBSON
}

type TwoScenarioProfits struct {
	Attack     summary.Profits `bson:"attack"`
	AttackFree summary.Profits `bson:"attackFree"`
}

// AttackBSON is the BSON-serializable version of Attack.
type AttackBSON struct {
	Hash  string `bson:"hash"`
	Block uint64 `bson:"block"` // the block number of attack tx

	Attacker        string             `bson:"attacker"`
	Victim          string             `bson:"victim"`
	AttackTx        string             `bson:"attackTx"`
	VictimTx        string             `bson:"victimTx"`
	ProfitTx        *string            `bson:"profitTx,omitempty"`
	AttackerProfits TwoScenarioProfits `bson:"attackerProfits"`
	VictimProfits   TwoScenarioProfits `bson:"victimProfits"`
	//OutOfGas        bool               `bson:"outOfGas"`
	//MismatchOracle  bool               `bson:"mismatchOracle,omitempty"`

	// AttackAnalysisBSON associated with this Attack
	Analysis []AttackAnalysisBSON `bson:"analysis"`

	// cache
	attack *Attack
}

type InfluenceTraceBSON struct {
	MsgCallLevel   []MsgCallBSON  `bson:"msgCallLevel"`
	OperationLevel []LocationBSON `bson:"operationLevel"`
}

// AttackAnalysisBSON is the BSON-serializable version of AttackAnalysis.
type AttackAnalysisBSON struct {
	Hash    string `bson:"hash"`
	Pattern string `bson:"pattern"`

	SharedVariable      summary.StateVariable    `bson:"sharedVariable"`
	OriginalValue       string                   `bson:"originalValue"`
	AlteredValue        string                   `bson:"alteredValue"`
	WriteLocation       *LocationWithMsgCallBSON `bson:"writeLocation"`
	ReadLocation        *LocationWithMsgCallBSON `bson:"readLocation"`
	ConsequenceLocation *LocationWithMsgCallBSON `bson:"consequenceLocation"`
	InfluenceTrace      InfluenceTraceBSON       `bson:"influenceTrace"`

	InfluenceString string `bson:"influenceString"`
}

func (s StorageVariableWithAddressingPaths) MarshalBSON() ([]byte, error) {
	var cp = struct {
		Address         string                           `bson:"address"`
		Storage         string                           `bson:"storage"`
		AddressingPaths []storage_address.AddressingPath `bson:"addressingPaths"`
		Value           string                           `bson:"value"`
		Type            summary.StateVarType             `bson:"type"`
	}{s.Address.Hex(), s.Storage.Hex(), s.AddressingPaths, s.Value.Hex(), s.Type()}
	return bson.Marshal(cp)
}

func (a *AttackAnalysis) AsAttackAnalysisBSON() *AttackAnalysisBSON {
	if a.attackAnalysisBSON == nil {
		influenceTraceBson := InfluenceTraceBSON{
			MsgCallLevel: lo.Map(
				a.MsgCallLevelInfluenceTrace(),
				func(c *tracers.MsgCall[*data_flow.Data], _ int) MsgCallBSON {
					return *MsgCallAsBSON(c)
				},
			),
			OperationLevel: lo.Map(
				a.InfluenceTrace,
				func(op *data_flow.Operation, _ int) LocationBSON {
					return *TraceLocationAsBSON(op)
				},
			),
		}
		a.attackAnalysisBSON = &AttackAnalysisBSON{
			Hash:           a.Hash().Hex(),
			Pattern:        string(a.Pattern),
			SharedVariable: a.SharedVariable,
			OriginalValue:  a.OriginalValue.String(),
			AlteredValue:   a.AlteredValue.String(),
			WriteLocation: TraceLocationAsBSONWithMsgCall(
				a.WritePoint,
				a.Attack.AttackTxRecord.TxSummary.MsgCall().FindMsgCall,
			),
			ReadLocation: TraceLocationAsBSONWithMsgCall(
				a.ReadPoint,
				a.Attack.VictimTxRecord.TxSummary.MsgCall().FindMsgCall,
			),
			ConsequenceLocation: TraceLocationAsBSONWithMsgCall(
				a.ConsequencePoint,
				a.Attack.VictimTxRecord.TxSummary.MsgCall().FindMsgCall,
			),
			InfluenceTrace:  influenceTraceBson,
			InfluenceString: a.InfluenceString(a.Attack.VictimTxRecord.State.GetCodeHash),
		}
	}
	return a.attackAnalysisBSON
}

type LocationBSON struct {
	MsgCallPosition string `bson:"msgCallPosition"`
	CodeAddr        string `bson:"codeAddr"`
	PC              uint64 `bson:"pc"`
	OpCode          string `bson:"op"`
	GasAvailable    uint64 `bson:"gasAvailable"`
	GasUsed         uint64 `bson:"gasUsed"`
	Index           uint   `bson:"index"`
}

type LocationWithMsgCallBSON struct {
	MsgCallBSON     MsgCallBSON `bson:"msgCall"`
	MsgCallPosition string      `bson:"msgCallPosition"`
	CodeAddr        string      `bson:"codeAddr"`
	PC              uint64      `bson:"pc"`
	OpCode          string      `bson:"op"`
	GasAvailable    uint64      `bson:"gasAvailable"`
	GasUsed         uint64      `bson:"gasUsed"`
	Index           uint        `bson:"index"`
}

func TraceLocationAsBSON(loc tracers.TraceLocation) *LocationBSON {
	if loc == nil {
		return nil
	}
	return &LocationBSON{
		MsgCallPosition: loc.MsgCallPosition().String(),
		CodeAddr:        loc.CodeAddr().Hex(),
		PC:              loc.PC(),
		OpCode:          loc.OpCode().String(),
		GasAvailable:    loc.GasAvailable(),
		GasUsed:         loc.GasUsed(),
		Index:           loc.Index(),
	}
}

func TraceLocationAsBSONWithMsgCall[D any](
	loc tracers.TraceLocation,
	msgCallFinder func(position1 tracers.CallPosition) *tracers.MsgCall[D],
) *LocationWithMsgCallBSON {
	if loc == nil {
		return nil
	}
	return &LocationWithMsgCallBSON{
		MsgCallBSON:     *MsgCallAsBSON(msgCallFinder(loc.MsgCallPosition())),
		MsgCallPosition: loc.MsgCallPosition().String(),
		CodeAddr:        loc.CodeAddr().Hex(),
		PC:              loc.PC(),
		OpCode:          loc.OpCode().String(),
		GasAvailable:    loc.GasAvailable(),
		GasUsed:         loc.GasUsed(),
		Index:           loc.Index(),
	}
}

type MsgCallCallerBSON struct {
	CodeAddr  *string       `bson:"codeAddr"`
	StateAddr string        `bson:"stateAddr"`
	CallSite  *LocationBSON `bson:"callSite"`
}

func MsgCallCallerAsBSON(caller tracers.MsgCallCaller) *MsgCallCallerBSON {
	var codeAddr *string
	if caller.CodeAddr != nil {
		codeAddr = lo.ToPtr(caller.CodeAddr.Hex())
	}
	var callSite *LocationBSON
	if caller.CallSite != nil {
		callSite = TraceLocationAsBSON(caller.CallSite)
	}
	return &MsgCallCallerBSON{
		CodeAddr:  codeAddr,
		StateAddr: caller.StateAddr.Hex(),
		CallSite:  callSite,
	}
}

type MsgCallBSON struct {
	Position string `bson:"position"`
	OpCode   string `bson:"type"`

	Caller      MsgCallCallerBSON `bson:"caller"`
	StateAddr   string            `bson:"stateAddr"`
	CodeAddr    string            `bson:"codeAddr"`
	Precompiled bool              `bson:"precompiled"`
	Input       string            `bson:"input"`
	Value       string            `bson:"value"`

	UsedGas    uint64  `bson:"usedGas"`
	Err        *string `bson:"err,omitempty"`
	ReturnData string  `bson:"return"`
}

func MsgCallAsBSON[D any](call *tracers.MsgCall[D]) *MsgCallBSON {
	if call == nil {
		return nil
	}
	var err *string
	if call.Result.Unwrap() != nil {
		e := call.Result.Unwrap().Error()
		err = &e
	}
	var returnData string
	if call.Result.Failed() {
		returnData = hexutil.Encode(call.Result.Revert())
	} else {
		returnData = hexutil.Encode(call.Result.Return())
	}
	return &MsgCallBSON{
		Position: call.Position.String(),

		OpCode:      call.OpCode.String(),
		Caller:      *MsgCallCallerAsBSON(call.Caller),
		StateAddr:   call.StateAddr.Hex(),
		CodeAddr:    call.CodeAddr.Hex(),
		Precompiled: call.Precompiled,
		Input:       hexutil.Encode(call.Input),
		Value:       call.Value.String(),

		UsedGas:    call.Result.UsedGas,
		Err:        err,
		ReturnData: returnData,
	}
}

func IsTransfer(call *tracers.MsgCall[*data_flow.Data]) bool {
	// Ether
	if call.OpCode == vm.CALL && call.Value.Sign() > 0 && len(call.Input) == 0 {
		return true
	}
	return IsTokenTransfer(call)
}

func IsTokenTransfer(call *tracers.MsgCall[*data_flow.Data]) bool {
	// ERC20
	if IsMethodInvocation(contract.ERC20ABI.Methods["transfer"], call) {
		return true
	}
	if IsMethodInvocation(contract.ERC20ABI.Methods["transferFrom"], call) {
		return true
	}
	// ERC721
	if IsMethodInvocation(contract.ERC721ABI.Methods["safeTransferFrom"], call) {
		return true
	}
	// overload of safeTransferFrom function
	if IsMethodInvocation(contract.ERC721ABI.Methods["safeTransferFrom0"], call) {
		return true
	}
	if IsMethodInvocation(contract.ERC721ABI.Methods["transferFrom"], call) {
		return true
	}
	// ERC777
	if IsMethodInvocation(contract.ERC777ABI.Methods["send"], call) {
		return true
	}
	if IsMethodInvocation(contract.ERC777ABI.Methods["operatorSend"], call) {
		return true
	}
	if IsMethodInvocation(contract.ERC777ABI.Methods["burn"], call) {
		return true
	}
	if IsMethodInvocation(contract.ERC777ABI.Methods["operatorBurn"], call) {
		return true
	}
	// ERC1155
	if IsMethodInvocation(contract.ERC1155ABI.Methods["safeTransferFrom"], call) {
		return true
	}
	if IsMethodInvocation(contract.ERC1155ABI.Methods["safeBatchTransferFrom"], call) {
		return true
	}
	// WETH9
	if call.StateAddr == contract.WETH9Address &&
		IsMethodInvocation(contract.WETH9ABI.Methods["deposit"], call) {
		return true
	}
	if call.StateAddr == contract.WETH9Address &&
		IsMethodInvocation(contract.WETH9ABI.Methods["withdraw"], call) {
		return true
	}
	return false
}

func IsMethodInvocation[D any](method abi.Method, call *tracers.MsgCall[D]) bool {
	switch method.Type {
	case abi.Receive, abi.Fallback:
		return true
	case abi.Function:
		if len(call.Input) < 4 {
			return false
		}
		fnSig := call.Input[:4]
		return bytes.Equal(method.ID, fnSig)
	default:
		return false
	}
}
