package examples

import (
	"context"
	"fmt"
	"github.com/Troublor/erebus-redgiant/chain"
	"github.com/Troublor/erebus-redgiant/chain/readers"
	"github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/Troublor/erebus-redgiant/dyengine/tracers"
	"github.com/Troublor/erebus-redgiant/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"testing"
)

func ChainReader(ctx context.Context) chain.BlockchainReader {
	var (
		erigonDB    chain.BlockchainReader
		erigonRpc   chain.BlockchainReader
		chainReader chain.BlockchainReader
	)
	var err error

	// Build chain reader using erigon database (datadir)
	datadir := "erigon/datadir"
	erigonDB, err = readers.NewErigonDBReader(ctx, datadir)
	if err != nil {
		log.Warn().Err(err).Msg("failed to create erigon db reader")
		erigonDB = nil
	}

	// Alternatively, build chain reader using erigon rpc
	erigonRpcAddr := "localhost:9090"
	erigonRpc, err = readers.NewErigonRpcReader(ctx, erigonRpcAddr)
	if err != nil {
		log.Warn().Err(err).Msg("failed to create erigon rpc reader")
		erigonRpc = nil
	}

	if erigonDB != nil {
		chainReader, err = readers.NewCachedBlockchainReader(erigonDB)
		if err != nil {
			log.Fatal().Err(err).Msg("Faild to create state reader using Erigon")
		}
	} else if erigonRpc != nil {
		chainReader, err = readers.NewCachedBlockchainReader(erigonRpc)
		if err != nil {
			log.Fatal().Err(err).Msg("Faild to create state reader using Erigon")
		}
	} else {
		log.Error().Msg("no erigon reader is available")
	}
	return chainReader
}

func TestReproduceTx(t *testing.T) {
	ctx := context.Background()

	chainReader := ChainReader(ctx)

	txHash := common.HexToHash("0xbcab6b1ef2346cc5c3ff67c9f029c346b68cd2b07a20dcb1f2a0b68a30119eff")
	tx, _, err := chainReader.TransactionByHash(ctx, txHash)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get transaction")
	}
	receipt, err := chainReader.TransactionReceipt(ctx, txHash)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get transaction receipt")
	}

	// get the blockchain state before the execution of the transaction whose block number and transaction index are specified.
	state, vmContext, err := helpers.PrepareStateAndContext(
		ctx, chainReader, receipt.BlockNumber, receipt.TransactionIndex,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to prepare state and context")
	}

	// use a custom tracer to trace the execution of the transaction
	tracer := tracers.NewStructLogTracer(tracers.StructLogConfig{
		Limit: 0,
	})
	exeVM := dyengine.NewExeVM(helpers.VMConfigOnMainnet())
	exeVM.SetTracer(tracer)
	// execute the transaction
	executionResult, actualReceipt, err := exeVM.ApplyTransaction(state, tx, vmContext, false, true)
	fmt.Printf("execution result: %v\n", executionResult)
	fmt.Printf("actual receipt: %v\n", actualReceipt)
}
