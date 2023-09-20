package tracers

import (
	"fmt"
	"strings"
)

type ICallPosition interface {
	IsRoot() bool
	String() string
	Depth() int
	AncestorOf(descendant ICallPosition) bool
	ParentOf(child ICallPosition) bool
	Cmp(other ICallPosition) int
	Equal(other ICallPosition) bool
}

type CallPosition struct {
	position position

	// compactPosition is the position of the call ignoring calls to precompiled contracts
	compactPosition position
}

func (p CallPosition) IsRoot() bool {
	return p.position.Depth() == 1
}

func (p CallPosition) String() string {
	if p.IsRoot() {
		return "root"
	}
	return fmt.Sprintf("%s(%s)", p.position.String(), p.compactPosition.String())
}

func (p CallPosition) Depth() int {
	return p.position.Depth()
}

func (p CallPosition) AncestorOf(descendant ICallPosition) bool {
	if d, ok := descendant.(CallPosition); ok {
		return p.position.AncestorOf(d.position)
	}
	return false
}

func (p CallPosition) ParentOf(child ICallPosition) bool {
	if c, ok := child.(CallPosition); ok {
		return p.position.ParentOf(c.position)
	}
	return false
}

func (p CallPosition) Cmp(other ICallPosition) int {
	if o, ok := other.(CallPosition); ok {
		return p.position.Cmp(o.position)
	}
	// TODO better error handling
	return 1
}

func (p CallPosition) Equal(other ICallPosition) bool {
	if o, ok := other.(CallPosition); ok {
		return p.position.Equal(o.position)
	}
	return false
}

type position []int

func (p position) String() string {
	if len(p) == 0 {
		return "root"
	}
	var sections []string
	for _, i := range p {
		sections = append(sections, fmt.Sprintf("%d", i))
	}
	return strings.Join(sections, "_")
}

func (p position) Depth() int {
	return len(p) + 1
}

// AncestorOf returns true if the current position is the direct/indirect ancestor,
// who initial the MsgCall of the descendant position under the umbrella.
func (p position) AncestorOf(descendant position) bool {
	if len(descendant) <= len(p) {
		return false
	}
	prefix := descendant[:len(p)]
	return p.Cmp(prefix) == 0 && len(descendant) > len(p)
}

// ParentOf returns true if the current position is the direct parent,
// who initiate the MsgCall of the child position.
func (p position) ParentOf(child position) bool {
	if len(child) <= len(p) {
		return false
	}
	prefix := child[:len(p)]
	return p.Cmp(prefix) == 0 && len(child) == len(p)+1
}

// Cmp compares two CallPositions.
// Cmp returns 1 if the MsgCall of current position happens before
// the MsgCall of the other position.
// Cmp returns -1 if the MsgCall of current position happens after
// the MsgCall of the other position.
// Otherwise, Cmp returns 0.
func (p position) Cmp(other position) int {
	i := 0
	for {
		if i >= len(p) {
			if i >= len(other) {
				return 0
			}
			return -1
		}
		if i >= len(other) {
			return 1
		}
		if p[i] < other[i] {
			return -1
		}
		if p[i] > other[i] {
			return 1
		}
		i++
	}
}

func (p position) Equal(other position) bool {
	return p.Cmp(other) == 0
}
