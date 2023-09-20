package readers

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Troublor/erebus-redgiant/chain"

	"github.com/allegro/bigcache"

	"github.com/ethereum/go-ethereum/common"
)

// cachedBlockchainReader is a BlockchainReader that caches the results of
// the given BlockchainReader.
type cachedBlockchainReader struct {
	chain.BlockchainReader
	chainId *big.Int

	// key: blockNumber:address
	blockHashes *bigcache.BigCache
	balances    *bigcache.BigCache
	codes       *bigcache.BigCache
	nonces      *bigcache.BigCache
	storages    *bigcache.BigCache
}

func NewCachedBlockchainReader(source chain.BlockchainReader) (*cachedBlockchainReader, error) {
	chainId, err := source.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	cacheConfig := bigcache.DefaultConfig(10 * time.Minute)
	cacheConfig.Verbose = false
	blockHashes, err := bigcache.NewBigCache(cacheConfig)
	if err != nil {
		return nil, err
	}
	balances, err := bigcache.NewBigCache(cacheConfig)
	if err != nil {
		return nil, err
	}
	codes, err := bigcache.NewBigCache(cacheConfig)
	if err != nil {
		return nil, err
	}
	nonces, err := bigcache.NewBigCache(cacheConfig)
	if err != nil {
		return nil, err
	}
	storages, err := bigcache.NewBigCache(cacheConfig)
	if err != nil {
		return nil, err
	}

	return &cachedBlockchainReader{
		BlockchainReader: source,
		chainId:          chainId,

		blockHashes: blockHashes,
		balances:    balances,
		codes:       codes,
		nonces:      nonces,
		storages:    storages,
	}, nil
}

func (s *cachedBlockchainReader) BalanceAt(
	ctx context.Context, account common.Address, blockNumber *big.Int,
) (*big.Int, error) {
	key := fmt.Sprintf("%d:%s", blockNumber.Uint64(), account)
	if entry, err := s.balances.Get(key); err == nil {
		balance := new(big.Int).SetBytes(entry)
		return balance, nil
	} else {
		balance, err := s.BlockchainReader.BalanceAt(ctx, account, blockNumber)
		if err != nil {
			return nil, err
		}
		err = s.balances.Set(key, balance.Bytes())
		if err != nil {
			return nil, err
		}
		return balance, nil
	}
}

func (s *cachedBlockchainReader) StorageAt(
	ctx context.Context, account common.Address, slot common.Hash, blockNumber *big.Int,
) ([]byte, error) {
	key := fmt.Sprintf("%d:%s:%s", blockNumber.Uint64(), account, slot)
	if entry, err := s.storages.Get(key); err == nil {
		return entry, nil
	} else {
		value, err := s.BlockchainReader.StorageAt(ctx, account, slot, blockNumber)
		if err != nil {
			return nil, err
		}
		err = s.storages.Set(key, value)
		if err != nil {
			return nil, err
		}
		return value, nil
	}
}

func (s *cachedBlockchainReader) CodeAt(
	ctx context.Context, account common.Address, blockNumber *big.Int,
) ([]byte, error) {
	key := fmt.Sprintf("%d:%s", blockNumber.Uint64(), account)
	if entry, err := s.codes.Get(key); err == nil {
		return entry, nil
	} else {
		code, err := s.BlockchainReader.CodeAt(ctx, account, blockNumber)
		if err != nil {
			return nil, err
		}
		err = s.codes.Set(key, code)
		if err != nil {
			return nil, err
		}
		return code, nil
	}
}

func (s *cachedBlockchainReader) NonceAt(
	ctx context.Context, account common.Address, blockNumber *big.Int,
) (uint64, error) {
	key := fmt.Sprintf("%d:%s", blockNumber.Uint64(), account)
	if entry, err := s.nonces.Get(key); err == nil {
		value := new(big.Int).SetBytes(entry)
		return value.Uint64(), nil
	} else {
		nonce, err := s.BlockchainReader.NonceAt(ctx, account, blockNumber)
		if err != nil {
			return 0, err
		}
		value := new(big.Int).SetUint64(nonce)
		err = s.nonces.Set(key, value.Bytes())
		if err != nil {
			return 0, err
		}
		return nonce, nil
	}
}

func (s *cachedBlockchainReader) BlockHashByNumber(
	ctx context.Context, blockNumber *big.Int,
) (common.Hash, error) {
	key := fmt.Sprintf("%d", blockNumber.Uint64())
	if entry, err := s.blockHashes.Get(key); err == nil {
		hash := common.BytesToHash(entry)
		return hash, nil
	} else {
		hash, err := s.BlockchainReader.BlockHashByNumber(ctx, blockNumber)
		if err != nil {
			return common.Hash{}, err
		}
		err = s.blockHashes.Set(key, hash.Bytes())
		if err != nil {
			return [32]byte{}, err
		}
		return hash, nil
	}
}
