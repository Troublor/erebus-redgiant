package data_flow

import (
	"github.com/Troublor/erebus-redgiant/dyengine/tracers"
)

type call struct {
	position tracers.CallPosition

	value FlowNode
	data  map[uint64]FlowNode

	returnData map[uint64]FlowNode
	success    FlowNode

	source FlowNode
}

func newCall(pos tracers.CallPosition) *call {
	return &call{
		position:   pos,
		data:       make(map[uint64]FlowNode),
		returnData: make(map[uint64]FlowNode),
	}
}

func (c *call) setFlowSource(source FlowNode) {
	c.source = source
}

func (c *call) matchMsgCall(msgCall *tracers.MsgCall[*Data]) bool {
	return c.position.Cmp(msgCall.Position) == 0
}

func (c *call) StoreData(memory *memory, offset, length uint64) {
	if length == 0 {
		return
	}
	start := memoryGranularity.Regulate(offset)
	for i := uint64(0); start+i < offset+length; i += memoryGranularity.Uint64() {
		c.data[i] = memory.mem[start+i]
	}
}

func (c *call) StoreReturnData(memory *memory, offset, length uint64) {
	if length == 0 {
		return
	}
	start := memoryGranularity.Regulate(offset)
	for i := uint64(0); start+i < offset+length; i += memoryGranularity.Uint64() {
		c.returnData[i] = memory.mem[start+i]
	}
}

func (c *call) GetData(start, length uint64) FlowNodeList {
	if length == 0 {
		return nil
	}
	var values FlowNodeList
	for i := memoryGranularity.Regulate(start); i < start+length; i += memoryGranularity.Uint64() {
		v := c.data[i]
		if v != UntaintedValue {
			values = append(values, v)
		}
	}
	if len(values) == 0 && c.source != nil {
		values = append(values, c.source)
	}
	return values
}

func (c *call) GetValue() FlowNode {
	return c.value
}

func (c *call) GetSuccess() FlowNode {
	if c.success != UntaintedValue {
		return c.success
	} else if c.source != nil {
		return c.source
	} else {
		return c.success
	}
}

func (c *call) GetReturnData(start, length uint64) FlowNodeList {
	if length == 0 {
		return nil
	}
	var values FlowNodeList
	for i := memoryGranularity.Regulate(start); i < start+length; i += memoryGranularity.Uint64() {
		v := c.returnData[i]
		if v != UntaintedValue {
			values = append(values, v)
		}
	}
	if len(values) == 0 && c.source != nil {
		values = append(values, c.source)
	}
	return values
}

func (c *call) GetAllReturnData() FlowNodeList {
	var values FlowNodeList
	for _, v := range c.returnData {
		if v != UntaintedValue {
			values = append(values, v)
		}
	}
	if len(values) == 0 && c.source != nil {
		values = append(values, c.source)
	}
	return values
}

func (c *call) ReturnDataIsTainted(analyzer Analyzer) bool {
	for _, node := range c.returnData {
		if node != UntaintedValue && node.From().IsTainted() {
			return true
		}
	}
	return false
}
