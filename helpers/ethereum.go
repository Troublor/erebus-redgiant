package helpers

import (
	"context"
	"fmt"
	"hash"
	"math/big"
	"strings"
	"sync"

	"github.com/Troublor/erebus-redgiant/contract"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

func GetTransactions(
	client *ethclient.Client, blockNumber *big.Int, maxTransactions int,
) (types.Transactions, types.Receipts, error) {
	ctx := context.Background()
	block, err := client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return nil, nil, err
	}

	if maxTransactions < 0 || maxTransactions > len(block.Transactions()) {
		maxTransactions = len(block.Transactions())
	}
	txs := make(types.Transactions, maxTransactions)
	receipts := make(types.Receipts, maxTransactions)

	errCh := make(chan error, maxTransactions)
	defer close(errCh)

	var wg sync.WaitGroup
	for i := 0; i < maxTransactions; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			transaction, err := client.TransactionInBlock(ctx, block.Hash(), uint(index))
			if err != nil {
				errCh <- err
				return
			}
			txs[index] = transaction
			receipt, err := client.TransactionReceipt(ctx, transaction.Hash())
			if err != nil {
				errCh <- err
				return
			}
			receipts[index] = receipt
		}(i)
	}
	wg.Wait()
	if len(errCh) > 0 {
		err := <-errCh
		return nil, nil, err
	}

	return txs, receipts, nil
}

// keccakState wraps sha3.state. In addition to the usual hash methods, it also supports
// Read to get a variable amount of data from the hash state. Read is faster than Sum
// because it doesn't copy the internal state, but also modifies the internal state.
type keccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

func Keccak256OfInt(value *big.Int) common.Hash {
	v, overflow := uint256.FromBig(value)
	SanityCheck(func() bool {
		return !overflow
	}, "uint256 overflow")
	b := make([]byte, 32)
	v.WriteToSlice(b[:])
	return Keccak256OfBytes(b)
}

func Keccak256OfBytes(payload []byte) common.Hash {
	var h common.Hash
	keccak256 := sha3.NewLegacyKeccak256().(keccakState)
	_, err := keccak256.Write(payload)
	if err != nil {
		panic(err)
	}
	_, err = keccak256.Read(h[:])
	if err != nil {
		panic(err)
	}
	return h
}

func StringifyTxData(contractAbi *abi.ABI, data []byte) string {
	m, arg, err := contract.UnpackInput(*contractAbi, data)
	if err != nil {
		return common.Bytes2Hex(data)
	}
	if m.Type == abi.Fallback {
		// return fallback(hex)
		return fmt.Sprintf("fallback(%x)", arg)
	}
	if m.Type == abi.Receive {
		// return receive()
		return fmt.Sprintf("receive(%x)", arg)
	}
	args := arg
	argS := make([]string, len(args))
	for i, arg := range args {
		argS[i] = StringifyArgument(&m.Inputs[i].Type, arg)
	}

	// return fn(arg...)
	return fmt.Sprintf("%s(%s)", m.Name, strings.Join(argS, ","))
}

func StringifyArgument(t *abi.Type, v interface{}) string {
	switch t.T {
	case abi.IntTy, abi.UintTy, abi.BoolTy:
		return fmt.Sprint(v)
	case abi.StringTy:
		return fmt.Sprintf("'%s'", v)
	case abi.AddressTy:
		return v.(common.Address).Hex()
	case abi.HashTy:
		return v.(common.Hash).Hex()
	case abi.BytesTy, abi.FixedBytesTy:
		return common.Bytes2Hex(v.([]byte))
	case abi.SliceTy, abi.ArrayTy:
		elements := make([]string, 0)
		for _, elem := range v.([]interface{}) {
			elements = append(elements, StringifyArgument(t.Elem, elem))
		}
		return fmt.Sprintf("[%s]", strings.Join(elements, ","))
	default:
		panic(fmt.Sprint("unknown type ", t.T))
	}
}

func GetMemoryCopyWithPadding(memory *vm.Memory, offset, size int64) []byte {
	var memData = make([]byte, size)
	if (offset + size) <= int64(memory.Len()) {
		copy(memData, memory.GetPtr(offset, size))
	} else {
		copy(memData, memory.GetPtr(offset, offset+size-int64(memory.Len())))
	}
	return memData
}
