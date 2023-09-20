package summary

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/Troublor/erebus-redgiant/dyengine/tracers"

	"github.com/ethereum/go-ethereum/core/types"

	. "github.com/Troublor/erebus-redgiant/contract"
	"github.com/ethereum/go-ethereum/common"
)

type AssetType = string

const (
	ZeroAsset    AssetType = "ZERO"
	EtherAsset   AssetType = "ETHER"
	ERC20Asset   AssetType = "ERC20_TOKEN"
	ERC777Asset  AssetType = "ERC777_TOKEN"
	ERC721Asset  AssetType = "ERC721_TOKEN"
	ERC1155Asset AssetType = "ERC1155_TOKEN"
)

var AllAssetTypes = [5]AssetType{EtherAsset, ERC20Asset, ERC777Asset, ERC721Asset, ERC1155Asset}

var ErrMergeNotPossible = errors.New("merge not possible")

type ITransfer interface {
	// Type returns the type of the transfer
	Type() AssetType

	// Profits returns an array of Profit according to the ITransfer.
	// If the ITransfer is Zero, nil is returned.
	// Zero Profit should not be returned.
	Profits() []Profit

	// Copy returns a copy of the ITransfer.
	Copy() ITransfer

	// From returns the from address of the transfer.
	// If Mint is true, From should return zero address (common.Address{}).
	From() common.Address

	// To returns the to address of the transfer.
	// If Burn is true, To should return zero address (common.Address{}).
	To() common.Address

	// Burn returns true if the value is burnt in the ITransfer.
	Burn() bool

	// Mint returns true if the value is minted in the ITransfer.
	Mint() bool

	// Merge merges the current ITransfer with the given ITransfer.
	// If the two ITransfer is reverse of each other, a Zero ITransfer is returned.
	// If one ITransfer has the same From as the To of the other ITransfer,
	// the merged ITransfer will be the logically connected ITransfer.
	// If From and To of the two ITransfer is the same, the value of the two ITransfer will be added.
	// The merged ITransfer should be a copy, modifying which should not affect the original ITransfer.
	// Zero ITransfer should not be returned.
	// For other cases, ErrMergeNotPossible error is returned.
	Merge(other ITransfer) ([]ITransfer, error)

	// Zero returns true if the ITransfer's underlying value is zero or From is equal to the value of To.
	Zero() bool

	// location returns the location in the codeAddr where the transfer takes place
	// Location could be nil if the transfer is EtherTransfer initiated by transaction itself.
	Location() tracers.TraceLocation
}

type fungibleTransfer struct {
	FromAccount      common.Address        `json:"from"`
	ToAccount        common.Address        `json:"to"`
	Amount           *big.Int              `json:"amount"`
	ContractLocation tracers.TraceLocation `json:"location"`
}

func (f fungibleTransfer) Location() tracers.TraceLocation {
	return f.ContractLocation
}

func (f fungibleTransfer) copy() fungibleTransfer {
	return fungibleTransfer{
		FromAccount: f.FromAccount,
		ToAccount:   f.ToAccount,
		Amount:      new(big.Int).Set(f.Amount),
	}
}

// merge two transfers. Zero transfer will not be returned.
func (f fungibleTransfer) merge(other fungibleTransfer) ([]fungibleTransfer, error) {
	var transfers []fungibleTransfer
	otherEtherTransfer := other
	if f.FromAccount == otherEtherTransfer.ToAccount &&
		f.ToAccount == otherEtherTransfer.FromAccount {
		// mutual transfer
		switch f.Amount.Cmp(otherEtherTransfer.Amount) {
		case 0:
			break
		case 1:
			transfer := fungibleTransfer{
				FromAccount: f.FromAccount,
				ToAccount:   otherEtherTransfer.FromAccount,
				Amount:      new(big.Int).Sub(f.Amount, otherEtherTransfer.Amount),
			}
			transfers = append(transfers, transfer)
		case -1:
			transfer := fungibleTransfer{
				FromAccount: otherEtherTransfer.FromAccount,
				ToAccount:   f.FromAccount,
				Amount:      new(big.Int).Sub(otherEtherTransfer.Amount, f.Amount),
			}
			transfers = append(transfers, transfer)
		}
	} else if f.FromAccount == otherEtherTransfer.FromAccount && f.ToAccount == otherEtherTransfer.ToAccount {
		// same transfer
		transfer := fungibleTransfer{
			FromAccount: f.FromAccount,
			ToAccount:   f.ToAccount,
			Amount:      new(big.Int).Add(f.Amount, otherEtherTransfer.Amount),
		}
		transfers = append(transfers, transfer)
	} else if f.FromAccount == otherEtherTransfer.ToAccount {
		switch f.Amount.Cmp(otherEtherTransfer.Amount) {
		case 0:
			transfer := fungibleTransfer{
				FromAccount: otherEtherTransfer.FromAccount,
				ToAccount:   f.ToAccount,
				Amount:      new(big.Int).Set(f.Amount),
			}
			transfers = append(transfers, transfer)
		case 1:
			passThroughTransfer := fungibleTransfer{
				FromAccount: otherEtherTransfer.FromAccount,
				ToAccount:   f.ToAccount,
				Amount:      new(big.Int).Set(otherEtherTransfer.Amount),
			}
			transfers = append(transfers, passThroughTransfer)
			additionalTransfer := fungibleTransfer{
				FromAccount: f.FromAccount,
				ToAccount:   f.ToAccount,
				Amount:      new(big.Int).Sub(f.Amount, otherEtherTransfer.Amount),
			}
			transfers = append(transfers, additionalTransfer)
		case -1:
			passThroughTransfer := fungibleTransfer{
				FromAccount: otherEtherTransfer.FromAccount,
				ToAccount:   f.ToAccount,
				Amount:      new(big.Int).Set(f.Amount),
			}
			transfers = append(transfers, passThroughTransfer)
			leftTransfer := fungibleTransfer{
				FromAccount: otherEtherTransfer.FromAccount,
				ToAccount:   f.FromAccount,
				Amount:      new(big.Int).Sub(otherEtherTransfer.Amount, f.Amount),
			}
			transfers = append(transfers, leftTransfer)
		}
	} else if f.ToAccount == otherEtherTransfer.FromAccount {
		merged, err := otherEtherTransfer.merge(f)
		if err != nil {
			return nil, err
		}
		transfers = append(transfers, merged...)
	} else {
		return nil, ErrMergeNotPossible
	}
	return transfers, nil
}

type EtherTransfer struct {
	fungibleTransfer
}

func (e EtherTransfer) MarshalJSON() ([]byte, error) {
	type Alias EtherTransfer
	return json.Marshal(&struct {
		Alias
		Type AssetType `json:"type"`
	}{
		Alias: (Alias)(e),
		Type:  e.Type(),
	})
}

func (e EtherTransfer) Type() AssetType {
	return EtherAsset
}

func (e EtherTransfer) Profits() []Profit {
	if e.Zero() {
		return nil
	}
	fromProfit := EtherProfit{
		Account: e.FromAccount,
		Amount:  new(big.Int).Neg(e.Amount),
	}
	toProfit := EtherProfit{
		Account: e.ToAccount,
		Amount:  e.Amount,
	}
	return []Profit{fromProfit, toProfit}
}

func (e EtherTransfer) Copy() ITransfer {
	return EtherTransfer{
		fungibleTransfer: e.fungibleTransfer.copy(),
	}
}

func (e EtherTransfer) From() common.Address {
	return e.FromAccount
}

func (e EtherTransfer) To() common.Address {
	return e.ToAccount
}

func (e EtherTransfer) Burn() bool {
	return false
}

func (e EtherTransfer) Mint() bool {
	return false
}

func (e EtherTransfer) Merge(other ITransfer) ([]ITransfer, error) {
	if other.Type() != e.Type() {
		return nil, ErrMergeNotPossible
	}
	otherEtherTransfer := other.(EtherTransfer)
	merged, err := e.merge(otherEtherTransfer.fungibleTransfer)
	if err != nil {
		return nil, err
	}
	var transfers = make([]ITransfer, len(merged))
	for i, transfer := range merged {
		transfers[i] = EtherTransfer{transfer}
	}
	return transfers, nil
}

func (e EtherTransfer) Zero() bool {
	return e.Amount.Cmp(big.NewInt(0)) == 0 || e.FromAccount == e.ToAccount
}

type ERC20Transfer struct {
	fungibleTransfer
	Contract common.Address `json:"contract"`
}

func (t ERC20Transfer) MarshalJSON() ([]byte, error) {
	type Alias ERC20Transfer
	return json.Marshal(&struct {
		Alias
		Type AssetType `json:"type"`
	}{
		Alias: (Alias)(t),
		Type:  t.Type(),
	})
}

func (t ERC20Transfer) Type() AssetType {
	return ERC20Asset
}

func (t ERC20Transfer) Profits() []Profit {
	if t.Zero() {
		return nil
	}
	var profits []Profit
	if !t.Mint() {
		fromProfit := ERC20Profit{
			Account:  t.FromAccount,
			Contract: t.Contract,
			Amount:   new(big.Int).Neg(t.Amount),
		}
		profits = append(profits, fromProfit)
	}
	if !t.Burn() {
		toProfit := ERC20Profit{
			Account:  t.ToAccount,
			Contract: t.Contract,
			Amount:   t.Amount,
		}
		profits = append(profits, toProfit)
	}
	return profits
}

func (t ERC20Transfer) Copy() ITransfer {
	return ERC20Transfer{
		fungibleTransfer: t.fungibleTransfer.copy(),
		Contract:         t.Contract,
	}
}

func (t ERC20Transfer) From() common.Address {
	return t.FromAccount
}

func (t ERC20Transfer) To() common.Address {
	return t.ToAccount
}

func (t ERC20Transfer) Burn() bool {
	return t.ToAccount == common.Address{}
}

func (t ERC20Transfer) Mint() bool {
	return t.FromAccount == common.Address{}
}

func (t ERC20Transfer) Merge(other ITransfer) ([]ITransfer, error) {
	if other.Type() != t.Type() {
		return nil, ErrMergeNotPossible
	}
	otherTransfer := other.(ERC20Transfer)
	if otherTransfer.Contract != t.Contract {
		return nil, ErrMergeNotPossible
	}
	merged, err := t.merge(otherTransfer.fungibleTransfer)
	if err != nil {
		return nil, err
	}
	var transfers = make([]ITransfer, len(merged))
	for i, transfer := range merged {
		transfers[i] = ERC20Transfer{transfer, t.Contract}
	}
	return transfers, nil
}

func (t ERC20Transfer) Zero() bool {
	return t.Amount.Cmp(big.NewInt(0)) == 0 || t.FromAccount == t.ToAccount
}

type ERC777Transfer struct {
	fungibleTransfer
	Contract common.Address `json:"contract"`
}

func (t ERC777Transfer) MarshalJSON() ([]byte, error) {
	type Alias ERC777Transfer
	return json.Marshal(&struct {
		Alias
		Type AssetType `json:"type"`
	}{
		Alias: (Alias)(t),
		Type:  t.Type(),
	})
}

func (t ERC777Transfer) Type() AssetType {
	return ERC777Asset
}

func (t ERC777Transfer) Profits() []Profit {
	if t.Zero() {
		return nil
	}
	var profits []Profit
	if !t.Mint() {
		fromProfit := ERC777Profit{
			Account:  t.FromAccount,
			Contract: t.Contract,
			Amount:   new(big.Int).Neg(t.Amount),
		}
		profits = append(profits, fromProfit)
	}
	if !t.Burn() {
		toProfit := ERC777Profit{
			Account:  t.ToAccount,
			Contract: t.Contract,
			Amount:   t.Amount,
		}
		profits = append(profits, toProfit)
	}
	return profits
}

func (t ERC777Transfer) Copy() ITransfer {
	return ERC777Transfer{
		fungibleTransfer: t.fungibleTransfer.copy(),
		Contract:         t.Contract,
	}
}

func (t ERC777Transfer) From() common.Address {
	return t.FromAccount
}

func (t ERC777Transfer) To() common.Address {
	return t.ToAccount
}

func (t ERC777Transfer) Burn() bool {
	return t.ToAccount == common.Address{}
}

func (t ERC777Transfer) Mint() bool {
	return t.FromAccount == common.Address{}
}

func (t ERC777Transfer) Merge(other ITransfer) ([]ITransfer, error) {
	if other.Type() != t.Type() {
		return nil, ErrMergeNotPossible
	}
	otherTransfer := other.(ERC777Transfer)
	if otherTransfer.Contract != t.Contract {
		return nil, ErrMergeNotPossible
	}
	merged, err := t.merge(otherTransfer.fungibleTransfer)
	if err != nil {
		return nil, err
	}
	var transfers = make([]ITransfer, len(merged))
	for i, transfer := range merged {
		transfers[i] = ERC777Transfer{transfer, t.Contract}
	}
	return transfers, nil
}

func (t ERC777Transfer) Zero() bool {
	return t.Amount.Cmp(big.NewInt(0)) == 0 || t.FromAccount == t.ToAccount
}

type ERC721Transfer struct {
	FromAccount      common.Address        `json:"from"`
	ToAccount        common.Address        `json:"to"`
	Contract         common.Address        `json:"contract"`
	TokenID          common.Hash           `json:"tokenId"`
	ContractLocation tracers.TraceLocation `json:"location"`
}

func (n ERC721Transfer) Location() tracers.TraceLocation {
	return n.ContractLocation
}

func (n ERC721Transfer) MarshalJSON() ([]byte, error) {
	type Alias ERC721Transfer
	return json.Marshal(&struct {
		Alias
		Type AssetType `json:"type"`
	}{
		Alias: (Alias)(n),
		Type:  n.Type(),
	})
}

func (n ERC721Transfer) Type() AssetType {
	return ERC721Asset
}

func (n ERC721Transfer) Profits() []Profit {
	if n.Zero() {
		return nil
	}
	var profits []Profit
	tokenID := n.TokenID
	if !n.Mint() {
		//fromProfit := ERC721Profit{
		//	Account:  n.FromAccount,
		//	contract: n.codeAddr,
		//	TokenID:  &tokenID,
		//	Receive:  false,
		//}

		fromProfit := ERC721CombinedProfit{
			Account:       n.FromAccount,
			Contract:      n.Contract,
			receiveTokens: map[common.Hash]interface{}{},
			giveTokens: map[common.Hash]interface{}{
				tokenID: struct{}{},
			},
		}

		profits = append(profits, fromProfit)
	}
	if !n.Burn() {
		//toProfit := ERC721Profit{
		//	Account:  n.ToAccount,
		//	contract: n.codeAddr,
		//	TokenID:  &tokenID,
		//	Receive:  true,
		//}

		toProfit := ERC721CombinedProfit{
			Account:  n.ToAccount,
			Contract: n.Contract,
			receiveTokens: map[common.Hash]interface{}{
				tokenID: struct{}{},
			},
			giveTokens: map[common.Hash]interface{}{},
		}

		profits = append(profits, toProfit)
	}
	return profits
}

func (n ERC721Transfer) Copy() ITransfer {
	return ERC721Transfer{
		FromAccount:      n.FromAccount,
		ToAccount:        n.ToAccount,
		Contract:         n.Contract,
		TokenID:          n.TokenID,
		ContractLocation: n.ContractLocation,
	}
}

func (n ERC721Transfer) From() common.Address {
	return n.FromAccount
}

func (n ERC721Transfer) To() common.Address {
	return n.ToAccount
}

func (n ERC721Transfer) Burn() bool {
	return n.ToAccount == common.Address{}
}

func (n ERC721Transfer) Mint() bool {
	return n.FromAccount == common.Address{}
}

func (n ERC721Transfer) Merge(other ITransfer) ([]ITransfer, error) {
	var transfers []ITransfer
	if other.Type() != n.Type() {
		return nil, ErrMergeNotPossible
	}
	otherTransfer := other.(ERC721Transfer)
	if otherTransfer.Contract != n.Contract || otherTransfer.TokenID != n.TokenID {
		return nil, ErrMergeNotPossible
	}
	if n.FromAccount == otherTransfer.ToAccount && n.ToAccount == otherTransfer.FromAccount {
	} else if n.FromAccount == otherTransfer.ToAccount {
		transfers = append(transfers, ERC721Transfer{
			FromAccount: otherTransfer.FromAccount,
			ToAccount:   n.ToAccount,
			Contract:    n.Contract,
			TokenID:     n.TokenID,
		})
	} else if n.ToAccount == otherTransfer.FromAccount {
		transfers = append(transfers, ERC721Transfer{
			FromAccount: n.FromAccount,
			ToAccount:   otherTransfer.ToAccount,
			Contract:    n.Contract,
			TokenID:     n.TokenID,
		})
	} else {
		return nil, ErrMergeNotPossible
	}
	return transfers, nil
}

func (n ERC721Transfer) Zero() bool {
	return false
}

type ERC1155Transfer struct {
	fungibleTransfer
	Contract common.Address `json:"contract"`
	TokenID  common.Hash    `json:"tokenId"`
}

func (t ERC1155Transfer) MarshalJSON() ([]byte, error) {
	type Alias ERC1155Transfer
	return json.Marshal(&struct {
		Alias
		Type AssetType `json:"type"`
	}{
		Alias: (Alias)(t),
		Type:  t.Type(),
	})
}

func (t ERC1155Transfer) Type() AssetType {
	return ERC1155Asset
}

func (t ERC1155Transfer) Profits() []Profit {
	if t.Zero() {
		return nil
	}
	var profits []Profit
	if !t.Mint() {
		fromProfit := ERC1155Profit{
			Account:  t.FromAccount,
			Contract: t.Contract,
			TokenID:  t.TokenID,
			Amount:   new(big.Int).Neg(t.Amount),
		}
		profits = append(profits, fromProfit)
	}
	if !t.Burn() {
		toProfit := ERC1155Profit{
			Account:  t.ToAccount,
			Contract: t.Contract,
			TokenID:  t.TokenID,
			Amount:   t.Amount,
		}
		profits = append(profits, toProfit)
	}
	return profits
}

func (t ERC1155Transfer) Copy() ITransfer {
	return ERC1155Transfer{
		fungibleTransfer: t.fungibleTransfer.copy(),
		Contract:         t.Contract,
		TokenID:          t.TokenID,
	}
}

func (t ERC1155Transfer) From() common.Address {
	return t.FromAccount
}

func (t ERC1155Transfer) To() common.Address {
	return t.ToAccount
}

func (t ERC1155Transfer) Burn() bool {
	return t.ToAccount == common.Address{}
}

func (t ERC1155Transfer) Mint() bool {
	return t.FromAccount == common.Address{}
}

func (t ERC1155Transfer) Merge(other ITransfer) ([]ITransfer, error) {
	if other.Type() != t.Type() {
		return nil, ErrMergeNotPossible
	}
	otherTransfer := other.(ERC1155Transfer)
	if otherTransfer.Contract != t.Contract || otherTransfer.TokenID != t.TokenID {
		return nil, ErrMergeNotPossible
	}
	merged, err := t.merge(otherTransfer.fungibleTransfer)
	if err != nil {
		return nil, err
	}
	var transfers = make([]ITransfer, len(merged))
	for i, transfer := range merged {
		transfers[i] = ERC1155Transfer{transfer, t.Contract, t.TokenID}
	}
	return transfers, nil
}

func (t ERC1155Transfer) Zero() bool {
	return t.Amount.Cmp(big.NewInt(0)) == 0 || t.ToAccount == t.FromAccount
}

func LogToTransfers(log *types.Log, location tracers.TraceLocation) ([]ITransfer, error) {
	if len(log.Topics) == 0 {
		return nil, ErrNotTransfer
	}
	switch log.Topics[0] {
	case ERC20TransferTopic, ERC721TransferTopic: // ERC20 and ERC721 have the same topic
		args, err := ParseEvent(ERC20ABI.Events["Transfer"], log)
		if err == ErrNotTransfer {
			return nil, ErrNotTransfer
		} else if err != nil {
			// try ERC721 transfer
			args, err := ParseEvent(ERC721ABI.Events["Transfer"], log)
			if err != nil {
				return nil, ErrNotTransfer
			}
			tokenID := common.BigToHash(args[2].(*big.Int))
			transfer := ERC721Transfer{
				FromAccount:      args[0].(common.Address),
				ToAccount:        args[1].(common.Address),
				Contract:         log.Address,
				TokenID:          tokenID,
				ContractLocation: location,
			}
			return []ITransfer{transfer}, nil
		} else {
			transfer := ERC20Transfer{
				fungibleTransfer: fungibleTransfer{
					FromAccount:      args[0].(common.Address),
					ToAccount:        args[1].(common.Address),
					Amount:           args[2].(*big.Int),
					ContractLocation: location,
				},
				Contract: log.Address,
			}
			return []ITransfer{transfer}, nil
		}

	case ERC777SentTopic:
		args, err := ParseEvent(ERC777ABI.Events["Sent"], log)
		if err != nil {
			return nil, ErrNotTransfer
		}
		// collect transfer
		transfer := ERC777Transfer{
			fungibleTransfer: fungibleTransfer{
				FromAccount:      args[1].(common.Address),
				ToAccount:        args[2].(common.Address),
				Amount:           args[3].(*big.Int),
				ContractLocation: location,
			},
			Contract: log.Address,
		}
		return []ITransfer{transfer}, nil

	case ERC777MintedTopic:
		args, err := ParseEvent(ERC777ABI.Events["Minted"], log)
		if err != nil {
			return nil, ErrNotTransfer
		}
		// collect transfer
		transfer := ERC777Transfer{
			fungibleTransfer: fungibleTransfer{
				FromAccount:      common.Address{},
				ToAccount:        args[1].(common.Address),
				Amount:           args[2].(*big.Int),
				ContractLocation: location,
			},
			Contract: log.Address,
		}
		return []ITransfer{transfer}, nil

	case ERC777BurnedTopic:
		args, err := ParseEvent(ERC777ABI.Events["Burned"], log)
		if err != nil {
			return nil, ErrNotTransfer
		}
		// collect transfer
		transfer := ERC777Transfer{
			fungibleTransfer: fungibleTransfer{
				FromAccount:      args[1].(common.Address),
				ToAccount:        common.Address{},
				Amount:           args[2].(*big.Int),
				ContractLocation: location,
			},
			Contract: log.Address,
		}
		return []ITransfer{transfer}, nil

	case ERC1155TransferSingleTopic:
		args, err := ParseEvent(ERC1155ABI.Events["TransferSingle"], log)
		if err != nil {
			return nil, ErrNotTransfer
		}
		tokenID := common.BigToHash(args[3].(*big.Int))
		transfer := ERC1155Transfer{
			fungibleTransfer: fungibleTransfer{
				FromAccount:      args[1].(common.Address),
				ToAccount:        args[2].(common.Address),
				Amount:           args[4].(*big.Int),
				ContractLocation: location,
			},
			Contract: log.Address,
			TokenID:  tokenID,
		}
		return []ITransfer{transfer}, nil

	case ERC1155TransferBatchTopic:
		args, err := ParseEvent(ERC1155ABI.Events["TransferBatch"], log)
		if err != nil {
			return nil, ErrNotTransfer
		}
		var transfers []ITransfer
		ids := args[3].([]*big.Int)
		values := args[4].([]*big.Int)
		for i, id := range ids {
			tokenID := common.BigToHash(id)
			transfers = append(transfers, ERC1155Transfer{
				fungibleTransfer: fungibleTransfer{
					FromAccount:      args[1].(common.Address),
					ToAccount:        args[2].(common.Address),
					Amount:           values[i],
					ContractLocation: location,
				},
				Contract: log.Address,
				TokenID:  tokenID,
			})
		}
		return transfers, nil

	case WETH9DepositTopic:
		if log.Address != WETH9Address {
			return nil, ErrNotTransfer
		}
		args, err := ParseEvent(WETH9ABI.Events["Deposit"], log)
		if err != nil {
			return nil, ErrNotTransfer
		}
		transfer := ERC20Transfer{
			fungibleTransfer: fungibleTransfer{
				FromAccount:      common.Address{},
				ToAccount:        args[0].(common.Address),
				Amount:           args[1].(*big.Int),
				ContractLocation: location,
			},
			Contract: log.Address,
		}
		return []ITransfer{transfer}, nil

	case WETH9WithdrawalTopic:
		if log.Address != WETH9Address {
			return nil, ErrNotTransfer
		}
		args, err := ParseEvent(WETH9ABI.Events["Withdrawal"], log)
		if err != nil {
			return nil, ErrNotTransfer
		}
		transfer := ERC20Transfer{
			fungibleTransfer: fungibleTransfer{
				FromAccount:      args[0].(common.Address),
				ToAccount:        common.Address{},
				Amount:           args[1].(*big.Int),
				ContractLocation: location,
			},
			Contract: log.Address,
		}
		return []ITransfer{transfer}, nil

	default:
		return nil, ErrNotTransfer
	}
}
