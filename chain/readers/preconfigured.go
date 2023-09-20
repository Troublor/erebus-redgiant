package readers

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type preconfiguredBlockState struct {
	Number   *big.Int
	Balances map[common.Address]*big.Int
	Codes    map[common.Address][]byte
	Nonces   map[common.Address]uint64
	Storages map[common.Address]map[common.Hash]common.Hash
}

var UndefinedStateErr = errors.New("the state is not defined in pivot")

const (
	ErrWhenUndefined  = 0
	ZeroWhenUndefined = 1
)

// preconfiguredStatePivot is only meant to facilitate testing and debugging.
type preconfiguredStatePivot struct {
	chainId *big.Int
	// states preconfigured the blockchain state.
	// It is a mapping from blockHash to block data,
	// including block Number, Balances, Codes, Nonces, Storages
	states map[common.Hash]preconfiguredBlockState

	behaviorMode int
}

func NewPreconfiguredBlockchainReader(
	chainId *big.Int, states map[common.Hash]preconfiguredBlockState, mode int,
) *preconfiguredStatePivot {
	if mode != ErrWhenUndefined {
		mode = ZeroWhenUndefined
	}
	return &preconfiguredStatePivot{
		chainId:      chainId,
		states:       states,
		behaviorMode: mode,
	}
}

func (p *preconfiguredStatePivot) BalanceAt(
	_ context.Context, account common.Address, blockNumber *big.Int,
) (*big.Int, error) {
	for _, b := range p.states {
		if b.Number.Cmp(blockNumber) == 0 {
			if bal, exist := b.Balances[account]; !exist {
				goto undefined
			} else {
				return bal, nil
			}
		}
	}
undefined:
	if p.behaviorMode == ZeroWhenUndefined {
		return big.NewInt(0), nil
	} else {
		return nil, UndefinedStateErr
	}
}

func (p *preconfiguredStatePivot) StorageAt(
	_ context.Context, account common.Address, key common.Hash, blockNumber *big.Int,
) ([]byte, error) {
	for _, b := range p.states {
		if b.Number.Cmp(blockNumber) == 0 {
			if accStorage, ok := b.Storages[account]; ok {
				if value, ok := accStorage[key]; ok {
					return value.Bytes(), nil
				}
			}
			goto undefined
		}
	}
undefined:
	if p.behaviorMode == ZeroWhenUndefined {
		return common.Hash{}.Bytes(), nil
	} else {
		return nil, UndefinedStateErr
	}
}

func (p *preconfiguredStatePivot) CodeAt(
	_ context.Context, account common.Address, blockNumber *big.Int,
) ([]byte, error) {
	for _, b := range p.states {
		if b.Number.Cmp(blockNumber) == 0 {
			if code, ok := b.Codes[account]; ok {
				return code, nil
			}
			goto undefined
		}
	}
undefined:
	if p.behaviorMode == ZeroWhenUndefined {
		return []byte{}, nil
	} else {
		return nil, UndefinedStateErr
	}
}

func (p *preconfiguredStatePivot) NonceAt(
	_ context.Context, account common.Address, blockNumber *big.Int,
) (uint64, error) {
	for _, b := range p.states {
		if b.Number.Cmp(blockNumber) == 0 {
			if nonce, ok := b.Nonces[account]; ok {
				return nonce, nil
			}
			goto undefined
		}
	}
undefined:
	if p.behaviorMode == ZeroWhenUndefined {
		return 0, nil
	} else {
		return 0, UndefinedStateErr
	}
}

func (p *preconfiguredStatePivot) ChainID(_ context.Context) (*big.Int, error) {
	return p.chainId, nil
}

func (p *preconfiguredStatePivot) BlockNumber(_ context.Context) (uint64, error) {
	max := big.NewInt(0)
	for _, b := range p.states {
		if b.Number.Cmp(max) > 0 {
			max = b.Number
		}
	}
	return max.Uint64(), nil
}

func (p *preconfiguredStatePivot) BlockHashByNumber(
	_ context.Context, number *big.Int,
) (common.Hash, error) {
	for h, b := range p.states {
		if b.Number.Cmp(number) == 0 {
			return h, nil
		}
	}
	if p.behaviorMode == ZeroWhenUndefined {
		return common.Hash{}, nil
	} else {
		return common.Hash{}, UndefinedStateErr
	}
}

func (p *preconfiguredStatePivot) Close() {
}
