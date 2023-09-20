package helpers

import (
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

type stackEffect struct {
	pops   int
	pushes int
}

var opcodeStackEffects [256]stackEffect

func GetStackEffects(op vm.OpCode) (pops int, pushes int) {
	stackEffect := opcodeStackEffects[op]
	return stackEffect.pops, stackEffect.pushes
}

func init() {
	instructionSet := InstructionSet()
	for op, execution := range instructionSet {
		var pops int
		var pushes int
		if execution == nil {
			pops, pushes = 0, 0
		} else if op >= vm.DUP1 && op <= vm.DUP16 {
			pops, pushes = 0, 1
		} else if op >= vm.SWAP1 && op <= vm.SWAP16 {
			pops, pushes = 0, 0
		} else {
			minStack := GetUnexportedField(execution, "minStack").(int)
			maxStack := GetUnexportedField(execution, "maxStack").(int)
			pops = minStack
			pushes = int(params.StackLimit) + pops - maxStack
		}
		opcodeStackEffects[op] = stackEffect{pops, pushes}
	}
}
