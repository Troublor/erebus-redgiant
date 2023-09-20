package dyengine_test

import (
	"github.com/Troublor/erebus-redgiant/dyengine"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PseudoTx", func() {
	It("should generate different hash with different data", func() {
		tx1 := dyengine.NewPseudoTx([]byte("data1"), nil)
		tx2 := dyengine.NewPseudoTx([]byte("data2"), nil)
		Expect(tx1.Hash()).NotTo(Equal(tx2.Hash()))
	})

	It("should generate same hash with same data", func() {
		tx1 := dyengine.NewPseudoTx(
			[]byte("data"),
			func(s dyengine.State) ([]byte, error) { return []byte("data1"), nil },
		)
		tx2 := dyengine.NewPseudoTx(
			[]byte("data"),
			func(s dyengine.State) ([]byte, error) { return []byte("data2"), nil },
		)
		Expect(tx1.Hash()).To(Equal(tx2.Hash()))
	})
})
