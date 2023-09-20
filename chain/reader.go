package chain

import (
	"context"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

//go:generate mockgen -destination=./mocks/reader.go -package=chain_mocks github.com/Troublor/erebus/redgiant/chain BlockchainReader

// BlockchainReader defines functions that read information from blockchain.
type BlockchainReader interface {
	io.Closer
	ethereum.ChainStateReader
	ethereum.TransactionReader

	// blockchain
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	BlockNumber(context.Context) (uint64, error)
	BlockHashByNumber(context.Context, *big.Int) (common.Hash, error)
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	TransactionCount(ctx context.Context, blockHash common.Hash) (uint, error)
	TransactionInBlock(ctx context.Context, blockHash common.Hash, index uint) (*types.Transaction, error)

	// misc
	ChainID(context.Context) (*big.Int, error)
}
