package storage_address

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/Troublor/erebus-redgiant/analysis/data_flow"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

type AddressingPath []*StorageAddressingNode

func (p AddressingPath) validate() {
	if len(p) < 2 {
		panic("AddressingPath must have at least 2 nodes")
	}
	if !p[0].Operation().OpCode().IsPush() {
		panic("First node must be a push")
	}
	lastOp := p[len(p)-1].Operation().OpCode()
	if lastOp != vm.SLOAD && lastOp != vm.SSTORE {
		panic("Last node must be a SLOAD or SSTORE")
	}
}

func (p AddressingPath) String() string {
	builder := strings.Builder{}
	for i, node := range p {
		if i == 0 {
			builder.WriteString("Seed")
		} else {
			builder.WriteString(fmt.Sprintf(" --%s-> %s", node.Operand.Hash(), node.Operation().OpCode()))
			if node.Ingredient != nil {
				builder.WriteString(fmt.Sprintf(":%s", node.Ingredient.Hash()))
			}
		}
	}
	return builder.String()
}

func (p AddressingPath) Seed() common.Hash {
	p.validate()
	return p[1].Operand.Hash()
}

func (p AddressingPath) Address() common.Hash {
	p.validate()
	return p[len(p)-1].Operand.Hash()
}

func (p AddressingPath) Opcode() vm.OpCode {
	p.validate()
	return p[len(p)-1].Operation().OpCode()
}

func (p AddressingPath) Copy() AddressingPath {
	newPath := make(AddressingPath, len(p))
	copy(newPath, p)
	return newPath
}

func (p AddressingPath) Reverse() AddressingPath {
	l := len(p)
	for i := 0; i < l/2; i++ {
		p[i], p[l-i-1] = p[l-i-1], p[i]
	}
	return p
}

// Equal returns true if the two paths have the same opcode, storage slot and addressing seed.
func (p AddressingPath) Equal(another AddressingPath) bool {
	return p.Opcode() == another.Opcode() && p.Address() == another.Address() && p.Seed() == another.Seed()
}

type StorageAddressingNode struct {
	operation *data_flow.Operation

	Operand    data_flow.Operand
	Ingredient *data_flow.Operand

	from data_flow.FlowNodeList
	to   data_flow.FlowNodeList
}

func (f StorageAddressingNode) MarshalBSON() ([]byte, error) {
	var cp = struct {
		CodeAddr   string  `bson:"codeAddr"`
		PC         string  `bson:"pc"`
		Opcode     string  `bson:"op"`
		Operand    *string `bson:"operand,omitempty"`
		Ingredient *string `bson:"ingredient,omitempty"`
	}{
		CodeAddr: f.operation.MsgCall().CodeAddr.Hex(),
		PC:       fmt.Sprintf("%d", f.operation.PC()),
		Opcode:   f.operation.OpCode().String(),
	}
	if !f.Operand.Nil() {
		temp := f.Operand.Hash().Hex()
		cp.Operand = &temp
	}
	if f.Ingredient != nil {
		temp := f.Ingredient.Hash().Hex()
		cp.Ingredient = &temp
	}
	return bson.Marshal(cp)
}

func (f *StorageAddressingNode) UnmarshalBSON(data []byte) error {
	var cp struct {
		CodeAddr   string  `bson:"codeAddr"`
		PC         string  `bson:"pc"`
		Opcode     string  `bson:"op"`
		Operand    *string `bson:"operand,omitempty"`
		Ingredient *string `bson:"ingredient,omitempty"`
	}
	if cp.Operand != nil {
		v, err := uint256.FromHex(*cp.Operand)
		if err != nil {
			return err
		}
		f.Operand = data_flow.Operand{Int: v}
	} else {
		f.Operand = data_flow.Operand{Int: nil}
	}
	if cp.Ingredient != nil {
		v, err := uint256.FromHex(*cp.Ingredient)
		if err != nil {
			return err
		}
		f.Ingredient = &data_flow.Operand{Int: v}
	}
	return nil
}

func (f *StorageAddressingNode) copy() *StorageAddressingNode {
	return &StorageAddressingNode{
		operation: f.operation,

		Operand:    f.Operand,
		Ingredient: f.Ingredient,

		from: f.from,
		to:   f.to,
	}
}

func (f *StorageAddressingNode) ID() data_flow.NodeID {
	return data_flow.NodeID(f.operation.ID())
}

func (f *StorageAddressingNode) Operation() *data_flow.Operation {
	return f.operation
}

func (f *StorageAddressingNode) From() data_flow.FlowNodeList {
	return f.from
}

func (f *StorageAddressingNode) To() data_flow.FlowNodeList {
	return f.to
}

func (f *StorageAddressingNode) AddUpstream(node ...data_flow.FlowNode) {
	f.from = append(f.from, node...)
}

func (f *StorageAddressingNode) AddressingPaths() []AddressingPath {
	var paths []AddressingPath

	var iter func(node *StorageAddressingNode, path AddressingPath)
	iter = func(node *StorageAddressingNode, path AddressingPath) {
		if node.Operation().OpCode().IsPush() {
			path = append(path, node)
			paths = append(paths, path.Reverse())
		}
		switch node.Operation().OpCode() {
		case vm.ADD:
			{
				ingredient := node.Operation().Arg(1)
				node.Operand = node.Operation().Arg(0)
				node.Ingredient = &ingredient
				upstream := node.From().Back(0)
				if upstream != nil {
					// the upstream is tainted
					cp := make(AddressingPath, len(path))
					copy(cp, path)
					cp = append(cp, node)
					iter(upstream.(*StorageAddressingNode), cp)
				}
			}
			{
				sibling := node.copy()
				ingredient := node.Operation().Arg(0)
				sibling.Operand = node.Operation().Arg(1)
				sibling.Ingredient = &ingredient
				upstream := node.From().Back(1)
				if upstream != nil {
					cp := make(AddressingPath, len(path))
					copy(cp, path)
					cp = append(cp, sibling)
					iter(upstream.(*StorageAddressingNode), cp)
				}
			}
		case vm.KECCAK256:
			for _, n := range node.From() {
				if n == nil {
					continue
				}
				cp := make(AddressingPath, len(path))
				copy(cp, path)
				cp = append(cp, node)
				iter(n.(*StorageAddressingNode), cp)
			}
		case vm.SLOAD, vm.SSTORE:
			for _, n := range node.From() {
				if n == nil {
					continue
				}
				cp := make(AddressingPath, len(path))
				copy(cp, path)
				cp = append(cp, node)
				iter(n.(*StorageAddressingNode), cp)
			}
		}
	}

	switch f.operation.OpCode() {
	case vm.SLOAD, vm.SSTORE:
		iter(f, make(AddressingPath, 0))
	default:
		panic("unsupported opcode")
	}
	return paths
}
