package readers_test

import (
	"context"
	"math/big"

	"github.com/Troublor/erebus-redgiant/chain/readers"

	"github.com/Troublor/erebus-redgiant/chain"
	"github.com/Troublor/erebus-redgiant/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

var _ = Describe("Erigon Reader", func() {
	It("should return error when rpc is not available", func() {
		_, err := readers.NewErigonRpcReader(context.Background(), "localhost:80")
		Expect(err).Error().ShouldNot(BeNil())
	})

	It("should return error when datadir is not available", func() {
		_, err := readers.NewErigonDBReader(context.Background(), "/tmp/erebus/non-exist")
		Expect(err).Error().ShouldNot(BeNil())
	})

	When("erigon is available", func() {
		var erigon chain.BlockchainReader
		var err error
		ctx := context.Background()

		specs := func() {
			It("should return ChainID", func() {
				chainId, err := erigon.ChainID(ctx)
				Expect(err).Should(BeNil())
				Expect(chainId).ShouldNot(BeNil())
			})

			It("should return the latest block", func() {
				blockNumber, err := erigon.BlockNumber(ctx)
				Expect(err).Should(BeNil())
				block, err := erigon.BlockByNumber(ctx, nil)
				Expect(err).Should(BeNil())
				Expect(block.Number().Uint64()).Should(Equal(blockNumber))
			})

			It("should return correct block data", func() {
				blockNumber := big.NewInt(13000000)
				block, err := erigon.BlockByNumber(ctx, blockNumber)
				Expect(err).Should(BeNil())
				Expect(block.Number().Uint64()).Should(Equal(blockNumber.Uint64()))
				Expect(block.Transactions().Len()).Should(Equal(17))
				Expect(block.GasLimit()).Should(Equal(uint64(30_000_000)))
			})
		}

		When("erigon is running in rpc mode", Ordered, func() {
			BeforeAll(func() {
				erigon, err = readers.NewErigonRpcReader(context.Background(), viper.GetString(config.CErigonRpc.Key))
				if err != nil {
					Skip("erigon rpc is not available")
				}
			})
			AfterAll(func() {
				_ = erigon.Close()
			})
			specs()
		})

		When("erigon is running in db mode", Ordered, func() {
			BeforeAll(func() {
				erigon, err = readers.NewErigonDBReader(
					context.Background(), viper.GetString(config.CErigonDataDir.Key))
				if err != nil {
					Skip("erigon db is not available")
				}
			})
			AfterAll(func() {
				_ = erigon.Close()
			})
			specs()
		})
	})
})
