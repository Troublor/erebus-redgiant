package tracers

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
)

type ProgramLocation interface {
	ID() string
	Equal(ProgramLocation) bool
	CodeAddr() common.Address
	PC() uint64
	OpCode() vm.OpCode
}

type TraceLocation interface {
	ProgramLocation
	GasAvailable() uint64
	GasUsed() uint64
	MsgCallPosition() CallPosition
	Index() uint
}

type location struct {
	pos          CallPosition
	codeAddr     common.Address
	pc           uint64
	opcode       vm.OpCode
	gasAvailable uint64 // the remaining gas before execute this location
	gasUsed      uint64
	index        uint
}

func (l location) ID() string {
	return fmt.Sprintf("(%s):%d-%d", l.pos.String(), l.pc, l.index)
}

func (l location) MsgCallPosition() CallPosition {
	return l.pos
}

func (l location) CodeAddr() common.Address {
	return l.codeAddr
}

func (l location) GasAvailable() uint64 {
	return l.gasAvailable
}

func (l location) GasUsed() uint64 {
	return l.gasUsed
}

func (l location) OpCode() vm.OpCode {
	return l.opcode
}

func (l location) PC() uint64 {
	return l.pc
}

func (l location) Equal(other ProgramLocation) bool {
	if ol, ok := other.(TraceLocation); ok {
		return l.index == ol.Index()
	}
	return false
}

func (l location) Index() uint {
	return l.index
}

type StructLogLocation struct {
	logger.StructLog

	index    uint
	codeAddr common.Address
	pos      CallPosition
}

func (l StructLogLocation) ID() string {
	return fmt.Sprintf("(%s):%d-%d", l.pos.String(), l.PC(), l.index)
}

func (l StructLogLocation) CodeAddr() common.Address {
	return l.codeAddr
}

func (l StructLogLocation) PC() uint64 {
	return l.StructLog.Pc
}

func (l StructLogLocation) OpCode() vm.OpCode {
	return l.StructLog.Op
}

func (l StructLogLocation) GasAvailable() uint64 {
	return l.StructLog.Gas
}

func (l StructLogLocation) GasUsed() uint64 {
	return l.StructLog.GasCost
}

func (l StructLogLocation) MsgCallPosition() CallPosition {
	return l.pos
}

func (l StructLogLocation) Index() uint {
	return l.index
}

func (l StructLogLocation) Equal(other ProgramLocation) bool {
	if ol, ok := other.(StructLogLocation); ok {
		return l.index == ol.index
	}
	return false
}

type IBasicBlock[L ProgramLocation] interface {
	Content() []L
	StateAddr() common.Address
	CodeAddr() common.Address
	Previous() IBasicBlock[L]
	Head() L
	Tail() L
	Next() IBasicBlock[L]
	Equal(IBasicBlock[L]) bool
}

type ITraceBlock = IBasicBlock[TraceLocation]

type traceBlock struct {
	content   []TraceLocation
	stateAddr common.Address
	codeAddr  common.Address
	previous  *traceBlock
	next      *traceBlock
}

func (b *traceBlock) Content() []TraceLocation {
	return b.content
}

func (b *traceBlock) StateAddr() common.Address {
	return b.stateAddr
}

func (b *traceBlock) CodeAddr() common.Address {
	return b.codeAddr
}

func (b *traceBlock) Previous() IBasicBlock[TraceLocation] {
	return b.previous
}

func (b *traceBlock) Equal(other IBasicBlock[TraceLocation]) bool {
	if ob, ok := other.(*traceBlock); ok {
		return b.Head().Equal(ob.Head())
	}
	return false
}

func (b *traceBlock) Head() TraceLocation {
	if len(b.content) > 0 {
		return b.content[0]
	} else {
		return nil
	}
}

func (b *traceBlock) Tail() TraceLocation {
	if len(b.content) > 0 {
		return b.content[len(b.content)-1]
	} else {
		return nil
	}
}

func (b *traceBlock) Next() IBasicBlock[TraceLocation] {
	return b.next
}

type Blocks[L ProgramLocation, B IBasicBlock[L]] []B

type Trace = Blocks[TraceLocation, ITraceBlock]

func (p Blocks[L, B]) SearchForLocation(codeAddr common.Address, pc uint64) L {
	for _, block := range p {
		if block.CodeAddr() != codeAddr {
			continue
		}
		if block.Head().PC() <= pc && pc <= block.Tail().PC() {
			for _, location := range block.Content() {
				if location.PC() == pc {
					return location
				}
			}
		}
	}
	var nilLocation L
	return nilLocation
}

func IsBasicBlockHead(previousLoc ProgramLocation, currentLoc ProgramLocation) bool {
	return previousLoc == nil || IsBasicBlockTail(previousLoc) || currentLoc.OpCode() == vm.JUMPDEST
}

func IsBasicBlockTail(currentLoc ProgramLocation) bool {
	currentOp := currentLoc.OpCode()
	return currentOp == vm.JUMP || currentOp == vm.JUMPI ||
		currentOp == vm.RETURN || currentOp == vm.STOP || currentOp == vm.INVALID || currentOp == vm.REVERT ||
		currentOp == vm.SELFDESTRUCT ||
		currentOp == vm.CREATE || currentOp == vm.CREATE2 ||
		currentOp == vm.CALL || currentOp == vm.CALLCODE || currentOp == vm.DELEGATECALL || currentOp == vm.STATICCALL
}
