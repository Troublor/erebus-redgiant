package data_flow

type Analyzer interface {
	NewFlowNode(operation *Operation) FlowNode
	CheckOperation(operation *Operation) (isSource, isSink bool)
	SinkTainted(collection *TrackerCollection, flowedValue FlowNode)
	FlowPolicy() FlowPolicy
}
