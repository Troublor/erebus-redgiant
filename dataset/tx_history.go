package dataset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/Troublor/erebus-redgiant/chain"
	"github.com/Troublor/erebus-redgiant/helpers"

	"github.com/Troublor/erebus-redgiant/analysis/summary"
	"github.com/Troublor/erebus/troubeth"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"

	engine "github.com/Troublor/erebus-redgiant/dyengine"
	. "github.com/Troublor/erebus-redgiant/dyengine/state"
	"github.com/Workiva/go-datastructures/queue"
	merge "github.com/Workiva/go-datastructures/sort"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	graphSimple "gonum.org/v1/gonum/graph/simple"
)

var ZeroTxPosition = TxPosition{0, 0}

type TxPosition struct {
	BlockNumber uint64
	TxIndex     uint
}

func (p TxPosition) String() string {
	return fmt.Sprintf("%d:%d", p.BlockNumber, p.TxIndex)
}

func (p TxPosition) Cmp(other TxPosition) int {
	if p.BlockNumber < other.BlockNumber {
		return -1
	} else if p.BlockNumber > other.BlockNumber {
		return 1
	} else {
		if p.TxIndex < other.TxIndex {
			return -1
		} else if p.TxIndex > other.TxIndex {
			return 1
		} else {
			return 0
		}
	}
}

type TxRecord struct {
	BlockNumber      *big.Int
	BlockHash        common.Hash
	TransactionIndex uint
	State            engine.State
	VmContext        *engine.VMContext
	Tx               *engine.Tx
	TxSummary        *summary.CallSummary
	Err              error

	SomeContractsNonVerified bool

	// cache
	cacheMu           sync.Mutex
	involvedAddresses []common.Address
}

// BuildTxRecordFromHash returns a TxRecord for a given transaction hash on mainnet.
// This has low performance since the state and vmContext of the TxRecord is built from scratch always.
// It is recommended to use TxHistory2 to get a batch of TxRecords in consecutive blocks.
func BuildTxRecordFromHash(
	ctx context.Context,
	chainReader chain.BlockchainReader,
	txHash common.Hash,
) (*TxRecord, error) {
	receipt, err := chainReader.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, err
	}
	state, vmContext, err := helpers.PrepareStateAndContext(
		ctx, chainReader, receipt.BlockNumber, receipt.TransactionIndex,
	)
	if err != nil {
		return nil, err
	}
	record := &TxRecord{
		BlockNumber:      receipt.BlockNumber,
		BlockHash:        receipt.BlockHash,
		TransactionIndex: receipt.TransactionIndex,
		State:            state,
		VmContext:        vmContext,
	}
	rawTx, _, err := chainReader.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}
	signer := types.MakeSigner(helpers.VMConfigOnMainnet().ChainConfig, receipt.BlockNumber)
	record.Tx, record.Err = engine.TxFromTransactionWithSigner(rawTx, signer)
	if record.Err != nil {
		return record, record.Err
	}

	exeVM := engine.NewExeVM(helpers.VMConfigOnMainnet())
	// summarize transaction
	tracer := summary.NewTxSummaryTracer(summary.Config{
		IncludeDef:      true,
		IncludeUse:      true,
		IncludeTransfer: true,
		IncludeProfit:   true,
	})
	exeVM.SetTracer(tracer)
	_, _, record.Err = exeVM.ApplyTx(state, record.Tx, vmContext, false, true)
	if record.Err != nil {
		return record, record.Err
	}
	record.TxSummary = tracer.Summary
	return record, nil
}

func (r *TxRecord) ID() int64 {
	pos := r.Position()
	return int64(pos.BlockNumber)*10000 + int64(pos.TxIndex)
}

func (r *TxRecord) Position() TxPosition {
	if r.Err != nil {
		panic("tx record with error does not have position")
	}
	if r.IsPseudo() {
		var pos TxPosition
		_ = json.Unmarshal(r.Tx.Data(), &pos)
		return pos
	}
	receipt := r.TxSummary.MsgCall().Receipt
	// This way to compute id requires the total number of transactions in a block to be less than 1000.
	return TxPosition{
		BlockNumber: receipt.BlockNumber.Uint64(),
		TxIndex:     receipt.TransactionIndex,
	}
}

func (r *TxRecord) IsPseudo() bool {
	return r.Tx.IsPseudo()
}

func (r *TxRecord) Compare(another merge.Comparator) int {
	other := another.(*TxRecord)
	return r.Position().Cmp(other.Position())
}

func (r *TxRecord) InvolvedAddresses() []common.Address {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	if r.IsPseudo() {
		return []common.Address{}
	}

	if r.involvedAddresses == nil {
		var getAddresses func(*summary.CallSummary) []common.Address
		getAddresses = func(sum *summary.CallSummary) (addresses []common.Address) {
			addresses = append(addresses, sum.MsgCall().Caller.StateAddr)
			addresses = append(addresses, sum.MsgCall().StateAddr)
			for _, nested := range sum.NestedSummaries() {
				addresses = append(addresses, getAddresses(nested)...)
			}
			return addresses
		}

		addresses := getAddresses(r.TxSummary)
		// remove duplicates
		included := make(map[common.Address]bool)
		r.involvedAddresses = make([]common.Address, 0, len(addresses))
		for _, addr := range addresses {
			if !included[addr] {
				r.involvedAddresses = append(r.involvedAddresses, addr)
				included[addr] = true
			}
		}
	}
	return r.involvedAddresses
}

func (r *TxRecord) NaiveOverlapWith(another *TxRecord) bool {
	if r.IsPseudo() {
		return false
	}

	thisAddresses := r.InvolvedAddresses()
	anotherAddresses := another.InvolvedAddresses()
	included := make(map[common.Address]bool)
	for _, addr := range thisAddresses {
		included[addr] = true
	}
	for _, addr := range anotherAddresses {
		if included[addr] {
			return true
		}
	}
	return false
}

// TxHistorySession represents a session of TxHistory
// consisting of the block hitory data within a block window.
// It is not thread safe.
// Parallel can be done between multiple session instances.
type TxHistorySession struct {
	txHistory *TxHistory

	from    uint64
	blocks  []Block
	hbGraph *graphSimple.DirectedGraph
}

type Block []*TxRecord

type blockRecord struct {
	Block
	loaded bool
	mu     sync.Mutex
}

type TxHistory struct {
	chainReader          chain.BlockchainReader
	contractInfoProvider *ContractInfoProvider

	// blocks is a list of Block containing TxRecord values.
	// Once a Block is appended in blocks, it should not be modified (only delete is allowed)
	blocks sync.Map // map[uint64]*blockRecord

	mu sync.Mutex
}

func NewTxHistory(
	chainReader chain.BlockchainReader, troubEth *troubeth.TroubEth,
) *TxHistory {
	var contractInfoProvider *ContractInfoProvider
	if troubEth != nil {
		contractInfoProvider = NewContractInfoProvider(troubEth)
	}
	return &TxHistory{
		chainReader:          chainReader,
		contractInfoProvider: contractInfoProvider,
	}
}

// computeTxRecords computes TxRecord values in a list of blocks.
// computeTxRecords will parallel the processing of blocks using the given threadPool.
// This function will always return a list of Block, if any error occurs when processing any transaction,
// the error will be recorded in TxRecord.Err.
// TxRecord.Err should be checked before using TxRecord.
// This function is thread-safe.
func (h *TxHistory) computeTxRecords(ctx context.Context, blockNumber uint64) (b Block) {
	state, err := NewForkedState(h.chainReader, big.NewInt(int64(blockNumber-1)))
	if err != nil {
		log.Error().Err(err).Msg("Failed to construct ForkedState")
		return
	}

	// fetch block from chainReader
	var block *types.Block
	block, err = h.chainReader.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		log.Error().Err(err).Uint64("number", blockNumber).Msg("Failed to get block")
		return
	}

	vmContext := helpers.VMContextFromBlock(block)
	exeVM := engine.NewExeVM(helpers.VMConfigOnMainnet())
	var recordAsyncWG sync.WaitGroup
	// defer recordAsyncWG.Wait()
	for j, rawTx := range block.Transactions() {
		// if any error occurs during summarize transaction,
		// the error will be stored in record.Err
		record := &TxRecord{
			BlockNumber:      block.Number(),
			BlockHash:        block.Hash(),
			TransactionIndex: uint(j),
			State:            state.Copy(),
			VmContext:        vmContext.Copy(),
		}
		b = append(b, record)

		signer := types.MakeSigner(helpers.VMConfigOnMainnet().ChainConfig, block.Number())
		record.Tx, record.Err = engine.TxFromTransactionWithSigner(rawTx, signer)
		if record.Err != nil {
			if errors.Is(record.Err, context.Canceled) {
				return
			}
			log.Error().Err(record.Err).Str("tx", rawTx.Hash().Hex()).Msg("Failed to construct tx")
			continue
		}

		// summarize transaction
		tracer := summary.NewTxSummaryTracer(summary.Config{
			IncludeDef:      true,
			IncludeUse:      true,
			IncludeTransfer: true,
			IncludeProfit:   true,
		})
		exeVM.SetTracer(tracer)
		_, _, record.Err = exeVM.ApplyTx(state, record.Tx, vmContext, false, true)
		if record.Err != nil {
			if errors.Is(record.Err, context.Canceled) {
				return
			}
			log.Error().Err(record.Err).Str("tx", rawTx.Hash().Hex()).Msg("Failed to summary tx")
			continue
		}
		record.TxSummary = tracer.Summary

		if h.contractInfoProvider != nil {
			recordAsyncWG.Add(1)
			go func() {
				defer recordAsyncWG.Done()
				for _, addr := range record.TxSummary.AllInvokedAddresses() {
					address := addr
					code, err := h.chainReader.CodeAt(ctx, addr, block.Number())
					if err != nil || len(code) == 0 {
						// this is not a contract
						continue
					}
					ctx, cancel := context.WithTimeout(ctx, time.Second)
					verified, err := h.contractInfoProvider.IsVerified(ctx, address)
					cancel()
					if err != nil {
						// be conservative
						log.Error().
							Err(err).
							Str("address", address.Hex()).
							Msg("Failed to check contract verified")
						verified = true
					}
					if !verified {
						record.SomeContractsNonVerified = true
						break
					}
				}
			}()
		}
	}

	// Add block reward pseudo transaction
	blockRewardRecord := &TxRecord{
		State:     state.Copy(),
		VmContext: vmContext.Copy(),
	}
	b = append(b, blockRewardRecord)
	// Select the correct block reward based on chain progression
	blockReward := ethash.FrontierBlockReward
	chainConfig := params.MainnetChainConfig
	if chainConfig.IsByzantium(block.Number()) {
		blockReward = ethash.ByzantiumBlockReward
	}
	if chainConfig.IsConstantinople(block.Number()) {
		blockReward = ethash.ConstantinopleBlockReward
	}
	pos := TxPosition{block.NumberU64(), uint(block.Transactions().Len())}
	posBytes, _ := json.Marshal(pos)
	blockRewardRecord.Tx = engine.NewPseudoTx(posBytes, func(s engine.State) ([]byte, error) {
		s.AddBalance(block.Coinbase(), blockReward)
		return []byte(fmt.Sprintf("%s block reward to %s", blockReward, block.Coinbase())), nil
	})
	// Summarize block reward pseudo transaction
	tracer := summary.NewTxSummaryTracer(summary.Config{
		IncludeTransfer: true,
		IncludeProfit:   true,
	})
	exeVM.SetTracer(tracer)
	_, _, blockRewardRecord.Err = exeVM.ApplyTx(state, blockRewardRecord.Tx, vmContext, false, true)
	if blockRewardRecord.Err != nil {
		log.Error().
			Err(blockRewardRecord.Err).
			Str("tx", blockRewardRecord.Tx.Hash().Hex()).
			Msg("Failed to summary pseudo tx")
	}
	blockRewardRecord.TxSummary = tracer.Summary

	return b
}

func (h *TxHistory) ForgetBlocks(blockNumbers ...uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, bn := range blockNumbers {
		h.blocks.Delete(bn)
	}
}

func (h *TxHistory) ForgetBlockRange(from, to uint64) {
	blockRange := make([]uint64, to-from+1)
	for i := 0; i < len(blockRange); i++ {
		blockRange[i] = from + uint64(i)
	}
	h.ForgetBlocks(blockRange...)
}

// AcquireBlocks concurrently fetches blocks with the given Block numbers if the Block is not available.
// It also maintains a useCount for each Block number.
// Thread safe.
func (h *TxHistory) AcquireBlocks(
	ctx context.Context,
	pool *ants.Pool,
	blockNumbers ...uint64,
) []Block {
	var wg sync.WaitGroup
	var blocks = make([]Block, len(blockNumbers))
	for i, bn := range blockNumbers {
		index := i
		blockNumber := bn

		h.mu.Lock()
		blockR, _ := h.blocks.LoadOrStore(blockNumber, &blockRecord{})
		br := blockR.(*blockRecord)
		h.mu.Unlock()

		br.mu.Lock()
		if br.loaded {
			br.mu.Unlock()
			log.Debug().Uint64("block", blockNumber).Msg("Block is already available")
			blocks[index] = blockR.(*blockRecord).Block
		} else {
			if pool != nil {
				wg.Add(1)
				_ = pool.Submit(func() {
					defer br.mu.Unlock()
					defer wg.Done()
					log.Info().Uint64("block", blockNumber).Msg("Computing tx records for block")
					b := h.computeTxRecords(ctx, blockNumber)
					br.Block = b
					br.loaded = true
					blocks[index] = b
				})
			} else {
				log.Info().Uint64("block", blockNumber).Msg("Computing tx records for block")
				b := h.computeTxRecords(ctx, blockNumber)
				br.Block = b
				br.loaded = true
				blocks[index] = b
				br.mu.Unlock()
			}
		}
	}

	wg.Wait()
	return blocks
}

// StartSession starts a session of tx history.
// It is essentially a slice of TxHistory2.
// It will fetch the new blocks if needed in TxHistory2.
// Thread safe.
func (h *TxHistory) StartSession(
	ctx context.Context, pool *ants.Pool, fromBlock uint64, windowSize int,
) *TxHistorySession {
	log.Debug().
		Uint64("from", fromBlock).
		Int("window", windowSize).
		Msg("TxHistory session acquiring blocks")
	defer log.Debug().
		Uint64("from", fromBlock).
		Int("window", windowSize).
		Msg("TxHistory session acquiring blocks done")
	bns := make([]uint64, windowSize)
	for i := 0; i < windowSize; i++ {
		bns[i] = fromBlock + uint64(i)
	}
	blocks := h.AcquireBlocks(ctx, pool, bns...)

	// build happen-before graph
	log.Debug().
		Uint64("from", fromBlock).
		Int("window", windowSize).
		Msg("Building happen-before graph")
	hbGraph := graphSimple.NewDirectedGraph()
	senderLastRecord := make(map[common.Address]*TxRecord)
	for _, block := range blocks {
		for _, r := range block {
			if r.Err != nil {
				continue
			}
			if r.Tx.IsPseudo() {
				continue
			}
			hbGraph.AddNode(r)
			if last, exist := senderLastRecord[r.Tx.From()]; exist {
				if r.Tx.Nonce() <= last.Tx.Nonce() {
					panic("nonce of sender is not monotonic")
				}
				if r.Tx.From() != last.Tx.From() {
					panic("sender is not consistent")
				}

				hbGraph.SetEdge(hbGraph.NewEdge(r, last))
			}
			senderLastRecord[r.Tx.From()] = r
		}
	}

	return &TxHistorySession{
		txHistory: h,

		from:    fromBlock,
		blocks:  blocks,
		hbGraph: hbGraph,
	}
}

func (s *TxHistorySession) From() uint64 {
	return s.from
}

func (s *TxHistorySession) Size() int {
	return len(s.blocks)
}

func (s *TxHistorySession) GetTxRecordByHash(hash common.Hash) (*TxRecord, error) {
	for _, block := range s.blocks {
		for _, txRecords := range block {
			if txRecords.Tx.Hash() == hash {
				return txRecords, nil
			}
		}
	}
	return nil, errors.New("not found")
}

func (s *TxHistorySession) TryGetTxRecord(pos TxPosition) (*TxRecord, error) {
	if pos.BlockNumber < s.from || pos.BlockNumber >= s.from+uint64(len(s.blocks)) {
		return nil, fmt.Errorf(
			"block number %d is out of range [%d, %d)",
			pos.BlockNumber, s.from, s.from+uint64(len(s.blocks)),
		)
	}
	block := s.blocks[pos.BlockNumber-s.from]
	if pos.TxIndex >= uint(len(block)) {
		return nil, fmt.Errorf("tx index %d is out of range [0, %d)", pos.TxIndex, len(block))
	}
	return block[pos.TxIndex], nil
}

func (s *TxHistorySession) GetTxRecord(pos TxPosition) *TxRecord {
	if r, err := s.TryGetTxRecord(pos); err != nil {
		panic(err)
	} else {
		return r
	}
}

// SlicePrerequisites returns the list of prerequisites (excluding itself) of the given tx position.
// The list of prerequisites is sorted by the position of the tx in the block.
// The list of prerequisites is bounded by the backBound TxPosition (not included).
// If the backBound is larger than or equal to the dependant, this function will act as if there is no bound.
func (s *TxHistorySession) SlicePrerequisites(
	dependant *TxRecord,
	backBound *TxRecord,
) []*TxRecord {
	slice := make([]merge.Comparator, 0)
	r := dependant
	if r.Err != nil {
		panic("dependencies of tx with error are not available")
	}
	if r.IsPseudo() {
		return make([]*TxRecord, 0)
	}

	var bound TxPosition
	if dependant.Position().Cmp(backBound.Position()) > 0 {
		bound = backBound.Position()
	} else {
		bound = ZeroTxPosition
	}

	spanningQueue := queue.New(64)
	_ = spanningQueue.Put(r)
	for spanningQueue.Len() > 0 {
		records, _ := spanningQueue.Get(1)
		dependencies := s.hbGraph.From(records[0].(*TxRecord).ID())
		for dependencies.Next() {
			dep := dependencies.Node().(*TxRecord)
			if dep.Position().Cmp(bound) > 0 {
				slice = merge.SymMerge(slice, []merge.Comparator{dep})
				_ = spanningQueue.Put(dep)
			}
		}
	}

	returns := make([]*TxRecord, len(slice))
	for i, v := range slice {
		returns[i] = v.(*TxRecord)
	}
	return returns
}

// SliceTxRecords returns the list of tx records in the given slice.
// The slice is specified by the from and to TxPosition (to TxPosition is not included).
func (s *TxHistorySession) SliceTxRecords(positions ...TxPosition) []*TxRecord {
	var returns []*TxRecord
	var fromPos TxPosition
	var toPos TxPosition
	switch len(positions) {
	case 0:
		fromPos = TxPosition{BlockNumber: s.from, TxIndex: 0}
		toPos = TxPosition{BlockNumber: s.from + uint64(len(s.blocks)), TxIndex: 0}
	case 1:
		fromPos = positions[0]
		toPos = TxPosition{BlockNumber: s.from + uint64(len(s.blocks)), TxIndex: 0}
	case 2:
		fromPos = positions[0]
		toPos = positions[1]
	default:
		panic("too many arguments")
	}
	pos := fromPos
	for pos.Cmp(toPos) < 0 {
		r, err := s.TryGetTxRecord(pos)
		if err != nil {
			pos.BlockNumber++
			pos.TxIndex = 0
		} else {
			returns = append(returns, r)
			pos.TxIndex++
		}
	}
	return returns
}

// Close closes the session and release the blocks in the underlying TxHistory2 blocks cache.
func (s *TxHistorySession) Close() {
	s.txHistory.ForgetBlockRange(s.from, s.from+uint64(len(s.blocks)))
	log.Info().
		Int("from", int(s.from)).
		Int("window", len(s.blocks)).
		Msg("Closed TxHistory session")
}
