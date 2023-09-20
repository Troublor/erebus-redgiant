package troubeth_test

import (
	"context"

	. "github.com/Troublor/erebus/troubeth"
	"github.com/ethereum/go-ethereum/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

var _ = Describe("TroubEth", func() {
	var troubEth *TroubEth

	BeforeEach(func() {
		var err error
		troubEth, err = NewTroubEth(viper.GetString("troubeth.url"))
		Expect(err).Should(BeNil())
	})

	It("should return error when troubeth service is not available", func() {
		_, err := NewTroubEth("http://localhost:1")
		Expect(err).To(MatchError(ContainSubstring("not connect to")))
	})

	It("should get abi when it is not a contract", func() {
		c := common.HexToAddress("0x7cB57B5A97eAbe94205C07890BE4c1aD31E486A8")
		abi, err := troubEth.GetAbi(context.Background(), c)
		Expect(abi).To(BeNil())
		Expect(err).To(BeNil())
	})

	It("should get abi when it is a non-verified contract", func() {
		c := common.HexToAddress("0x5bD25d2f4f26Bc82A34dE016D34612A28A0Cd492")
		abi, err := troubEth.GetAbi(context.Background(), c)
		Expect(abi).To(BeNil())
		Expect(err).To(BeNil())
	})

	It("should get abi when it is a verified contract", func() {
		c := common.HexToAddress("0xE592427A0AEce92De3Edee1F18E0157C05861564")
		abi, err := troubEth.GetAbi(context.Background(), c)
		Expect(err).To(BeNil())
		Expect(abi).To(Not(BeNil()))
	})
})
