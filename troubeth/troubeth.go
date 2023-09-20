package troubeth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/allegro/bigcache"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrNotContract    = errors.New("not a contract")
	ErrInvalidAddress = errors.New("invalid address")
)

type TroubEth struct {
	Url string

	// cache
	abiMap        map[common.Address]*abi.ABI
	sourceMap     map[common.Address][]byte
	verifiedCache *bigcache.BigCache
}

func NewTroubEth(url string) (*TroubEth, error) {
	cacheConfig := bigcache.DefaultConfig(12 * time.Hour)
	cacheConfig.Verbose = false
	verifiedCache, err := bigcache.NewBigCache(cacheConfig)
	if err != nil {
		return nil, err
	}
	t := &TroubEth{
		Url: url,

		// cache
		abiMap:        make(map[common.Address]*abi.ABI),
		sourceMap:     make(map[common.Address][]byte),
		verifiedCache: verifiedCache,
	}

	if !t.checkAlive() {
		return nil, fmt.Errorf("troubeth: cannot connect to %s", t.Url)
	}
	return t, nil
}

func (e *TroubEth) checkAlive() bool {
	_, err := http.Get(e.Url)
	return err == nil
}

func (e *TroubEth) GetAbi(_ context.Context, contract common.Address) (*abi.ABI, error) {
	if contractAbi, cached := e.abiMap[contract]; cached {
		return contractAbi, nil
	}

	resp, err := http.Get(fmt.Sprintf("%s/contract/%s/abi", e.Url, contract.Hex()))
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		var contractAbi abi.ABI
		err = json.NewDecoder(resp.Body).Decode(&contractAbi)
		if err != nil {
			return nil, err
		}
		e.abiMap[contract] = &contractAbi
		return &contractAbi, nil
	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusNotAcceptable:
		return nil, ErrNotContract
	case http.StatusBadRequest:
		fallthrough
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidAddress, resp.Body)
	}
}

func (e *TroubEth) GetSource(_ context.Context, contract common.Address) ([]byte, error) {
	if source, cached := e.sourceMap[contract]; cached {
		return source, nil
	}

	resp, err := http.Get(fmt.Sprintf("%s/contract/%s/source", e.Url, contract.Hex()))
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		source, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		e.sourceMap[contract] = source
		return source, nil
	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusNotAcceptable:
		return nil, ErrNotContract
	case http.StatusBadRequest:
		fallthrough
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidAddress, resp.Body)
	}
}

func (e *TroubEth) IsVerified(ctx context.Context, contract common.Address) (bool, error) {
	if verified, err := e.verifiedCache.Get(contract.Hex()); err == nil {
		return verified[0] == 1, nil
	} else {
		req, err := http.NewRequestWithContext(ctx,
			"GET", fmt.Sprintf("%s/contract/%s/name", e.Url, contract.Hex()),
			nil,
		)
		if err != nil {
			return false, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false, err
		}
		switch resp.StatusCode {
		case http.StatusOK:
			verified = []byte{byte(1)}
			err := e.verifiedCache.Set(contract.Hex(), verified)
			if err != nil {
				return false, err
			}
			return true, nil
		default:
			verified = []byte{byte(0)}
			err := e.verifiedCache.Set(contract.Hex(), verified)
			if err != nil {
				return false, err
			}
			return false, nil
		}
	}
}
