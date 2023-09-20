package dataset

import (
	"context"
	"fmt"

	"github.com/Troublor/erebus-redgiant/chain"
	"github.com/samber/lo"

	"github.com/Troublor/erebus-redgiant/analysis/summary"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Attack struct {
	Attacker, Victim                                              common.Address
	AttackTxRecord, VictimTxRecord, ProfitTxRecord                *TxRecord
	AttackTxAsIfSummary, VictimTxAsIfSummary, ProfitTxAsIfSummary *summary.CallSummary

	//OutOfGas       bool
	//MismatchOracle bool
	Analysis []*AttackAnalysis

	// cache
	attackerProfits, victimProfits, attackerAsIfProfits, victimAsIfProfits summary.Profits
	hash                                                                   *common.Hash
	attackBSON                                                             *AttackBSON
}

func ComputeAttackHash(attackTx, victimTx common.Hash, profitTx *common.Hash) common.Hash {
	if profitTx != nil {
		return crypto.Keccak256Hash(
			attackTx.Bytes(),
			victimTx.Bytes(),
			profitTx.Bytes(),
		)
	}
	return crypto.Keccak256Hash(
		attackTx.Bytes(),
		victimTx.Bytes(),
	)
}

func (a *Attack) Hash() common.Hash {
	if a.hash == nil {
		if a.ProfitTxRecord != nil {
			hash := ComputeAttackHash(
				a.AttackTxRecord.Tx.Hash(),
				a.VictimTxRecord.Tx.Hash(),
				lo.ToPtr(a.ProfitTxRecord.Tx.Hash()),
			)
			a.hash = &hash
		} else {
			hash := ComputeAttackHash(
				a.AttackTxRecord.Tx.Hash(),
				a.VictimTxRecord.Tx.Hash(),
				nil,
			)
			a.hash = &hash
		}
	}
	return *a.hash
}

func (ab *AttackBSON) AsAttack(
	ctx context.Context,
	chainReader chain.BlockchainReader,
) (*Attack, error) {
	history := NewTxHistory(chainReader, nil)
	return ab.AsAttackWithTxHistory(ctx, chainReader, history)
}

func (ab *AttackBSON) AsAttackWithTxHistory(
	ctx context.Context,
	chainReader chain.BlockchainReader,
	history *TxHistory,
) (*Attack, error) {
	if ab.attack == nil {
		attackTx := common.HexToHash(ab.AttackTx)
		victimTx := common.HexToHash(ab.VictimTx)
		var profitTx *common.Hash
		if ab.ProfitTx != nil {
			t := common.HexToHash(*ab.ProfitTx)
			profitTx = &t
		}
		var err error
		ab.attack, err = ConstructAttackWithTxHistory(
			ctx,
			chainReader,
			attackTx,
			victimTx,
			profitTx,
			history,
		)
		if err != nil {
			return nil, err
		}
	}
	return ab.attack, nil
}

func ConstructAttack(
	ctx context.Context,
	chainReader chain.BlockchainReader,
	attackTx, victimTx common.Hash, profitTx *common.Hash,
) (*Attack, error) {
	history := NewTxHistory(chainReader, nil)
	return ConstructAttackWithTxHistory(ctx, chainReader, attackTx, victimTx, profitTx, history)
}

// ConstructAttack creates an Attack if the given attack/victim/profit tx pair indeed comprises an attack.
// ConstructAttack will call AttackSearcher without parallism and try to find the attack.
func ConstructAttackWithTxHistory(
	ctx context.Context,
	chainReader chain.BlockchainReader,
	attackTx, victimTx common.Hash, profitTx *common.Hash,
	history *TxHistory,
) (*Attack, error) {
	attackReceipt, err := chainReader.TransactionReceipt(ctx, attackTx)
	if err != nil {
		return nil, fmt.Errorf("failed to get attack receipt: %v", err.Error())
	}
	from := attackReceipt.BlockNumber
	victimReceipt, err := chainReader.TransactionReceipt(ctx, victimTx)
	if err != nil {
		return nil, fmt.Errorf("failed to get victim receipt: %v", err.Error())
	}
	to := victimReceipt.BlockNumber
	if profitTx != nil {
		profitReceipt, err := chainReader.TransactionReceipt(ctx, *profitTx)
		if err != nil {
			return nil, fmt.Errorf("failed to get profit receipt: %v", err.Error())
		}
		to = profitReceipt.BlockNumber
	}

	window := int(to.Uint64() - from.Uint64() + 1)

	searcher := NewAttackSearcher(chainReader, history)
	var attack *Attack
	searcher.SetAttackHandler(func(session *TxHistorySession, a *Attack) {
		attack = a
	})
	searchWindow := searcher.OpenSearchWindow(ctx, from.Uint64(), window)
	defer searchWindow.Close()

	searchWindow.SetFocus(attackTx, victimTx, profitTx)
	searchWindow.SetSearchPivot(from.Uint64(), attackReceipt.TransactionIndex)

	searchWindow.Search(ctx)

	if attack != nil {
		_, err := attack.Analyze(chainReader, searchWindow.TxHistorySession())
		if err != nil {
			return nil, fmt.Errorf("failed to analyze attack: %w", err)
		}
	}

	return attack, nil
}
