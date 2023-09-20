package readers

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"os"
	"path"
	"runtime"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon-lib/kv/mdbx"
	erigonCommon "github.com/ledgerwatch/erigon/common"
	erigonRawdb "github.com/ledgerwatch/erigon/core/rawdb"
	erigonTypes "github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/eth/stagedsync/stages"
	"github.com/ledgerwatch/erigon/turbo/adapter"
	"github.com/ledgerwatch/erigon/turbo/rpchelper"
	erigonLog "github.com/ledgerwatch/log/v3"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/semaphore"
)

type erigonDBReader struct {
	db kv.RoDB
}

func NewErigonDBReader(ctx context.Context, datadir string) (*erigonDBReader, error) {
	logger := erigonLog.New()
	logger.SetHandler(erigonZeroLogHandler())
	limiter := semaphore.NewWeighted(int64(runtime.NumCPU()))
	datadir = path.Join(datadir, "chaindata")
	_, err := os.Stat(datadir)
	if err != nil {
		return nil, fmt.Errorf("failed to stat erigon db: %w", err)
	}
	db, err := mdbx.NewMDBX(logger).RoTxsLimiter(limiter).Path(datadir).Readonly().Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open erigon db: %w", err)
	}
	if compatErr := checkDbCompatibility(ctx, db); compatErr != nil {
		return nil, fmt.Errorf("erigon db incompatible: %w", compatErr)
	}
	return &erigonDBReader{db: db}, nil
}

func (e *erigonDBReader) Close() error {
	e.db.Close()
	return nil
}

func (e *erigonDBReader) BalanceAt(
	ctx context.Context, account common.Address, blockNumber *big.Int,
) (*big.Int, error) {
	dbtx, err := e.db.BeginRo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer dbtx.Rollback()

	if blockNumber == nil {
		bn, err := e.BlockNumber(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest block number: %w", err)
		}
		blockNumber = big.NewInt(int64(bn))
	}

	acc, err := rpchelper.GetAccount(dbtx, blockNumber.Uint64(), toErigonAddress(account))
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if acc == nil {
		return big.NewInt(0), nil
	}
	return acc.Balance.ToBig(), nil
}

func (e *erigonDBReader) StorageAt(
	ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int,
) ([]byte, error) {
	dbtx, err := e.db.BeginRo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer dbtx.Rollback()

	if blockNumber == nil {
		bn, err := e.BlockNumber(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest block number: %w", err)
		}
		blockNumber = big.NewInt(int64(bn))
	}

	reader := adapter.NewStateReader(dbtx, blockNumber.Uint64())
	acc, err := reader.ReadAccountData(toErigonAddress(account))
	if err != nil {
		return nil, fmt.Errorf("failed to read acount data: %w", err)
	}
	if acc == nil {
		return []byte{}, nil
	}

	res, err := reader.ReadAccountStorage(
		toErigonAddress(account),
		acc.Incarnation,
		(*erigonCommon.Hash)(&key),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to read account storage: %w", err)
	}
	if res == nil {
		return []byte{}, nil
	}
	return common.LeftPadBytes(res, 32), nil
}

func (e *erigonDBReader) CodeAt(
	ctx context.Context,
	account common.Address,
	blockNumber *big.Int,
) ([]byte, error) {
	dbtx, err := e.db.BeginRo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer dbtx.Rollback()

	if blockNumber == nil {
		bn, err := e.BlockNumber(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest block number: %w", err)
		}
		blockNumber = big.NewInt(int64(bn))
	}

	reader := adapter.NewStateReader(dbtx, blockNumber.Uint64())
	acc, err := reader.ReadAccountData(toErigonAddress(account))
	if err != nil {
		return nil, fmt.Errorf("failed to read acount data: %w", err)
	}
	if acc == nil {
		return []byte{}, nil
	}

	res, err := reader.ReadAccountCode(toErigonAddress(account), acc.Incarnation, acc.CodeHash)
	if err != nil {
		return nil, fmt.Errorf("failed to read account code: %w", err)
	}
	return res, nil
}

func (e *erigonDBReader) NonceAt(
	ctx context.Context,
	account common.Address,
	blockNumber *big.Int,
) (uint64, error) {
	dbtx, err := e.db.BeginRo(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer dbtx.Rollback()

	if blockNumber == nil {
		bn, err := e.BlockNumber(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to get latest block number: %w", err)
		}
		blockNumber = big.NewInt(int64(bn))
	}

	reader := adapter.NewStateReader(dbtx, blockNumber.Uint64())
	acc, err := reader.ReadAccountData(toErigonAddress(account))
	if err != nil {
		return 0, fmt.Errorf("failed to read acount data: %w", err)
	}
	if acc == nil {
		return 0, nil
	}
	return acc.Nonce, nil
}

func (e *erigonDBReader) TransactionByHash(
	ctx context.Context, txHash common.Hash,
) (tx *types.Transaction, isPending bool, err error) {
	dbtx, err := e.db.BeginRo(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer dbtx.Rollback()

	erigonTx, _, _, _, err := erigonRawdb.ReadTransactionByHash(dbtx, toErigonHash(txHash))
	if err != nil {
		return nil, false, fmt.Errorf("failed to read transaction by hash: %w", err)
	}
	if erigonTx == nil {
		return nil, false, ethereum.NotFound
	}
	return fromErigonTransaction(erigonTx), false, nil
}

func (e *erigonDBReader) TransactionReceipt(
	ctx context.Context,
	txHash common.Hash,
) (*types.Receipt, error) {
	dbtx, err := e.db.BeginRo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer dbtx.Rollback()

	erigonReceipt, _, _, _, err := erigonRawdb.ReadReceipt(dbtx, toErigonHash(txHash))
	if err != nil {
		return nil, fmt.Errorf("failed to read receipt by hash: %w", err)
	}
	if erigonReceipt == nil {
		return nil, ethereum.NotFound
	}
	for _, l := range erigonReceipt.Logs {
		if l.Data == nil {
			l.Data = []byte{}
		}
	}
	return fromErigonReceipt(erigonReceipt), nil
}

func (e *erigonDBReader) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	dbtx, err := e.db.BeginRo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer dbtx.Rollback()

	erigonBlock, err := erigonRawdb.ReadBlockByHash(dbtx, toErigonHash(hash))
	if err != nil {
		return nil, fmt.Errorf("failed to read block by hash: %w", err)
	}
	if erigonBlock == nil {
		return nil, ethereum.NotFound
	}
	return fromErigonBlock(erigonBlock), nil
}

func (e *erigonDBReader) BlockByNumber(
	ctx context.Context,
	blockNumber *big.Int,
) (*types.Block, error) {
	dbtx, err := e.db.BeginRo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer dbtx.Rollback()

	if blockNumber == nil {
		bn, err := e.BlockNumber(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest block number: %w", err)
		}
		blockNumber = big.NewInt(int64(bn))
	}

	erigonBlock, err := erigonRawdb.ReadBlockByNumber(dbtx, blockNumber.Uint64())
	if err != nil {
		return nil, fmt.Errorf("failed to read block by number: %w", err)
	}
	if erigonBlock == nil {
		return nil, ethereum.NotFound
	}
	return fromErigonBlock(erigonBlock), nil
}

func (e *erigonDBReader) HeaderByHash(
	ctx context.Context,
	hash common.Hash,
) (*types.Header, error) {
	dbtx, err := e.db.BeginRo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer dbtx.Rollback()

	erigonHeader, err := erigonRawdb.ReadHeaderByHash(dbtx, toErigonHash(hash))
	if err != nil {
		return nil, fmt.Errorf("failed to read header by hash: %w", err)
	}
	if erigonHeader == nil {
		return nil, ethereum.NotFound
	}
	return fromErigonHeader(erigonHeader), nil
}

func (e *erigonDBReader) HeaderByNumber(
	ctx context.Context,
	blockNumber *big.Int,
) (*types.Header, error) {
	dbtx, err := e.db.BeginRo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer dbtx.Rollback()

	if blockNumber == nil {
		bn, err := e.BlockNumber(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest block number: %w", err)
		}
		blockNumber = big.NewInt(int64(bn))
	}

	erigonHeader := erigonRawdb.ReadHeaderByNumber(dbtx, blockNumber.Uint64())
	if erigonHeader == nil {
		return nil, ethereum.NotFound
	}
	return fromErigonHeader(erigonHeader), nil
}

func (e *erigonDBReader) TransactionCount(
	ctx context.Context,
	blockHash common.Hash,
) (uint, error) {
	block, err := e.BlockByHash(ctx, blockHash)
	if err != nil {
		return 0, fmt.Errorf("failed to get block by hash: %w", err)
	}
	if block == nil {
		return 0, ethereum.NotFound
	}
	return uint(block.Transactions().Len()), nil
}

func (e *erigonDBReader) TransactionInBlock(
	ctx context.Context, blockHash common.Hash, index uint,
) (*types.Transaction, error) {
	dbtx, err := e.db.BeginRo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer dbtx.Rollback()

	bN := erigonRawdb.ReadHeaderNumber(dbtx, toErigonHash(blockHash))
	if bN == nil {
		return nil, ethereum.NotFound
	}
	body, err := erigonRawdb.ReadBodyWithTransactions(dbtx, toErigonHash(blockHash), *bN)
	if err != nil {
		return nil, fmt.Errorf("failed to read body with transactions: %w", err)
	}
	if body == nil {
		return nil, ethereum.NotFound
	}
	return fromErigonTransaction(body.Transactions[index]), nil
}

func (e *erigonDBReader) ChainID(ctx context.Context) (*big.Int, error) {
	dbtx, err := e.db.BeginRo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer dbtx.Rollback()

	genesisBlock, err := erigonRawdb.ReadBlockByNumber(dbtx, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to read genesis block: %w", err)
	}
	cc, err := erigonRawdb.ReadChainConfig(dbtx, genesisBlock.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to read chain config: %w", err)
	}
	return cc.ChainID, nil
}

func (e *erigonDBReader) BlockNumber(ctx context.Context) (uint64, error) {
	dbtx, err := e.db.BeginRo(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer dbtx.Rollback()

	execution, err := stages.GetStageProgress(dbtx, stages.Finish)
	if err != nil {
		return 0, fmt.Errorf("failed to get stage progress: %w", err)
	}
	return execution, nil
}

func (e *erigonDBReader) BlockHashByNumber(
	ctx context.Context,
	blockNumber *big.Int,
) (common.Hash, error) {
	dbtx, err := e.db.BeginRo(ctx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer dbtx.Rollback()

	if blockNumber == nil {
		bn, err := e.BlockNumber(ctx)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to get latest block number: %w", err)
		}
		blockNumber = big.NewInt(int64(bn))
	}

	block, err := erigonRawdb.ReadBlockByNumber(dbtx, blockNumber.Uint64())
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to read block by number: %w", err)
	}
	return common.Hash(block.Hash()), nil
}

// checkDbCompatibility is ported from erigon/cmd/rpcdaemon/cli/config.go.
func checkDbCompatibility(ctx context.Context, db kv.RoDB) error {
	// DB schema version compatibility check
	var version []byte
	var compatErr error
	var compatTx kv.Tx
	if compatTx, compatErr = db.BeginRo(ctx); compatErr != nil {
		return fmt.Errorf("open Ro Tx for DB schema compability check: %w", compatErr)
	}
	defer compatTx.Rollback()
	if version, compatErr = compatTx.GetOne(kv.DatabaseInfo, kv.DBSchemaVersionKey); compatErr != nil {
		return fmt.Errorf("read version for DB schema compability check: %w", compatErr)
	}
	if len(version) != 12 {
		return fmt.Errorf(
			"database does not have major schema version. upgrade and restart Erigon core",
		)
	}
	major := binary.BigEndian.Uint32(version)
	minor := binary.BigEndian.Uint32(version[4:])
	patch := binary.BigEndian.Uint32(version[8:])
	var compatible bool
	dbSchemaVersion := &kv.DBSchemaVersion
	if major != dbSchemaVersion.Major {
		compatible = false
	} else if minor != dbSchemaVersion.Minor {
		compatible = false
	} else {
		compatible = true
	}
	if !compatible {
		return fmt.Errorf("incompatible DB Schema versions: reader %d.%d.%d, database %d.%d.%d",
			dbSchemaVersion.Major, dbSchemaVersion.Minor, dbSchemaVersion.Patch,
			major, minor, patch)
	}
	log.Info().
		Str("reader", fmt.Sprintf("%d.%d.%d", dbSchemaVersion.Major, dbSchemaVersion.Minor, dbSchemaVersion.Patch)).
		Str("database", fmt.Sprintf("%d.%d.%d", major, minor, patch)).
		Msg("DB Schema compatible")
	return nil
}

/* Type converters */

func toErigonAddress(addr common.Address) erigonCommon.Address {
	return erigonCommon.Address(addr)
}

func toErigonHash(hash common.Hash) erigonCommon.Hash {
	return erigonCommon.Hash(hash)
}

func fromErigonAddress(addr erigonCommon.Address) common.Address {
	return common.Address(addr)
}

func fromErigonHash(hash erigonCommon.Hash) common.Hash {
	return common.Hash(hash)
}

func fromErigonAccessList(accessList erigonTypes.AccessList) types.AccessList {
	if accessList == nil {
		return nil
	}
	l := make(types.AccessList, len(accessList))
	for i := 0; i < len(l); i++ {
		ks := make([]common.Hash, len(accessList[i].StorageKeys))
		for j := 0; j < len(ks); j++ {
			ks[j] = fromErigonHash(accessList[i].StorageKeys[j])
		}
		l[i] = types.AccessTuple{
			Address:     fromErigonAddress(accessList[i].Address),
			StorageKeys: ks,
		}
	}
	return l
}

func fromErigonTransaction(tx erigonTypes.Transaction) *types.Transaction {
	if tx == nil {
		return nil
	}
	var to *common.Address
	if tx.GetTo() != nil {
		temp := fromErigonAddress(*tx.GetTo())
		to = &temp
	}
	V, R, S := tx.RawSignatureValues()
	switch tx.Type() {
	case types.LegacyTxType:
		return types.NewTx(&types.LegacyTx{
			Nonce:    tx.GetNonce(),
			GasPrice: tx.GetPrice().ToBig(),
			Gas:      tx.GetGas(),
			To:       to,
			Value:    tx.GetValue().ToBig(),
			Data:     tx.GetData(),
			V:        V.ToBig(),
			R:        R.ToBig(),
			S:        S.ToBig(),
		})
	case types.AccessListTxType:
		return types.NewTx(&types.AccessListTx{
			ChainID:    tx.GetChainID().ToBig(),
			Nonce:      tx.GetNonce(),
			GasPrice:   tx.GetPrice().ToBig(),
			Gas:        tx.GetGas(),
			To:         to,
			Value:      tx.GetValue().ToBig(),
			Data:       tx.GetData(),
			AccessList: fromErigonAccessList(tx.GetAccessList()),
			V:          V.ToBig(),
			R:          R.ToBig(),
			S:          S.ToBig(),
		})
	case types.DynamicFeeTxType:
		return types.NewTx(&types.DynamicFeeTx{
			ChainID:    tx.GetChainID().ToBig(),
			Nonce:      tx.GetNonce(),
			GasTipCap:  tx.GetTip().ToBig(),
			GasFeeCap:  tx.GetFeeCap().ToBig(),
			Gas:        tx.GetGas(),
			To:         to,
			Value:      tx.GetValue().ToBig(),
			Data:       tx.GetData(),
			AccessList: fromErigonAccessList(tx.GetAccessList()),
			V:          V.ToBig(),
			R:          R.ToBig(),
			S:          S.ToBig(),
		})
	default:
		panic("unsupported transaction type: " + string(tx.Type()))
	}
}

func fromErigonLog(log *erigonTypes.Log) *types.Log {
	if log == nil {
		return nil
	}
	topics := make([]common.Hash, len(log.Topics))
	for i := 0; i < len(topics); i++ {
		topics[i] = fromErigonHash(log.Topics[i])
	}
	return &types.Log{
		Address: fromErigonAddress(log.Address),
		Topics:  topics,
		Data:    log.Data,

		BlockNumber: log.BlockNumber,
		TxHash:      fromErigonHash(log.TxHash),
		TxIndex:     log.TxIndex,
		BlockHash:   fromErigonHash(log.BlockHash),
		Index:       log.Index,

		Removed: log.Removed,
	}
}

func fromErigonReceipt(receipt *erigonTypes.Receipt) *types.Receipt {
	if receipt == nil {
		return nil
	}
	logs := make([]*types.Log, len(receipt.Logs))
	for i := 0; i < len(logs); i++ {
		logs[i] = fromErigonLog(receipt.Logs[i])
	}
	return &types.Receipt{
		Type:              receipt.Type,
		PostState:         receipt.PostState,
		Status:            receipt.Status,
		CumulativeGasUsed: receipt.CumulativeGasUsed,
		Bloom:             types.Bloom(receipt.Bloom),
		Logs:              logs,

		TxHash:          fromErigonHash(receipt.TxHash),
		ContractAddress: fromErigonAddress(receipt.ContractAddress),
		GasUsed:         receipt.GasUsed,

		BlockHash:        fromErigonHash(receipt.BlockHash),
		BlockNumber:      receipt.BlockNumber,
		TransactionIndex: receipt.TransactionIndex,
	}
}

func fromErigonHeader(header *erigonTypes.Header) *types.Header {
	if header == nil {
		return nil
	}
	return &types.Header{
		ParentHash:  fromErigonHash(header.ParentHash),
		UncleHash:   fromErigonHash(header.UncleHash),
		Coinbase:    fromErigonAddress(header.Coinbase),
		Root:        fromErigonHash(header.Root),
		TxHash:      fromErigonHash(header.TxHash),
		ReceiptHash: fromErigonHash(header.ReceiptHash),
		Bloom:       types.Bloom(header.Bloom),
		Difficulty:  header.Difficulty,
		Number:      header.Number,
		GasLimit:    header.GasLimit,
		GasUsed:     header.GasUsed,
		Time:        header.Time,
		Extra:       header.Extra,
		MixDigest:   fromErigonHash(header.MixDigest),
		Nonce:       types.BlockNonce(header.Nonce),

		BaseFee: header.BaseFee,
	}
}

func fromErigonBlock(block *erigonTypes.Block) *types.Block {
	if block == nil {
		return nil
	}
	txs := make([]*types.Transaction, block.Transactions().Len())
	for i := 0; i < len(txs); i++ {
		txs[i] = fromErigonTransaction(block.Transactions()[i])
	}
	uncles := make([]*types.Header, len(block.Uncles()))
	for i := 0; i < len(uncles); i++ {
		uncles[i] = fromErigonHeader(block.Uncles()[i])
	}
	b := types.NewBlockWithHeader(fromErigonHeader(block.Header()))
	return b.WithBody(txs, uncles)
}
