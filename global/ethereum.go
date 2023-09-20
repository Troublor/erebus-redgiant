package global

import (
	"github.com/Troublor/erebus-redgiant/chain"
	"github.com/Troublor/erebus-redgiant/chain/readers"
	"github.com/Troublor/erebus-redgiant/config"
	"github.com/Troublor/erebus/troubeth"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	erigonDB      chain.BlockchainReader
	erigonRpc     chain.BlockchainReader
	chainReader   chain.BlockchainReader
	troubEth      *troubeth.TroubEth
	jsonRpcClient *rpc.Client
)

func JsonRpcClient() *rpc.Client {
	if jsonRpcClient != nil {
		return jsonRpcClient
	}

	var err error
	jsonRpcClient, err = rpc.DialContext(Ctx(), viper.GetString(config.CEthApiUrl.Key))
	if err != nil {
		log.Warn().Err(err).Msg("JsonRpcClient not available")
		jsonRpcClient = nil
	}
	return jsonRpcClient
}

func BlockchainReader() chain.BlockchainReader {
	if chainReader != nil {
		return chainReader
	}

	var err error

	datadir := viper.GetString(config.CErigonDataDir.Key)
	if datadir != "" {
		erigonDB, err = readers.NewErigonDBReader(Ctx(), datadir)
		if err != nil {
			log.Warn().Err(err).Msg("failed to create erigon db reader")
			erigonDB = nil
		}
	}

	erigonRpcAddr := viper.GetString(config.CErigonRpc.Key)
	if erigonRpcAddr != "" {
		erigonRpc, err = readers.NewErigonRpcReader(Ctx(), erigonRpcAddr)
		if err != nil {
			log.Warn().Err(err).Msg("failed to create erigon rpc reader")
			erigonRpc = nil
		}
	}

	if erigonDB != nil {
		chainReader, err = readers.NewCachedBlockchainReader(erigonDB)
		if err != nil {
			log.Fatal().Err(err).Msg("Faild to create state reader using Erigon")
		}
		RegisterCleanupTask(func() {
			_ = chainReader.Close()
		})
	} else if erigonRpc != nil {
		chainReader, err = readers.NewCachedBlockchainReader(erigonRpc)
		if err != nil {
			log.Fatal().Err(err).Msg("Faild to create state reader using Erigon")
		}
		RegisterCleanupTask(func() {
			_ = chainReader.Close()
		})
	} else {
		log.Error().Msg("no erigon reader is available")
	}

	return chainReader
}

func TroubEth() *troubeth.TroubEth {
	if troubEth != nil {
		return troubEth
	}

	var err error
	troubEth, err = troubeth.NewTroubEth(viper.GetString(config.CTroubEthUrl.Key))
	if err != nil {
		log.Warn().Err(err).Msg("TroubEth not available")
	}
	return troubEth
}
