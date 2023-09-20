package summary

import (
	_ "embed"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var ErrNotTransfer = errors.New("not profit")
var ErrNotComparable = errors.New("not comparable")

type Profit interface {
	// Type returns the type of the profit
	Type() AssetType

	// Beneficiary returns the account address that get this profit.
	Beneficiary() common.Address

	// Copy returns a copy of the profit.
	Copy() Profit

	// Merge merges the current profit with the given profit and returns the merged profit.
	// The merged profit should be separated from the original profit (copy if necessary).
	// If the two profits are not comparable (Cmp returns ErrNotComparable), they are not possible to merge.
	// If merge operation is not possible, an ErrMergeNotPossible error is returned.
	Merge(other Profit) (Profit, error)

	// Cmp compares the current profit with the given profit and returns the comparison result.
	// If the two Profit value is not comparable, an ErrNotComparable error is returned.
	Cmp(other Profit) (int, error)

	// Zero returns true if the profit is effectively zero (no profit).
	Zero() bool

	// Positive returns true if the Beneficiary gains profit (this profit is positive).
	Positive() bool
}

type ZeroProfit struct{}

func (z ZeroProfit) Type() AssetType {
	return ZeroAsset
}

func (z ZeroProfit) Beneficiary() common.Address {
	return common.Address{}
}

func (z ZeroProfit) Copy() Profit {
	return ZeroProfit{}
}

func (z ZeroProfit) Merge(other Profit) (Profit, error) {
	return other.Copy(), nil
}

func (z ZeroProfit) Cmp(other Profit) (int, error) {
	if other.Positive() {
		return -1, nil
	} else if other.Zero() {
		return 0, nil
	} else {
		return 1, nil
	}
}

func (z ZeroProfit) Zero() bool {
	return true
}

func (z ZeroProfit) Positive() bool {
	return false
}

// Profits represents a slice of Profit.
// Note that it is essentially a slice, so it can be used as a slice.
type Profits []Profit

// Compact merges all possible Profit in the Profits and returns the resulting Profits.
// Zero Profit will be removed in the result.
func (ps Profits) Compact() Profits {
	r := make(Profits, 0, len(ps))
	r = r.Add(ps...)
	rr := make(Profits, 0, len(r))
	for _, p := range ps {
		if !p.Zero() {
			rr = append(rr, p)
		}
	}
	return rr
}

// Add adds new profits to current Profits.
// New profits are merged with existing ones if possible.
// But existing profits are not merged with each other. Use Compact to merge existing ones.
// If any existing profit becomes Zero after merging new ones, it will still remain in the result.
// No Zero profit will be removed in the result.
// Also use Compact to remove Zero profit.
func (ps Profits) Add(elems ...Profit) Profits {
	slice := ps
elemsLoop:
	for _, elem := range elems {
		// copy the profit to ensure that the profit is not modified
		elem = elem.Copy()

		for i, p := range slice {
			if merged, err := p.Merge(elem); err == nil {
				// merge was successful, replace the profit
				slice[i] = merged
				// continue with the next profit to add
				continue elemsLoop
			} else if err != ErrMergeNotPossible {
				panic(fmt.Errorf("unexpected error: %w", err))
			}
		}
		// no merge was possible for this profit, append the profit
		slice = append(slice, elem)
	}

	return slice
}

// Cmp compares two set of Profits.
// It compares A and B in two phases.
// Phase 1:
//
//	compact profits in A.
//	compact profits in B.
//	such that each profit in one set has at most one comparable profit in the other set.
//
// Phase 2:
//
//	For each profit P in each set, try find a comparable P' in the other set.
//		if P' does not exist, use ZeroProfit.
//		Call P.Cmp(P')
//	If all such comparison returns the same, return the comparison result.
//	Otherwise, return ErrNotComparable error.
func (ps Profits) Cmp(other Profits) (int, error) {
	this := ps.Compact()
	other = other.Compact()
	if len(this) == 0 {
		this = this.Add(ZeroProfit{})
	}
	if len(other) == 0 {
		other = other.Add(ZeroProfit{})
	}

	innerCmp := func(t, o Profits) (int, error) {
		var err error
		var tmpResult = 0
		for _, p := range t {
			var cmp int
			for _, p2 := range o {
				cmp, err = p.Cmp(p2)
				if err != nil {
					continue
				}
				goto update
			}
			// no comparable profit found in other set, use ZeroProfit
			cmp = -1
			if p.Positive() {
				cmp = 1
			} else if p.Zero() {
				cmp = 0
			}
		update:
			if tmpResult == 0 {
				tmpResult = cmp
			} else if cmp != 0 && tmpResult != cmp {
				return 0, ErrNotComparable
			}
		}
		return tmpResult, nil
	}

	result, err := innerCmp(this, other)
	if err != nil {
		return 0, err
	}
	reverseResult, err := innerCmp(other, this)
	if err != nil {
		return 0, err
	}

	if result == -reverseResult {
		return result, nil
	} else {
		return 0, ErrNotComparable
	}
}

// SomeMore uses Cmp to compare the current Profits with the given Profits.
// SomeMore returns true if some profit in current Profits is more (Cmp == 1) than the given Profits.
func (ps Profits) SomeMore(other Profits) (bool, Profit, Profit) {
outerLoop:
	for _, thisP := range ps {
		for _, otherP := range other {
			cmp, err := thisP.Cmp(otherP)
			if err == nil { // comparable
				if cmp > 0 {
					return true, thisP, otherP
				} else {
					continue outerLoop
				}
			}
		}
		// no profits in the given Profits is comparable
		if !thisP.Zero() {
			switch thisP.Positive() {
			case true:
				return true, thisP, nil
			case false:
			}
		}
	}
	// at this point, all profits in current Profits are comparable
	// with some profit in the given Profits, and they are equal.
	// This means the given Profits subsumes current Profits.
outerLoop2:
	for _, otherP := range other {
		for _, thisP := range ps {
			cmp, err := thisP.Cmp(otherP)
			if err == nil { // comparable
				if cmp > 0 {
					return true, thisP, otherP
				} else {
					continue outerLoop2
				}
			}
		}
		// no profits in the given Profits is comparable
		if !otherP.Zero() {
			switch otherP.Positive() {
			case false:
				return true, nil, otherP
			case true:
			}
		}
	}
	return false, nil, nil
}

func (ps Profits) GroupByType() map[AssetType]Profits {
	r := make(map[AssetType]Profits)
	for _, t := range AllAssetTypes {
		r[t] = Profits{}
	}
	for _, p := range ps {
		r[p.Type()] = r[p.Type()].Add(p)
	}
	return r
}

func (ps Profits) ProfitsOf(account common.Address) Profits {
	r := make(Profits, 0, len(ps))
	for _, p := range ps {
		if p.Beneficiary() == account {
			r = r.Add(p)
		}
	}
	return r
}

func (ps Profits) EtherProfitOf(account common.Address) EtherProfit {
	p := EtherProfit{
		Account: account,
		Amount:  big.NewInt(0),
	}
	for _, profit := range ps {
		merged, err := p.Merge(profit)
		if err == nil {
			p = merged.(EtherProfit)
		}
	}
	return p
}

func (ps Profits) ERC20ProfitOf(contract common.Address, account common.Address) ERC20Profit {
	p := ERC20Profit{
		Account:  account,
		Contract: contract,
		Amount:   big.NewInt(0),
	}
	for _, profit := range ps {
		merged, err := p.Merge(profit)
		if err == nil {
			p = merged.(ERC20Profit)
		}
	}
	return p
}

func (ps Profits) AllERC20ProfitsOf(account common.Address) map[common.Address]ERC20Profit {
	sum := map[common.Address]ERC20Profit{}
	for _, profit := range ps {
		if p, ok := profit.(ERC20Profit); ok {
			if p.Account == account {
				if sumProfit, exist := sum[p.Contract]; exist {
					merged, err := sumProfit.Merge(p)
					if err == nil {
						sum[p.Contract] = merged.(ERC20Profit)
					}
				} else {
					sum[p.Contract] = p
				}
			}
		}
	}
	for c, profit := range sum {
		if profit.Zero() {
			delete(sum, c)
		}
	}
	return sum
}

func (ps Profits) ERC721ProfitsOf(contract common.Address, account common.Address) Profits {
	profits := make(Profits, 0)
	for _, profit := range ps {
		if p, ok := profit.(ERC721Profit); ok {
			if p.Account == account && p.Contract == contract {
				profits = profits.Add(p)
			}
		}
	}
	return profits.Compact()
}

func (ps Profits) AllERC721ProfitsOf(account common.Address) map[common.Address]Profits {
	sum := map[common.Address]Profits{}
	for _, profit := range ps {
		if p, ok := profit.(ERC721Profit); ok {
			if p.Account == account {
				if sumProfits, exist := sum[p.Contract]; exist {
					sumProfits = sumProfits.Add(p)
					sum[p.Contract] = sumProfits
				} else {
					sum[p.Contract] = Profits{p}
				}
			}
		}
	}
	for c, profits := range sum {
		compact := profits.Compact()
		if len(compact) == 0 {
			delete(sum, c)
		} else {
			sum[c] = compact
		}
	}
	return sum
}

func (ps Profits) ERC777ProfitOf(contract, account common.Address) ERC777Profit {
	p := ERC777Profit{
		Account:  account,
		Contract: contract,
		Amount:   big.NewInt(0),
	}
	for _, profit := range ps {
		merged, err := p.Merge(profit)
		if err == nil {
			p = merged.(ERC777Profit)
		}
	}
	return p
}

func (ps Profits) AllERC777ProfitsOf(account common.Address) map[common.Address]ERC777Profit {
	sum := map[common.Address]ERC777Profit{}
	for _, profit := range ps {
		if p, ok := profit.(ERC777Profit); ok {
			if p.Account == account {
				if sumProfit, exist := sum[p.Contract]; exist {
					merged, err := sumProfit.Merge(p)
					if err == nil {
						sum[p.Contract] = merged.(ERC777Profit)
					}
				} else {
					sum[p.Contract] = p
				}
			}
		}
	}
	for c, profit := range sum {
		if profit.Zero() {
			delete(sum, c)
		}
	}
	return sum
}

func (ps Profits) ERC1155ProfitsOf(contract, account common.Address) map[common.Hash]ERC1155Profit {
	sum := map[common.Hash]ERC1155Profit{}
	for _, profit := range ps {
		if p, ok := profit.(ERC1155Profit); ok {
			if p.Account == account && p.Contract == contract {
				if sumProfit, exist := sum[p.TokenID]; exist {
					merged, err := sumProfit.Merge(p)
					if err == nil {
						sum[p.TokenID] = merged.(ERC1155Profit)
					}
				} else {
					sum[p.TokenID] = p
				}
			}
		}
	}
	for c, profit := range sum {
		if profit.Zero() {
			delete(sum, c)
		}
	}
	return sum
}

func (ps Profits) AllERC1155ProfitsOf(
	account common.Address,
) map[common.Address]map[common.Hash]ERC1155Profit {
	sum := map[common.Address]map[common.Hash]ERC1155Profit{}
	for _, profit := range ps {
		if p, ok := profit.(ERC1155Profit); ok {
			if p.Account == account {
				if _, exist := sum[p.Contract]; !exist {
					sum[p.Contract] = make(map[common.Hash]ERC1155Profit)
				}
				sumProfit := sum[p.Contract]
				if tokenProfit, contain := sumProfit[p.TokenID]; contain {
					merged, err := tokenProfit.Merge(p)
					if err == nil {
						sumProfit[p.TokenID] = merged.(ERC1155Profit)
					}
				} else {
					sumProfit[p.TokenID] = p
				}
			}
		}
	}
	for c, profit := range sum {
		for tokenID, p := range profit {
			if p.Zero() {
				delete(profit, tokenID)
			}
		}
		if len(profit) == 0 {
			delete(sum, c)
		}
	}
	return sum
}

type EtherProfit struct {
	Account common.Address
	Amount  *big.Int
}

func (e EtherProfit) Type() AssetType {
	return EtherAsset
}

func (e EtherProfit) Beneficiary() common.Address {
	return e.Account
}

func (e EtherProfit) Copy() Profit {
	return EtherProfit{
		Account: e.Account,
		Amount:  new(big.Int).Set(e.Amount),
	}
}

func (e EtherProfit) Merge(other Profit) (Profit, error) {
	if _, err := e.Cmp(other); err != nil {
		return nil, ErrMergeNotPossible
	}
	if ep, ok := other.(EtherProfit); ok {
		if e.Account == ep.Account {
			value := new(big.Int).Add(e.Amount, ep.Amount)
			r := EtherProfit{
				Account: e.Account,
				Amount:  value,
			}
			return r, nil
		}
	}
	return nil, ErrMergeNotPossible
}

func (e EtherProfit) Cmp(other Profit) (int, error) {
	if e.Type() == other.Type() && e.Beneficiary() == other.Beneficiary() {
		return e.Amount.Cmp(other.(EtherProfit).Amount), nil
	}
	return 0, ErrNotComparable
}

func (e EtherProfit) Zero() bool {
	return e.Amount.Cmp(big.NewInt(0)) == 0
}

func (e EtherProfit) Positive() bool {
	return e.Amount.Cmp(big.NewInt(0)) > 0
}

type ERC20Profit struct {
	Account  common.Address
	Contract common.Address
	Amount   *big.Int
}

func (t ERC20Profit) Type() AssetType {
	return ERC20Asset
}

func (t ERC20Profit) Beneficiary() common.Address {
	return t.Account
}

func (t ERC20Profit) Copy() Profit {
	return ERC20Profit{
		Account:  t.Account,
		Contract: t.Contract,
		Amount:   new(big.Int).Set(t.Amount),
	}
}

func (t ERC20Profit) Merge(other Profit) (Profit, error) {
	if _, err := t.Cmp(other); err != nil {
		return nil, ErrMergeNotPossible
	}
	if tp, ok := other.(ERC20Profit); ok {
		if t.Account == tp.Account && t.Contract == tp.Contract {
			value := new(big.Int).Add(t.Amount, tp.Amount)
			r := ERC20Profit{
				Account:  t.Account,
				Contract: t.Contract,
				Amount:   value,
			}
			return r, nil
		}
	}
	return nil, ErrMergeNotPossible
}

func (t ERC20Profit) Cmp(other Profit) (int, error) {
	if t.Type() == other.Type() &&
		t.Beneficiary() == other.Beneficiary() &&
		t.Contract == other.(ERC20Profit).Contract {
		return t.Amount.Cmp(other.(ERC20Profit).Amount), nil
	}
	return 0, ErrNotComparable
}

func (t ERC20Profit) Zero() bool {
	return t.Amount.Cmp(big.NewInt(0)) == 0
}

func (t ERC20Profit) Positive() bool {
	return t.Amount.Cmp(big.NewInt(0)) > 0
}

type ERC721CombinedProfit struct {
	Account       common.Address
	Contract      common.Address
	receiveTokens map[common.Hash]interface{}
	giveTokens    map[common.Hash]interface{}
}

func (n ERC721CombinedProfit) Type() AssetType {
	return ERC721Asset
}

func (n ERC721CombinedProfit) Beneficiary() common.Address {
	return n.Account
}

func (n ERC721CombinedProfit) Copy() Profit {
	cpRT := make(map[common.Hash]interface{})
	for hash, b := range n.receiveTokens {
		cpRT[hash] = b
	}
	cpGT := make(map[common.Hash]interface{})
	for hash, b := range n.giveTokens {
		cpGT[hash] = b
	}
	return ERC721CombinedProfit{
		Account:       n.Account,
		Contract:      n.Contract,
		receiveTokens: cpRT,
		giveTokens:    cpGT,
	}
}

func (n ERC721CombinedProfit) Merge(other Profit) (Profit, error) {
	if _, err := n.Cmp(other); err != nil {
		return nil, ErrMergeNotPossible
	}
	merged := n.Copy().(ERC721CombinedProfit)

	for r := range other.(ERC721CombinedProfit).receiveTokens {
		if _, gave := merged.giveTokens[r]; gave {
			delete(merged.giveTokens, r)
		} else {
			if _, received := merged.receiveTokens[r]; received {
				// Gosh, some contracts really implement in this way.
				// panic("account cannot receive the same ERC721 token twice")
			} else {
				merged.receiveTokens[r] = struct{}{}
			}
		}
	}
	for g := range other.(ERC721CombinedProfit).giveTokens {
		if _, received := merged.receiveTokens[g]; received {
			delete(merged.receiveTokens, g)
		} else {
			if _, gave := merged.giveTokens[g]; gave {
				// Gosh, some contracts really implement in this way.
				// panic("account cannot give the same ERC721 token twice")
			} else {
				merged.giveTokens[g] = struct{}{}
			}
		}
	}
	return merged, nil
}

func (n ERC721CombinedProfit) Cmp(other Profit) (int, error) {
	p := func(n ERC721CombinedProfit) int {
		return len(n.receiveTokens) - len(n.giveTokens)
	}
	if n.Type() == other.Type() &&
		n.Beneficiary() == other.Beneficiary() &&
		n.Contract == other.(ERC721CombinedProfit).Contract {
		if p(n) > p(other.(ERC721CombinedProfit)) {
			return 1, nil
		} else if p(n) < p(other.(ERC721CombinedProfit)) {
			return -1, nil
		} else {
			return 0, nil
		}
	}
	return 0, ErrNotComparable
}

func (n ERC721CombinedProfit) Zero() bool {
	return len(n.receiveTokens)-len(n.giveTokens) == 0
}

func (n ERC721CombinedProfit) Positive() bool {
	return len(n.receiveTokens)-len(n.giveTokens) > 0
}

// ERC721Profit is a profit for an ERC721 token.
// Deprecated: use ERC721CombinedProfit instead.
type ERC721Profit struct {
	Account  common.Address
	Contract common.Address
	TokenID  *common.Hash
	Receive  bool
}

func (n ERC721Profit) Type() AssetType {
	return ERC721Asset
}

func (n ERC721Profit) Beneficiary() common.Address {
	return n.Account
}

func (n ERC721Profit) Copy() Profit {
	var tokenID *common.Hash
	if n.TokenID != nil {
		tokenID = new(common.Hash)
		*tokenID = *n.TokenID
	}
	return ERC721Profit{
		Account:  n.Account,
		Contract: n.Contract,
		TokenID:  tokenID,
		Receive:  n.Receive,
	}
}

func (n ERC721Profit) Merge(other Profit) (Profit, error) {
	if _, err := n.Cmp(other); err != nil {
		return nil, ErrMergeNotPossible
	}
	if np, ok := other.(ERC721Profit); ok {
		if n.Account == np.Account && n.Contract == np.Contract {
			if n.Zero() {
				return np.Copy(), nil
			} else if np.Zero() {
				return n.Copy(), nil
			} else if *n.TokenID == *np.TokenID {
				if n.Receive != np.Receive {
					// one receive and one not receive, result in zero profit
					r := ERC721Profit{
						Account:  n.Account,
						Contract: n.Contract,
						TokenID:  nil,
					}
					return r, nil
				} else {
					panic(errors.New("impossible: merging two positive/negative ERC721 profits"))
				}
			}
		}
	}
	return nil, ErrMergeNotPossible
}

func (n ERC721Profit) Cmp(other Profit) (int, error) {
	if n.TokenID == nil || other.(ERC721Profit).TokenID == nil {
		return 0, ErrNotComparable
	}
	if n.Type() == other.Type() &&
		n.Beneficiary() == other.Beneficiary() &&
		n.Contract == other.(ERC721Profit).Contract &&
		*n.TokenID == *other.(ERC721Profit).TokenID {
		switch other.(ERC721Profit).Receive {
		case true:
			switch n.Receive {
			case true:
				return 0, nil
			case false:
				return -1, nil
			}
		case false:
			switch n.Receive {
			case true:
				return 1, nil
			case false:
				return 0, nil
			}
		}
	}
	return 0, ErrNotComparable
}

func (n ERC721Profit) Zero() bool {
	return n.TokenID == nil
}

func (n ERC721Profit) Positive() bool {
	return n.TokenID != nil && n.Receive
}

type ERC777Profit struct {
	Account  common.Address
	Contract common.Address
	Amount   *big.Int
}

func (t ERC777Profit) Type() AssetType {
	return ERC777Asset
}

func (t ERC777Profit) Beneficiary() common.Address {
	return t.Account
}

func (t ERC777Profit) Copy() Profit {
	return ERC777Profit{
		Account:  t.Account,
		Contract: t.Contract,
		Amount:   new(big.Int).Set(t.Amount),
	}
}

func (t ERC777Profit) Merge(other Profit) (Profit, error) {
	if _, err := t.Cmp(other); err != nil {
		return nil, ErrMergeNotPossible
	}
	if tp, ok := other.(ERC777Profit); ok {
		if t.Account == tp.Account && t.Contract == tp.Contract {
			value := new(big.Int).Add(t.Amount, tp.Amount)
			r := ERC777Profit{
				Account:  t.Account,
				Contract: t.Contract,
				Amount:   value,
			}
			return r, nil
		}
	}
	return nil, ErrMergeNotPossible
}

func (t ERC777Profit) Cmp(other Profit) (int, error) {
	if t.Type() == other.Type() &&
		t.Beneficiary() == other.Beneficiary() &&
		t.Contract == other.(ERC777Profit).Contract {
		return t.Amount.Cmp(other.(ERC777Profit).Amount), nil
	}
	return 0, ErrNotComparable
}

func (t ERC777Profit) Zero() bool {
	return t.Amount.Cmp(big.NewInt(0)) == 0
}

func (t ERC777Profit) Positive() bool {
	return t.Amount.Cmp(big.NewInt(0)) > 0
}

type ERC1155Profit struct {
	Account  common.Address
	Contract common.Address
	TokenID  common.Hash
	Amount   *big.Int
}

func (t ERC1155Profit) Type() AssetType {
	return ERC1155Asset
}

func (t ERC1155Profit) Beneficiary() common.Address {
	return t.Account
}

func (t ERC1155Profit) Copy() Profit {
	return ERC1155Profit{
		Account:  t.Account,
		Contract: t.Contract,
		TokenID:  t.TokenID,
		Amount:   new(big.Int).Set(t.Amount),
	}
}

func (t ERC1155Profit) Merge(other Profit) (Profit, error) {
	if _, err := t.Cmp(other); err != nil {
		return nil, ErrMergeNotPossible
	}
	if tp, ok := other.(ERC1155Profit); ok {
		if t.Account == tp.Account && t.Contract == tp.Contract && t.TokenID == tp.TokenID {
			value := new(big.Int).Add(t.Amount, tp.Amount)
			r := ERC1155Profit{
				Account:  t.Account,
				Contract: t.Contract,
				TokenID:  t.TokenID,
				Amount:   value,
			}
			return r, nil
		}
	}
	return nil, ErrMergeNotPossible
}

func (t ERC1155Profit) Cmp(other Profit) (int, error) {
	if t.Type() == other.Type() &&
		t.Beneficiary() == other.Beneficiary() &&
		t.Contract == other.(ERC1155Profit).Contract &&
		t.TokenID == other.(ERC1155Profit).TokenID {
		return t.Amount.Cmp(other.(ERC1155Profit).Amount), nil
	}
	return 0, ErrNotComparable
}

func (t ERC1155Profit) Zero() bool {
	return t.Amount.Cmp(big.NewInt(0)) == 0
}

func (t ERC1155Profit) Positive() bool {
	return t.Amount.Cmp(big.NewInt(0)) > 0
}
