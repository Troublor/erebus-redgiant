package summary

import (
	"fmt"
	"hash/fnv"

	"github.com/ethereum/go-ethereum/common"
	"gonum.org/v1/gonum/graph/encoding"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/multi"
)

type account struct {
	address common.Address
}

func (a account) ID() int64 {
	h := fnv.New64()
	_, _ = h.Write(a.address.Bytes())
	return int64(h.Sum64())
}

func (a account) DOTID() string {
	return a.address.Hex()
}

type transferGraphNode struct {
	multi.Line
	transfer ITransfer
}

func (t transferGraphNode) Attributes() []encoding.Attribute {
	var tok string
	var value string
	switch t.transfer.Type() {
	case EtherAsset:
		tok = "ETH"
		value = t.transfer.(EtherTransfer).Amount.String()
	case ERC20Asset:
		transfer := t.transfer.(ERC20Transfer)
		tok = transfer.Contract.Hex()[:6]
		value = transfer.Amount.String()
	case ERC721Asset:
		transfer := t.transfer.(ERC721Transfer)
		tok = transfer.Contract.Hex()[:6]
		value = fmt.Sprintf("%s", transfer.TokenID)
	case ERC777Asset:
		transfer := t.transfer.(ERC777Transfer)
		tok = transfer.Contract.Hex()[:6]
		value = transfer.Amount.String()
	case ERC1155Asset:
		transfer := t.transfer.(ERC1155Transfer)
		tokenID := transfer.TokenID.Big()
		tok = fmt.Sprintf("%s:%d", transfer.Contract.Hex()[:6], tokenID)
		value = transfer.Amount.String()
	}
	return []encoding.Attribute{
		{Key: "label", Value: fmt.Sprintf("%s:%s", tok, value)},
	}
}

func (s *CallSummary) ToDotGraph() ([]byte, error) {
	transfers := s.OverallTransfers()
	graph := multi.NewDirectedGraph()
	for _, t := range transfers {
		from := account{address: t.From()}
		to := account{address: t.To()}
		graph.SetLine(transferGraphNode{
			Line:     graph.NewLine(from, to).(multi.Line),
			transfer: t,
		})
	}
	data, err := dot.MarshalMulti(graph, s.msgCall.Receipt.TxHash.Hex(), "", "")
	if err != nil {
		return nil, err
	}
	return data, nil
}
