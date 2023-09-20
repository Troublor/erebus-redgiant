package state_test

import (
	"math/big"

	chain "github.com/Troublor/erebus-redgiant/chain/mocks"
	"github.com/Troublor/erebus-redgiant/dyengine/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ForkedState", func() {
	var mockCtrl *gomock.Controller
	var mockedChainReader *chain.MockBlockchainReader
	var forkNumber *big.Int

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockedChainReader = chain.NewMockBlockchainReader(mockCtrl)
		forkNumber = big.NewInt(100)
	})

	It("should use the latest block if forkNumber is not specified", func() {
		addr := common.HexToAddress("0x1")
		mockedChainReader.EXPECT().
			BlockNumber(gomock.Any()).
			Return(forkNumber.Uint64(), nil)
		mockedChainReader.EXPECT().
			BalanceAt(gomock.Any(), gomock.Eq(addr), forkNumber).
			Return(big.NewInt(999), nil)
		mockedChainReader.EXPECT().
			CodeAt(gomock.Any(), gomock.Eq(addr), forkNumber).
			Return([]byte{0x23, 0x33}, nil)
		mockedChainReader.EXPECT().
			NonceAt(gomock.Any(), gomock.Eq(addr), forkNumber).
			Return(uint64(9), nil)
		forkedState, err := state.NewForkedState(mockedChainReader, nil)
		Expect(err).To(BeNil())
		Expect(forkedState.GetBalance(addr)).To(Equal(big.NewInt(999)))
	})

})
