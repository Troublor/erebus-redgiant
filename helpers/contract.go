package helpers

import (
	"bytes"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"io/ioutil"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func LoadBinaryFromFile(file string) []byte {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	binData := make([]byte, hex.DecodedLen(len(data)))
	l, err := hex.Decode(binData, bytes.TrimSpace(data))
	if err != nil || l != len(binData) {
		panic(err)
	}
	return binData
}

func LoadAbiFromFile(file string) abi.ABI {
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	contractAbi, err := abi.JSON(f)
	if err != nil {
		panic(err)
	}
	return contractAbi
}

func LoadAbiFromString(s string) abi.ABI {
	contractAbi, err := abi.JSON(strings.NewReader(s))
	if err != nil {
		panic(err)
	}
	return contractAbi
}

func IsPrecompiledContract(addr common.Address) bool {
	// we approximate the precompiled contracts by their address
	// a better way is to check the precompiled contract map for each hardfork, but that requires ChainConfig and block number
	return addr.Hash().Big().Cmp(big.NewInt(255)) <= 0
}
