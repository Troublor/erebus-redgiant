package helpers

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
)

func BenchmarkGetStackEffects(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetStackEffects(vm.ADD)
	}
}
