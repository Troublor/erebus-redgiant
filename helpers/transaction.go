package helpers

import (
	"context"

	"github.com/Troublor/erebus-redgiant/chain"
	"github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TxFromHash(ctx context.Context, chainReader chain.BlockchainReader, hash common.Hash) (*dyengine.Tx, error) {
	transaction, _, err := chainReader.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	receipt, err := chainReader.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}
	signer := types.MakeSigner(VMConfigOnMainnet().ChainConfig, receipt.BlockNumber)
	return dyengine.TxFromTransactionWithSigner(transaction, signer)
}
