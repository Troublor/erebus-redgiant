package helpers

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/Troublor/erebus-redgiant/chain"
	"github.com/Troublor/erebus-redgiant/dyengine"
	"github.com/Troublor/erebus-redgiant/dyengine/state"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func NewBlockContext(
	coinbase common.Address,
	blockNumber *big.Int,
	difficulty *big.Int,
	time *big.Int,
	gasLimit uint64,
	baseFee *big.Int,
) vm.BlockContext {
	return vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    coinbase,
		BlockNumber: blockNumber,
		Time:        time,
		Difficulty:  difficulty,
		GasLimit:    gasLimit,
		GetHash:     nil, // to be filled by State.GetHashFn()
		BaseFee:     baseFee,
	}
}

func NewVMContext(
	coinbase common.Address,
	blockHash common.Hash,
	blockNumber *big.Int,
	blockDifficulty *big.Int,
	blockTime *big.Int,
	gasLimit uint64,
	baseFee *big.Int,
) *dyengine.VMContext {
	return &dyengine.VMContext{
		BlockContext:     NewBlockContext(coinbase, blockNumber, blockDifficulty, blockTime, gasLimit, baseFee),
		GasPool:          new(core.GasPool).AddGas(gasLimit),
		BlockHash:        blockHash,
		TransactionIndex: 0,
	}
}

func VMConfigOnMainnet() dyengine.VMConfig {
	return dyengine.VMConfig{
		ChainConfig: params.MainnetChainConfig,
	}
}

func VMContextFromBlock(block *types.Block) *dyengine.VMContext {
	vmContext := NewVMContext(
		block.Coinbase(),
		block.Hash(),
		block.Number(),
		block.Difficulty(),
		big.NewInt(int64(block.Time())),
		block.GasLimit(),
		block.BaseFee(),
	)
	return vmContext
}

// DebuggingVMContext returns a VMContext in which all data is zero.
// This is typically used together with DebuggingCall.
func DebuggingVMContext() *dyengine.VMContext {
	return NewVMContext(
		common.HexToAddress("0"),
		common.HexToHash("0"),
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(time.Now().Unix()),
		math.MaxUint64,
		big.NewInt(0),
	)
}

func NewDebuggingExeVM() *dyengine.ExeVM {
	return &dyengine.ExeVM{
		Config: dyengine.VMConfig{
			ChainConfig: &params.ChainConfig{
				ChainID:             big.NewInt(1),
				HomesteadBlock:      big.NewInt(0),
				DAOForkBlock:        big.NewInt(0),
				DAOForkSupport:      true,
				EIP150Block:         big.NewInt(0),
				EIP150Hash:          common.HexToHash("0"),
				EIP155Block:         big.NewInt(0),
				EIP158Block:         big.NewInt(0),
				ByzantiumBlock:      big.NewInt(0),
				ConstantinopleBlock: big.NewInt(0),
				PetersburgBlock:     big.NewInt(0),
				IstanbulBlock:       big.NewInt(0),
				MuirGlacierBlock:    big.NewInt(0),
				BerlinBlock:         big.NewInt(0),
				LondonBlock:         big.NewInt(0),
				ArrowGlacierBlock:   big.NewInt(00000000),
				Ethash:              new(params.EthashConfig),
			},
		},
	}
}

// PrepareStateAndContext helps set up the State under which the transaction at blockNumber
// with index txIndex should execute.
// This is helpful when we reproduce transactions on chain.
func PrepareStateAndContext(
	ctx context.Context, chainReader chain.BlockchainReader, blockNumber *big.Int, txIndex uint,
) (dyengine.State, *dyengine.VMContext, error) {
	forkNumber := new(big.Int).Sub(blockNumber, big.NewInt(1))
	st, err := state.NewForkedState(chainReader, forkNumber)
	if err != nil {
		return nil, nil, err
	}

	// replay preceding transactions
	block, err := chainReader.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return nil, nil, err
	}
	if block == nil {
		return nil, nil, fmt.Errorf("block number %d does not exist", blockNumber)
	}

	vmContext := VMContextFromBlock(block)
	if txIndex <= 0 {
		// no preceding transactions to replay
		return st, vmContext, nil
	}

	exeVM := &dyengine.ExeVM{
		Config: VMConfigOnMainnet(),
	}
	_, _, _, err = exeVM.ApplyTransactions(st, block.Transactions()[:txIndex], vmContext, false, false)
	if err != nil {
		return nil, nil, err
	}
	return st, vmContext, nil
}
