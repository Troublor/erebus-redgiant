package config

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// Deprecated.
var LatestSigner types.Signer

// Deprecated.
var ChainConfig *params.ChainConfig

func init() {
	ChainConfig = params.MainnetChainConfig
	LatestSigner = types.NewLondonSigner(ChainConfig.ChainID)
}

// Deprecated.
func UpdateSigner(blockNumber *big.Int) {
	LatestSigner = types.MakeSigner(ChainConfig, blockNumber)
}
