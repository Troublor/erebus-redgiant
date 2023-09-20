package dataset_test

import (
	"bufio"
	"fmt"
	"math/big"
	"testing"

	"github.com/Troublor/erebus-redgiant/global"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDataset(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dataset Suite")
}

var _ = BeforeSuite(func() {
	chainID, err := global.BlockchainReader().ChainID(global.Ctx())
	if err != nil {
		Skip(fmt.Sprintf("Failed to get chainID: %v", err))
	} else if chainID.Cmp(big.NewInt(1)) != 0 {
		Skip(fmt.Sprintf("ChainID is %v, not 1. This test suite is only meant for mainnet", chainID))
	}
})

func ReadLine(reader *bufio.Reader) (line []byte, err error) {
	line = make([]byte, 0, 1024)
	var data []byte
	var isPrefix bool
	isPrefix = true
	for isPrefix {
		data, isPrefix, err = reader.ReadLine()
		line = append(line, data...)
	}
	return line, err
}
