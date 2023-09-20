package summary_test

import (
	"github.com/ethereum/go-ethereum/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Troublor/erebus-redgiant/analysis/summary"
	. "github.com/Troublor/erebus-redgiant/analysis/summary"
)

var MockAsset AssetType = "MOCK_ASSET"

type mockProfit struct {
	addr  common.Address
	value int
}

// Type returns the type of the profit.
func (p *mockProfit) Type() AssetType {
	return MockAsset
}

// Beneficiary returns the account address that get this profit.
func (p *mockProfit) Beneficiary() common.Address {
	return p.addr
}

// Copy returns a copy of the profit.
func (p *mockProfit) Copy() Profit {
	return &mockProfit{
		addr: p.addr,
	}
}

func (p *mockProfit) Merge(other Profit) (Profit, error) {
	if other.Beneficiary() != p.addr {
		return nil, ErrMergeNotPossible
	}
	return &mockProfit{
		addr:  p.addr,
		value: p.value + other.(*mockProfit).value,
	}, nil
}

func (p *mockProfit) Cmp(other summary.Profit) (int, error) {
	if other.Beneficiary() != p.addr {
		return 0, ErrNotComparable
	}
	if p.value < other.(*mockProfit).value {
		return -1, nil
	} else if p.value > other.(*mockProfit).value {
		return 1, nil
	}
	return 0, nil
}

// Zero returns true if the profit is effectively zero (no profit).
func (p *mockProfit) Zero() bool {
	return p.value == 0
}

// Positive returns true if the Beneficiary gains profit (this profit is positive).
func (p *mockProfit) Positive() bool {
	return p.value > 0
}

var _ = (*mockProfit)(nil)

var _ = Describe("Profits", func() {
	Context("Cmp", func() {
		var ps1 Profits
		BeforeEach(func() {
			ps1 = Profits{
				&mockProfit{addr: common.HexToAddress("0x1"), value: 1},
				&mockProfit{addr: common.HexToAddress("0x2"), value: 2},
			}
		})
		Context("when two sets contains the types of profits", func() {
			It("should return -1 when all profits in ps1 <= ps2", func() {
				ps2 := Profits{
					&mockProfit{addr: common.HexToAddress("0x1"), value: 2},
					&mockProfit{addr: common.HexToAddress("0x2"), value: 2},
				}
				r, err := ps1.Cmp(ps2)
				Expect(err).To(BeNil())
				Expect(r).To(Equal(-1))
			})
			It("should return -1 when all profits in ps1 < ps2", func() {
				ps2 := Profits{
					&mockProfit{addr: common.HexToAddress("0x1"), value: 2},
					&mockProfit{addr: common.HexToAddress("0x2"), value: 3},
				}
				r, err := ps1.Cmp(ps2)
				Expect(err).To(BeNil())
				Expect(r).To(Equal(-1))
			})
			It("should return 0 when all profits in ps1 = ps2", func() {
				ps2 := Profits{
					&mockProfit{addr: common.HexToAddress("0x1"), value: 1},
					&mockProfit{addr: common.HexToAddress("0x2"), value: 2},
				}
				r, err := ps1.Cmp(ps2)
				Expect(err).To(BeNil())
				Expect(r).To(Equal(0))
			})
			It("should return 1 when all profits in ps1 > ps2", func() {
				ps2 := Profits{
					&mockProfit{addr: common.HexToAddress("0x1"), value: 0},
					&mockProfit{addr: common.HexToAddress("0x2"), value: 1},
				}
				r, err := ps1.Cmp(ps2)
				Expect(err).To(BeNil())
				Expect(r).To(Equal(1))
			})
			It("should return 1 when all profits in ps1 >= ps2", func() {
				ps2 := Profits{
					&mockProfit{addr: common.HexToAddress("0x1"), value: 1},
					&mockProfit{addr: common.HexToAddress("0x2"), value: 1},
				}
				r, err := ps1.Cmp(ps2)
				Expect(err).To(BeNil())
				Expect(r).To(Equal(1))
			})
			It("should return ErrNotComparable when not all profits in ps1 > ps2", func() {
				ps2 := Profits{
					&mockProfit{addr: common.HexToAddress("0x1"), value: 2},
					&mockProfit{addr: common.HexToAddress("0x2"), value: 1},
				}
				r, err := ps1.Cmp(ps2)
				Expect(err).To(Equal(ErrNotComparable))
				Expect(r).To(Equal(0))
			})
		})

		Context("when one set include the other", func() {
			It("should return -1 when all profits in ps1 <= ps2", func() {
				ps2 := Profits{
					&mockProfit{addr: common.HexToAddress("0x1"), value: 2},
					&mockProfit{addr: common.HexToAddress("0x2"), value: 2},
					&mockProfit{addr: common.HexToAddress("0x3"), value: 2},
				}
				r, err := ps1.Cmp(ps2)
				Expect(err).To(BeNil())
				Expect(r).To(Equal(-1))
			})
			It("should return 1 when all profits in ps1 > ps2", func() {
				ps2 := Profits{
					&mockProfit{addr: common.HexToAddress("0x1"), value: 1},
					&mockProfit{addr: common.HexToAddress("0x2"), value: 1},
					&mockProfit{addr: common.HexToAddress("0x3"), value: -1},
				}
				r, err := ps1.Cmp(ps2)
				Expect(err).To(BeNil())
				Expect(r).To(Equal(1))
			})
			It("should return ErrNotComparable when not all profits in ps1 > ps2", func() {
				ps2 := Profits{
					&mockProfit{addr: common.HexToAddress("0x1"), value: 1},
					&mockProfit{addr: common.HexToAddress("0x2"), value: 1},
					&mockProfit{addr: common.HexToAddress("0x3"), value: 1},
				}
				r, err := ps1.Cmp(ps2)
				Expect(err).To(Equal(ErrNotComparable))
				Expect(r).To(Equal(0))
			})
		})

		Context("when two sets have overlap", func() {
			It("should return 1 when all profits in ps1 >= ps2", func() {
				ps2 := Profits{
					&mockProfit{addr: common.HexToAddress("0x2"), value: 2},
					&mockProfit{addr: common.HexToAddress("0x3"), value: -1},
				}
				r, err := ps1.Cmp(ps2)
				Expect(err).To(BeNil())
				Expect(r).To(Equal(1))
			})
			It("should return ErrNotComparable when not all profits in ps1 >= ps2", func() {
				ps2 := Profits{
					&mockProfit{addr: common.HexToAddress("0x2"), value: 2},
					&mockProfit{addr: common.HexToAddress("0x3"), value: 1},
				}
				r, err := ps1.Cmp(ps2)
				Expect(err).To(Equal(ErrNotComparable))
				Expect(r).To(Equal(0))
			})
		})

		Context("when one set is empty", func() {
			It("should return 1 when all profits in non-empty set is positive", func() {
				r, err := ps1.Cmp(nil)
				Expect(err).To(BeNil())
				Expect(r).To(Equal(1))
			})
			It("should return ErrNotComparable when all profits in non-empty set is positive", func() {
				ps1 = Profits{
					&mockProfit{addr: common.HexToAddress("0x1"), value: 1},
					&mockProfit{addr: common.HexToAddress("0x2"), value: -1},
				}
				r, err := ps1.Cmp(nil)
				Expect(err).To(Equal(ErrNotComparable))
				Expect(r).To(Equal(0))
			})
		})
	})
})
