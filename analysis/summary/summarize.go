package summary

import (
	"context"
	_ "embed"

	"github.com/Troublor/erebus-redgiant/chain"
	engine "github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/Troublor/erebus-redgiant/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

// This file contains helper functions to summarize the effect of one transaction.

// SummarizeTx summarize the given tx on the given state and vmContext.
// This function provide convenient helper to summary tx with IncludeAllConfig.
// If you want to customize the config, use ExeVM directly with TxSummaryTracer.
func SummarizeTx(state engine.State, vmContext *engine.VMContext, tx *engine.Tx) (*CallSummary, error) {
	exeVM := &engine.ExeVM{
		Config: helpers.VMConfigOnMainnet(),
		Tracer: NewTxSummaryTracer(IncludeAllConfig),
	}
	exeVM.Config.CapGasToBlockLimit = true
	exeVM.Config.RegulateBaseFee = true
	exeVM.Config.BypassNonceAndSenderCheck = false

	_, _, err := exeVM.ApplyTx(state, tx, vmContext, false, true)
	if err != nil {
		return nil, err
	}
	return exeVM.Tracer.(*TxSummaryTracer).Summary, nil
}

// SummarizeTxByHash summarizes the transaction, which can be found on blockchain with txHash,
// on the its original state and vmContext on blockchain.
func SummarizeTxByHash(
	ctx context.Context, chainReader chain.BlockchainReader,
	txHash common.Hash,
) (*CallSummary, error) {
	var err error
	tx, err := helpers.TxFromHash(ctx, chainReader, txHash)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to retrieve transaction")
		return nil, err
	}
	receipt, err := chainReader.TransactionReceipt(ctx, txHash)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to retrieve transaction receipt")
		return nil, err
	}

	var state engine.State
	var vmContext *engine.VMContext

	state, vmContext, err =
		helpers.PrepareStateAndContext(ctx, chainReader, receipt.BlockNumber, receipt.TransactionIndex)
	if err != nil {
		return nil, err
	}

	exeVM := &engine.ExeVM{
		Config: helpers.VMConfigOnMainnet(),
		Tracer: NewTxSummaryTracer(IncludeAllConfig),
	}
	_, _, err = exeVM.ApplyTx(state, tx, vmContext, false, true)
	if err != nil {
		return nil, err
	}
	return exeVM.Tracer.(*TxSummaryTracer).Summary, nil
}
