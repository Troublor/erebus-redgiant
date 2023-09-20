package data_flow

var memoryGranularity ArrayGranularity = 32 // bytes

type ArrayGranularity uint64

func (m ArrayGranularity) Regulate(index uint64) uint64 {
	return index - (index % uint64(m))
}
func (m ArrayGranularity) Uint64() uint64 {
	return uint64(m)
}

// memory is implement at Word (32 bytes) level granularity.
type memory struct {
	mem map[uint64]FlowNode
}

func newMemory() *memory {
	return &memory{
		mem: make(map[uint64]FlowNode),
	}
}

func (m *memory) Load(start, length uint64) FlowNodeList {
	if length == 0 {
		return nil
	}
	var values FlowNodeList
	for i := memoryGranularity.Regulate(start); i < start+length; i += memoryGranularity.Uint64() {
		v := m.mem[i]
		if v != UntaintedValue {
			values = append(values, v)
		}
	}
	return values
}

func (m *memory) Store(start, length uint64, value FlowNode) {
	if length == 0 {
		return
	}
	for i := memoryGranularity.Regulate(start); i < start+length; i += memoryGranularity.Uint64() {
		old := m.mem[i]
		if (i < start || i+memoryGranularity.Uint64() > start+length) && old != UntaintedValue {
			if value == UntaintedValue {
				m.mem[i] = old
			} else {
				node := NewRawFlowNode(value.Operation(), "memory_merge")
				node.AddUpstream(value, old)
				m.mem[i] = node
			}
		} else {
			m.mem[i] = value
		}
	}
}

func (m *memory) Clear(start, length uint64) {
	m.Store(start, length, UntaintedValue)
}

func (m *memory) Get(index uint64) FlowNode {
	i := memoryGranularity.Regulate(index)
	return m.mem[i]
}

func (m *memory) Set(index uint64, node FlowNode) {
	i := memoryGranularity.Regulate(index)
	old := m.mem[i]
	if i < index && old != UntaintedValue {
		if node == UntaintedValue {
			m.mem[i] = old
		} else {
			node := NewRawFlowNode(node.Operation(), "memory_merge")
			node.AddUpstream(node, old)
			m.mem[i] = node
		}
	} else {
		m.mem[i] = node
	}
}

func (m *memory) StoreCallData(call *call, destOffset, offset, length uint64) {
	index := memoryGranularity.Regulate(destOffset)
	for i := memoryGranularity.Regulate(offset); i < offset+length; i += memoryGranularity.Uint64() {
		old := m.mem[index]
		callData := call.data[i]
		if (index < destOffset || index+memoryGranularity.Uint64() > destOffset+length) &&
			old != UntaintedValue {
			if callData == UntaintedValue {
				m.mem[index] = old
			} else {
				node := NewRawFlowNode(callData.Operation(), "memory_merge")
				node.AddUpstream(callData, old)
				m.mem[index] = node
			}
		} else {
			m.mem[index] = callData
		}
		index += memoryGranularity.Uint64()
	}
}

func (m *memory) StoreCallReturnData(call *call, destOffset, offset, length uint64) {
	index := memoryGranularity.Regulate(destOffset)
	for i := memoryGranularity.Regulate(offset); i < offset+length; i += memoryGranularity.Uint64() {
		old := m.mem[index]
		callReturnData := call.returnData[i]
		if (index < destOffset || index+memoryGranularity.Uint64() > destOffset+length) &&
			old != UntaintedValue {
			if callReturnData == UntaintedValue {
				m.mem[index] = old
			} else {
				node := NewRawFlowNode(callReturnData.Operation(), "memory_merge")
				node.AddUpstream(callReturnData, old)
				m.mem[index] = node
			}
		} else {
			m.mem[index] = callReturnData
		}
		index += memoryGranularity.Uint64()
	}
}
