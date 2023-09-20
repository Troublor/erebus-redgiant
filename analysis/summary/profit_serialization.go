package summary

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
)

type ProfitBsonDecoder struct {
}

func (pd *ProfitBsonDecoder) DecodeValue(
	dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value,
) error {
	if !val.CanSet() {
		return bsoncodec.ValueDecoderError{
			Name:     "ProfitDecodeValue",
			Received: val,
		}
	}

	tempMap := make(map[string]interface{})
	mapCodec := bsoncodec.NewMapCodec()
	rv := reflect.ValueOf(tempMap)
	err := mapCodec.DecodeValue(dc, vr, rv)
	if err != nil {
		return fmt.Errorf("could not decode profit as map: %v", err)
	}

	var profit Profit
	switch tempMap["type"] {
	case EtherAsset:
		a := tempMap["amount"].(string)
		amount, ok := big.NewInt(0).SetString(a, 10)
		if !ok {
			return fmt.Errorf("failed to parse amount %s", a)
		}
		profit = EtherProfit{
			Account: common.HexToAddress(tempMap["account"].(string)),
			Amount:  amount,
		}
	case ERC20Asset:
		a := tempMap["amount"].(string)
		amount, ok := big.NewInt(0).SetString(a, 10)
		if !ok {
			return fmt.Errorf("failed to parse amount %s", a)
		}
		profit = ERC20Profit{
			Account:  common.HexToAddress(tempMap["account"].(string)),
			Contract: common.HexToAddress(tempMap["contract"].(string)),
			Amount:   amount,
		}
	case ERC721Asset:
		receiveTokens := make(map[common.Hash]interface{})
		for _, token := range tempMap["receiveTokens"].(bson.A) {
			receiveTokens[common.HexToHash(token.(string))] = nil
		}
		giveTokens := make(map[common.Hash]interface{})
		for _, token := range tempMap["giveTokens"].(bson.A) {
			giveTokens[common.HexToHash(token.(string))] = nil
		}
		profit = ERC721CombinedProfit{
			Account:       common.HexToAddress(tempMap["account"].(string)),
			Contract:      common.HexToAddress(tempMap["contract"].(string)),
			receiveTokens: receiveTokens,
			giveTokens:    giveTokens,
		}
	case ERC777Asset:
		a := tempMap["amount"].(string)
		amount, ok := big.NewInt(0).SetString(a, 10)
		if !ok {
			return fmt.Errorf("failed to parse amount %s", a)
		}
		profit = ERC777Profit{
			Account:  common.HexToAddress(tempMap["account"].(string)),
			Contract: common.HexToAddress(tempMap["contract"].(string)),
			Amount:   amount,
		}
	case ERC1155Asset:
		a := tempMap["amount"].(string)
		amount, ok := big.NewInt(0).SetString(a, 10)
		if !ok {
			return fmt.Errorf("failed to parse amount %s", a)
		}
		profit = ERC1155Profit{
			Account:  common.HexToAddress(tempMap["account"].(string)),
			Contract: common.HexToAddress(tempMap["contract"].(string)),
			TokenID:  common.HexToHash(tempMap["tokenId"].(string)),
			Amount:   amount,
		}
	}
	val.Set(reflect.ValueOf(profit))
	return nil
}

func (e EtherProfit) MarshalBSON() ([]byte, error) {
	var cp = struct {
		Account string `bson:"account"`
		Amount  string `bson:"amount"`
		Type    string `bson:"type"`
	}{e.Account.Hex(), e.Amount.String(), e.Type()}
	return bson.Marshal(cp)
}

func (t ERC20Profit) MarshalBSON() ([]byte, error) {
	var cp = struct {
		Account  string `bson:"account"`
		Contract string `bson:"contract"`
		Amount   string `bson:"amount"`
		Type     string `bson:"type"`
	}{t.Account.Hex(), t.Contract.Hex(), t.Amount.String(), t.Type()}
	return bson.Marshal(cp)
}

func (n ERC721CombinedProfit) MarshalBSON() ([]byte, error) {
	rts := make([]string, 0, len(n.receiveTokens))
	for hash := range n.receiveTokens {
		rts = append(rts, hash.Hex())
	}
	gts := make([]string, 0, len(n.giveTokens))
	for hash := range n.giveTokens {
		gts = append(gts, hash.Hex())
	}
	var cp = struct {
		Account       string   `bson:"account"`
		Contract      string   `bson:"contract"`
		ReceiveTokens []string `bson:"receiveTokens"`
		GiveTokens    []string `bson:"giveTokens"`
		Type          string   `bson:"type"`
	}{n.Account.Hex(), n.Contract.Hex(), rts, gts, n.Type()}
	return bson.Marshal(cp)
}

func (n ERC721Profit) MarshalBSON() ([]byte, error) {
	var tokenID *string
	if n.TokenID != nil {
		id := n.TokenID.Hex()
		tokenID = &id
	}
	var cp = struct {
		Account  string  `bson:"account"`
		Contract string  `bson:"contract"`
		TokenID  *string `bson:"tokenId"`
		Receive  bool    `bson:"receive"`
		Type     string  `bson:"type"`
	}{n.Account.Hex(), n.Contract.Hex(), tokenID, n.Receive, n.Type()}
	return bson.Marshal(cp)
}

func (t ERC777Profit) MarshalBSON() ([]byte, error) {
	var cp = struct {
		Account  string `bson:"account"`
		Contract string `bson:"contract"`
		Amount   string `bson:"amount"`
		Type     string `bson:"type"`
	}{t.Account.Hex(), t.Contract.Hex(), t.Amount.String(), t.Type()}
	return bson.Marshal(cp)
}

func (t ERC1155Profit) MarshalBSON() ([]byte, error) {
	var cp = struct {
		Account  string `bson:"account"`
		Contract string `bson:"contract"`
		TokenID  string `bson:"tokenId"`
		Amount   string `bson:"amount"`
		Type     string `bson:"type"`
	}{t.Account.Hex(), t.Contract.Hex(), t.TokenID.Hex(), t.Amount.String(), t.Type()}
	return bson.Marshal(cp)
}
