package contract

import (
	"bytes"
	_ "embed"
	"encoding/hex"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

//go:embed assets/erc20.abi.json
var ERC20AbiJSON string
var ERC20ABI abi.ABI

//go:embed assets/erc721.abi.json
var ERC721AbiJSON string
var ERC721ABI abi.ABI

//go:embed assets/erc777.abi.json
var ERC777AbiJSON string
var ERC777ABI abi.ABI

//go:embed assets/erc1155.abi.json
var ERC1155AbiJSON string
var ERC1155ABI abi.ABI

//go:embed assets/weth.abi.json
var WETH9AbiJSON string
var WETH9ABI abi.ABI
var WETH9Address common.Address = common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")

var ERC20TransferTopic common.Hash
var ERC777SentTopic common.Hash
var ERC777MintedTopic common.Hash
var ERC777BurnedTopic common.Hash
var ERC721TransferTopic common.Hash
var ERC1155TransferSingleTopic common.Hash
var ERC1155TransferBatchTopic common.Hash
var WETH9DepositTopic common.Hash
var WETH9WithdrawalTopic common.Hash

func init() {
	var err error
	ERC20ABI, err = abi.JSON(strings.NewReader(ERC20AbiJSON))
	if err != nil {
		panic(err)
	}
	ERC721ABI, err = abi.JSON(strings.NewReader(ERC721AbiJSON))
	if err != nil {
		panic(err)
	}
	ERC1155ABI, err = abi.JSON(strings.NewReader(ERC1155AbiJSON))
	if err != nil {
		panic(err)
	}
	ERC777ABI, err = abi.JSON(strings.NewReader(ERC777AbiJSON))
	if err != nil {
		panic(err)
	}
	WETH9ABI, err = abi.JSON(strings.NewReader(WETH9AbiJSON))
	if err != nil {
		panic(err)
	}
	ERC20TransferTopic = crypto.Keccak256Hash([]byte(ERC20ABI.Events["Transfer"].Sig))
	ERC777SentTopic = crypto.Keccak256Hash([]byte(ERC777ABI.Events["Sent"].Sig))
	ERC777MintedTopic = crypto.Keccak256Hash([]byte(ERC777ABI.Events["Minted"].Sig))
	ERC777BurnedTopic = crypto.Keccak256Hash([]byte(ERC777ABI.Events["Burned"].Sig))
	ERC721TransferTopic = crypto.Keccak256Hash([]byte(ERC721ABI.Events["Transfer"].Sig))
	ERC1155TransferSingleTopic = crypto.Keccak256Hash([]byte(ERC1155ABI.Events["TransferSingle"].Sig))
	ERC1155TransferBatchTopic = crypto.Keccak256Hash([]byte(ERC1155ABI.Events["TransferBatch"].Sig))
	WETH9DepositTopic = crypto.Keccak256Hash([]byte(WETH9ABI.Events["Deposit"].Sig))
	WETH9WithdrawalTopic = crypto.Keccak256Hash([]byte(WETH9ABI.Events["Withdrawal"].Sig))
}

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

func MustPack(contractAbi abi.ABI, name string, args ...interface{}) []byte {
	data, err := contractAbi.Pack(name, args...)
	if err != nil {
		panic(err)
	}
	return data
}
