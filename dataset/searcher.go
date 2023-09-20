package dataset

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Troublor/erebus-redgiant/chain"
	"github.com/Troublor/erebus-redgiant/helpers"

	"github.com/Troublor/erebus-redgiant/analysis/summary"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"

	engine "github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

func AttackSearchVMConfig() engine.VMConfig {
	config := helpers.VMConfigOnMainnet()
	config.CapGasToBlockLimit = true
	config.RegulateBaseFee = true
	config.NoBaseFee = true
	return config
}

type AttackHandler func(*TxHistorySession, *Attack)

type AttackSearcher struct {
	pool        *ants.Pool
	stateReader chain.BlockchainReader
	txHistory   *TxHistory
	handler     AttackHandler
	oracleHook  func(
		session *TxHistorySession,
		attackers, victims []common.Address,
		ar, vr, pr *TxRecord,
		attackTxAsIfSummary, victimTxAsIfSummary, profitTxAsIfSummary *summary.TxSummary,
	)

	latestBlock     atomic.Value // uint64
	prefetchedBlock uint64
}

func NewAttackSearcher(stateReader chain.BlockchainReader, txHistory *TxHistory) *AttackSearcher {
	return &AttackSearcher{
		stateReader: stateReader,
		txHistory:   txHistory,
	}
}

func (s *AttackSearcher) isWhitelistError(err error) bool {
	whitelistErrors := []error{
		// FIXME temporary workaround,
		// for insufficient fund for gas (when there is one previous tx and send ether to victim).
		// a better solution is to include value transfer transactions in the dependencies
		core.ErrInsufficientFunds,
		core.ErrInsufficientFundsForTransfer,
		// FIXME temporary workaround, for insufficient intrinsic gas error
		core.ErrIntrinsicGas,
	}

	for _, e := range whitelistErrors {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

func (s *AttackSearcher) PrefetchBlocks(ctx context.Context, pool *ants.Pool, num int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			lb, ok := s.latestBlock.Load().(uint64)
			if !ok {
				continue
			}
			if lb+uint64(num) > s.prefetchedBlock {
				s.txHistory.AcquireBlocks(ctx, pool, s.genBlockRange(lb, num)...)
				s.prefetchedBlock = lb + uint64(num)
			}
			time.Sleep(time.Second)
		}
	}
}

func (s *AttackSearcher) genBlockRange(from uint64, size int) []uint64 {
	r := make([]uint64, size)
	for i := 0; i < size; i++ {
		r[i] = from + uint64(i)
	}
	return r
}

func (s *AttackSearcher) OpenSearchWindow(
	ctx context.Context,
	from uint64,
	window int,
) *SearchWindow {
	session := s.txHistory.StartSession(ctx, s.pool, from, window)
	s.latestBlock.Store(from)
	return &SearchWindow{
		searcher: s,
		session:  session,
	}
}

func (s *AttackSearcher) SetAttackHandler(handler AttackHandler) {
	s.handler = handler
}

// SetPool provides a goroutine pool to the searcher, such that
// the SearchWindow opened by this AttackSearcher will search in parallel using this pool.
func (s *AttackSearcher) SetPool(pool *ants.Pool) {
	s.pool = pool
}

func (s *AttackSearcher) SetOracleHook(hook func(
	session *TxHistorySession,
	attackers, victims []common.Address,
	ar, vr, pr *TxRecord,
	attackTxAsIfSummary, victimTxAsIfSummary, profitTxAsIfSummary *summary.TxSummary,
)) {
	s.oracleHook = hook
}

func (s *AttackSearcher) CheckOracle(
	session *TxHistorySession,
	attackers, victims []common.Address,
	ar, vr, pr *TxRecord,
	attackTxAsIfSummary, victimTxAsIfSummary, profitTxAsIfSummary *summary.TxSummary,
) bool {
	if s.oracleHook != nil {
		s.oracleHook(
			session,
			attackers, victims,
			ar, vr, pr,
			attackTxAsIfSummary, victimTxAsIfSummary, profitTxAsIfSummary,
		)
	}

	if len(attackers) == 0 || len(victims) == 0 {
		return false
	}

	//var outOfGas bool
	//var matchOracle bool
	var attacker, victim common.Address
	var attackTxProfits, victimTxProfits, attackTxAsIfProfits, victimTxAsIfProfits summary.Profits
	var attackerProfits, attackerAsIfProfits, victimProfits, victimAsIfProfits summary.Profits

	// special treatment for out-of-gas caused by front-running
	//if vr.TxSummary.MsgCall().OutOfGas() != victimTxAsIfSummary.MsgCall().OutOfGas() {
	//	attacker = ar.Tx.From()
	//	victim = vr.Tx.From()
	//	outOfGas = true
	//}

	attackTxProfits = attackTxProfits.Add(ar.TxSummary.OverallProfits()...)
	victimTxProfits = victimTxProfits.Add(vr.TxSummary.OverallProfits()...)
	attackTxAsIfProfits = attackTxAsIfProfits.Add(attackTxAsIfSummary.OverallProfits()...)
	victimTxAsIfProfits = victimTxAsIfProfits.Add(victimTxAsIfSummary.OverallProfits()...)
	if pr != nil {
		attackTxProfits.Add(pr.TxSummary.OverallProfits()...)
	}
	if profitTxAsIfSummary != nil {
		attackTxAsIfProfits = attackTxAsIfProfits.Add(profitTxAsIfSummary.OverallProfits()...)
	}

	for _, attacker = range attackers {
		attackerProfits = attackTxProfits.ProfitsOf(attacker)
		attackerAsIfProfits = attackTxAsIfProfits.ProfitsOf(attacker)
		if cmp, err := attackerProfits.Cmp(attackerAsIfProfits); err == nil && cmp > 0 {
			// attacker gains more profits in the attack
			goto checkVictimLoss
		}
	}
	//if outOfGas {
	//	attacker = ar.Tx.From()
	//	goto buildAttack
	//}
	return false

checkVictimLoss:
	for _, victim = range victims {
		victimProfits = victimTxProfits.ProfitsOf(victim)
		victimAsIfProfits = victimTxAsIfProfits.ProfitsOf(victim)
		if cmp, err := victimProfits.Cmp(victimAsIfProfits); err == nil && cmp < 0 {
			// victim loses more profits in the attack
			//matchOracle = true
			goto buildAttack
		}
	}
	//if outOfGas {
	//	attacker = ar.Tx.From()
	//	victim = vr.Tx.From()
	//	goto buildAttack
	//}
	return false

buildAttack:
	// report attack
	attack := &Attack{
		Attacker:            attacker,
		Victim:              victim,
		AttackTxRecord:      ar,
		VictimTxRecord:      vr,
		ProfitTxRecord:      pr,
		AttackTxAsIfSummary: attackTxAsIfSummary,
		VictimTxAsIfSummary: victimTxAsIfSummary,
		ProfitTxAsIfSummary: profitTxAsIfSummary,
		//OutOfGas:            outOfGas,
		//MismatchOracle:      !matchOracle,

		attackerProfits:     attackerProfits,
		victimProfits:       victimProfits,
		attackerAsIfProfits: attackerAsIfProfits,
		victimAsIfProfits:   victimAsIfProfits,
	}
	if s.handler != nil {
		s.handler(session, attack)
	}
	return true
}

type SearchWindow struct {
	searcher *AttackSearcher

	session      *TxHistorySession
	allTxRecords []*TxRecord

	searchPivot TxPosition
	filter      func(ar, vr, pr *TxRecord) bool // nil to skip none. return false to skip the pair,
}

func (w *SearchWindow) TxHistorySession() *TxHistorySession {
	return w.session
}

// Close closes the SearchWindow and release the block resources used in this window.
func (w *SearchWindow) Close() {
	w.searcher.txHistory.ForgetBlockRange(
		w.session.From(),
		w.session.From()+uint64(w.session.Size()),
	)
	w.session.Close()
}

func (w *SearchWindow) SetSearchPivot(block uint64, txIndex uint) {
	w.searchPivot = TxPosition{
		BlockNumber: block,
		TxIndex:     txIndex,
	}
}

func (w *SearchWindow) SetFilter(f func(ar, vr, pr *TxRecord) bool) {
	w.filter = f
}

// SetFocus is a convenient method of SetFilter to only search for given attack tx pair.
// SetFocus invokes SetFilter so they cannot be used together.
func (w *SearchWindow) SetFocus(attackTx, victimTx common.Hash, profitTx *common.Hash) {
	w.SetFilter(func(ar, vr, pr *TxRecord) (keep bool) {
		// filter only the pair we care about
		if ar.Tx.Hash() == attackTx {
			if vr == nil {
				return true
			}
			if vr.Tx.Hash() == victimTx {
				if pr == nil {
					return true
				}
				if profitTx != nil && pr.Tx.Hash() == *profitTx {
					return true
				}
			}
		}
		return false
	})
}

// Search search in the window for attack cases.
// This function only returns when search is finished.
// This function is not thread-safe.
func (w *SearchWindow) Search(ctx context.Context) {
	var wg sync.WaitGroup

	w.allTxRecords = w.session.SliceTxRecords()

attackLoop:
	for ii, arr := range w.allTxRecords[:len(w.allTxRecords)-1] {
		// early exit if context is done
		select {
		case <-ctx.Done():
			log.Info().Err(ctx.Err()).Msg("Search early exit due to context done")
			return
		default:
		}

		i := ii
		ar := arr

		// skip error tx and pseudo tx
		if ar.Err != nil || ar.IsPseudo() {
			continue attackLoop
		}

		// skip filtered tx
		if w.filter != nil && !w.filter(ar, nil, nil) {
			continue
		}

		// TODO: should we keep this?
		// disprove if the attack tx is contract creation
		if ar.Tx.To() == nil {
			continue attackLoop
		} else {
			// disprove if the attack tx is simple ether transfer
			code, err := w.searcher.stateReader.CodeAt(
				ctx, *ar.Tx.To(), new(big.Int).Sub(ar.TxSummary.MsgCall().Receipt.BlockNumber, big.NewInt(1)),
			)
			if err != nil && len(code) == 0 {
				continue attackLoop
			}
		}

		if w.searcher.pool != nil {
			wg.Add(1)
			_ = w.searcher.pool.Submit(func() {
				defer wg.Done()
				w.searchVictimGivenAttack(ctx, i)
			})
		} else {
			w.searchVictimGivenAttack(ctx, i)
		}
	}

	wg.Wait()
}

func (w *SearchWindow) searchVictimGivenAttack(ctx context.Context, arIndex int) {
	var err error
	ar := w.allTxRecords[arIndex]
	log.Debug().Str("attack", ar.Tx.Hash().Hex()).Msg("Search as if it is an attack transaction")

	exeVM := &engine.ExeVM{
		Config: AttackSearchVMConfig(),
	}

victimLoop:
	for vrIndex, vr := range w.allTxRecords[arIndex+1:] {
		// early exit if context is done
		select {
		case <-ctx.Done():
			log.Info().Err(ctx.Err()).Msg("Search early exit due to context done")
			return
		default:
		}

		// skip error tx and pseudo tx
		if vr.Err != nil || vr.IsPseudo() {
			continue victimLoop
		}

		// skip filtered tx
		if w.filter != nil && !w.filter(ar, vr, nil) {
			continue
		}

		// skip if we know for sure contracts invoked by the victim tx is not verified
		if vr.SomeContractsNonVerified {
			continue victimLoop
		}

		// disprove if the victim tx is contract creation
		if vr.Tx.To() == nil {
			continue victimLoop
		} else {
			// disprove if the victimTx is simple ether transfer
			code, err := w.searcher.stateReader.CodeAt(
				ctx, *vr.Tx.To(), new(big.Int).Sub(vr.TxSummary.MsgCall().Receipt.BlockNumber, big.NewInt(1)),
			)
			if err != nil && len(code) == 0 {
				continue victimLoop
			}
		}

		// disprove if the attackTx and victimTx are from the same sender
		if ar.Tx.From() == vr.Tx.From() {
			continue victimLoop
		}

		// disprove if victimTx and attackTx has no overlap (accounts, contracts, etc)
		if !vr.NaiveOverlapWith(ar) {
			continue victimLoop
		}

		// disprove if victimTx does not depend on attackTx,
		// since otherwise the victim profits will be the same whether there is attack or not.
		if !w.searcher.hasDependency(ar.TxSummary, vr.TxSummary) {
			continue victimLoop
		}

		// now we reproduce as if the attack was not performed
		// reproduce victim before attack
		state := ar.State.Copy()
		vmContext := ar.VmContext.Copy()

		// execute the prerequisite transaction of victim transaction before attack
		exeVM.SetTracer(nil)
		prerequisites := w.session.SlicePrerequisites(vr, ar)
		for _, prerequisite := range prerequisites {
			log.Debug().
				Str("attackTx", ar.Tx.Hash().Hex()).
				Str("victimTx", vr.Tx.Hash().Hex()).
				Str("prerequisite", prerequisite.Tx.Hash().Hex()).
				Msg("Executing victim prerequisite")
			_, _, err := exeVM.ApplyTx(state, prerequisite.Tx, vmContext, false, false)
			if err != nil {
				if w.searcher.isWhitelistError(err) {
					continue victimLoop
				}
				log.Error().
					Err(err).
					Str("attackTx", ar.Tx.Hash().Hex()).
					Str("victimTx", vr.Tx.Hash().Hex()).
					Str("prerequisite", prerequisite.Tx.Hash().Hex()).
					Msg("Failed to apply victim prerequisite tx")
				continue victimLoop
			}
		}

		// execute the victim transaction
		log.Debug().
			Str("attackTx", ar.Tx.Hash().Hex()).
			Str("victimTx", vr.Tx.Hash().Hex()).
			Msg("Summarizing victim transaction as if no attack happened")
		tracer := summary.NewTxSummaryTracer(summary.Config{
			IncludeTransfer: true,
			IncludeProfit:   true,
			IncludeTrace:    true,
		})
		exeVM.SetTracer(tracer)
		_, _, err = exeVM.ApplyTx(state, vr.Tx, vmContext, false, true)
		if err != nil {
			if w.searcher.isWhitelistError(err) {
				continue victimLoop
			}
			log.Error().
				Err(err).
				Str("attackTx", ar.Tx.Hash().Hex()).
				Str("victimTx", vr.Tx.Hash().Hex()).
				Msg("Failed to summarize victim tx before attack")
			continue victimLoop
		}
		victimAsIfSummary := tracer.Summary

		// execute the attack transaction after victim
		log.Debug().
			Str("attackTx", ar.Tx.Hash().Hex()).
			Str("victimTx", vr.Tx.Hash().Hex()).
			Msg("Summarizing attack transaction as if no attack happened")
		tracer = summary.NewTxSummaryTracer(summary.Config{
			IncludeTransfer: true,
			IncludeProfit:   true,
			IncludeTrace:    true,
		})
		exeVM.SetTracer(tracer)
		_, _, err = exeVM.ApplyTx(state, ar.Tx, vmContext, false, true)
		if err != nil {
			if w.searcher.isWhitelistError(err) {
				continue victimLoop
			}
			log.Error().
				Err(err).
				Str("attackTx", ar.Tx.Hash().Hex()).
				Str("victimTx", vr.Tx.Hash().Hex()).
				Msg("Failed to summarize attack tx before after victim")
			continue victimLoop
		}
		attackAsIfSummary := tracer.Summary

		// infer attackers and victim
		attackers, victims := w.searcher.inferAttackersVictims(ar, vr, attackAsIfSummary, victimAsIfSummary)

		// only check if the victim is after search pivot
		if vr.Position().Cmp(w.searchPivot) >= 0 {
			log.Debug().
				Str("attackTx", ar.Tx.Hash().Hex()).
				Str("victimTx", vr.Tx.Hash().Hex()).
				Msg("Checking if it is a real attack")
			// this has not been searched in previous slide
			found := w.searcher.CheckOracle(
				w.session,
				attackers, victims,
				ar, vr, nil,
				attackAsIfSummary, victimAsIfSummary, nil,
			)
			if found {
				// if the attack is already found without profit transaction, we can stop here
				return
			}
		}

	profitLoop:
		for _, pr := range w.allTxRecords[arIndex+vrIndex+2:] {
			// early exit if context is done
			select {
			case <-ctx.Done():
				log.Info().Err(ctx.Err()).Msg("Search early exit due to context done")
				return
			default:
			}

			// skip if the profit transaction is not profitable
			if pr.Err != nil || pr.IsPseudo() {
				continue profitLoop
			}

			// skip if the profit transaction is not after the victim transaction
			if w.filter != nil && !w.filter(ar, vr, pr) {
				continue
			}

			// no need to search if the profit transaction is not newly added in previous txHistory slide
			// because in this case, attack, victim and profit tx must have been searched in previous slide
			if pr.Position().Cmp(w.searchPivot) < 0 {
				// this has been searched in previous slide
				continue profitLoop
			}

			// disprove if the victimTx and profitTx are from the same sender
			if vr.Tx.From() == pr.Tx.From() {
				continue profitLoop
			}

			// disprove if the profitTx and attackTx are from the different sender, and they don't have the same bot.
			// if pr.Tx.From() != ar.Tx.From() && pr.Tx.To() != ar.Tx.To() {
			// 	continue profitLoop
			// }

			// disprove if victimTx and attackTx has no overlap (accounts, contracts, etc)
			if !pr.NaiveOverlapWith(ar) {
				continue profitLoop
			}

			// disprove if attacker does not make any profits in profitTx
			attackersGetProfits := func(summary *summary.CallSummary) bool {
				profits := summary.OverallProfits()
				for _, attacker := range attackers {
					attackProfits := profits.ProfitsOf(attacker)
					for _, profit := range attackProfits {
						if profit.Positive() {
							return true
						}
					}
				}
				return false
			}
			// if we reach this point, attacker must have not made profits in attack transaction.
			// so we disapprove if profit transaction also cannot make profits.
			if !attackersGetProfits(pr.TxSummary) {
				continue
			}

			// now we reproduce profit tx after attack transaction
			copiedState, copiedVmContext := state.Copy(), vmContext.Copy()
			prerequisites = w.session.SlicePrerequisites(pr, ar)
			exeVM.SetTracer(nil)
			for _, prerequisite := range prerequisites {
				log.Debug().
					Str("attackTx", ar.Tx.Hash().Hex()).
					Str("victimTx", vr.Tx.Hash().Hex()).
					Str("profitTx", pr.Tx.Hash().Hex()).
					Str("prerequisite", prerequisite.Tx.Hash().Hex()).
					Msg("Executing victim prerequisite")
				_, _, err = exeVM.ApplyTx(copiedState, prerequisite.Tx, copiedVmContext, false, false)
				if err != nil {
					if w.searcher.isWhitelistError(err) {
						continue profitLoop
					}
					log.Error().Err(err).
						Str("attackTx", ar.Tx.Hash().Hex()).
						Str("victimTx", vr.Tx.Hash().Hex()).
						Str("profitTx", pr.Tx.Hash().Hex()).
						Str("prerequisite", prerequisite.Tx.Hash().Hex()).
						Msg("Failed to apply profit prerequisite tx")
					continue profitLoop
				}
			}

			log.Debug().
				Str("attackTx", ar.Tx.Hash().Hex()).
				Str("victimTx", vr.Tx.Hash().Hex()).
				Str("profitTx", pr.Tx.Hash().Hex()).
				Msg("Summarizing profit transaction as if no attack happened")
			tracer = summary.NewTxSummaryTracer(summary.Config{
				IncludeTransfer: true,
				IncludeProfit:   true,
				IncludeTrace:    true,
			})
			exeVM.SetTracer(tracer)
			_, _, err = exeVM.ApplyTx(copiedState, pr.Tx, copiedVmContext, false, true)
			if err != nil {
				if w.searcher.isWhitelistError(err) {
					continue profitLoop
				}
				log.Error().Err(err).
					Str("attackTx", ar.Tx.Hash().Hex()).
					Str("victimTx", vr.Tx.Hash().Hex()).
					Str("profitTx", pr.Tx.Hash().Hex()).
					Msg("Failed to summarize profit tx after attack")
				continue profitLoop
			}
			profitAsIfSummary := tracer.Summary

			log.Debug().
				Str("attackTx", ar.Tx.Hash().Hex()).
				Str("victimTx", vr.Tx.Hash().Hex()).
				Str("profitTx", pr.Tx.Hash().Hex()).
				Msg("Checking if it is a real attack")
			w.searcher.CheckOracle(
				w.session,
				attackers, victims,
				ar, vr, pr,
				attackAsIfSummary, victimAsIfSummary, profitAsIfSummary,
			)
		}
	}
}

// hasDependency returns true if the given dependant depends on the given dependency.
// The dependency relationship is inferred by shared summary.StateVariable.
// If dependant reads some summary.StateVariable that is written by dependency,
// then the dependency is considered to be true.
func (s *AttackSearcher) hasDependency(dependency, dependant *summary.CallSummary) bool {
	for _, dependencyV := range dependency.OverallDefs() {
		for _, dependantV := range dependant.OverallUses() {
			if dependencyV.Same(dependantV) {
				return true
			}
		}
	}
	return false
}

func (s *AttackSearcher) inferAttackersVictims(
	ar, vr *TxRecord,
	attackAsIfSummary, victimAsIfSummary *summary.TxSummary,
) (attackers, victims []common.Address) {
	// potential attackers and victims cannot overlap, otherwise return nil
	attackers = []common.Address{
		ar.TxSummary.MsgCall().Caller.StateAddr,
		ar.TxSummary.MsgCall().StateAddr,
	}
	victims = []common.Address{
		vr.TxSummary.MsgCall().Caller.StateAddr,
	}

	for _, a := range attackers {
		for _, v := range victims {
			if a == v {
				return nil, nil
			}
		}
	}

	// attackerWhitelist := map[common.Address]bool{
	// 	common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"): true, // WETH contract
	// }

	// attackersMap, victimsMap := make(map[common.Address]bool), make(map[common.Address]bool)
	// var pAttackers, pVictims []common.Address
	// for _, p := range append(
	// 	ar.TxSummary.OverallProfits(),
	// 	attackAsIfSummary.OverallProfits()...,
	// ) {
	// 	beneficiary := p.Beneficiary()
	// 	if !attackerWhitelist[beneficiary] {
	// 		attackersMap[beneficiary] = true
	// 		pAttackers = append(pAttackers, beneficiary)
	// 	}
	// }
	// for _, p := range append(
	// 	vr.TxSummary.OverallProfits(),
	// 	victimAsIfSummary.OverallProfits()...,
	// ) {
	// 	victimsMap[p.Beneficiary()] = true
	// 	pVictims = append(pVictims, p.Beneficiary())
	// }
	// for _, addr := range pAttackers {
	// 	if !victimsMap[addr] {
	// 		attackers = append(attackers, addr)
	// 	}
	// }
	// for _, addr := range pVictims {
	// 	if !attackersMap[addr] {
	// 		victims = append(victims, addr)
	// 	}
	// }

	return attackers, victims
}
