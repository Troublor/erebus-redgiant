package summary

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
)

type StateVariableBsonDecoder struct {
}

func (pd *StateVariableBsonDecoder) DecodeValue(
	dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value,
) error {
	if !val.CanSet() {
		return bsoncodec.ValueDecoderError{
			Name:     "StateVariableDecodeValue",
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

	var stateVariable StateVariable
	switch tempMap["type"] {
	case StorageVar:
		stateVariable = StorageVariable{
			Address: common.HexToAddress(tempMap["address"].(string)),
			Storage: common.HexToHash(tempMap["storage"].(string)),
			Value:   common.HexToHash(tempMap["value"].(string)),
		}
	case BalanceVar:
		v := tempMap["value"].(string)
		value, ok := big.NewInt(0).SetString(v, 10)
		if !ok {
			return fmt.Errorf("failed to parse value %s", v)
		}
		stateVariable = BalanceVariable{
			Address: common.HexToAddress(tempMap["address"].(string)),
			Value:   value,
		}
	case CodeVar:
		stateVariable = CodeVariable{
			Address: common.HexToAddress(tempMap["address"].(string)),
			Op:      vm.StringToOp(tempMap["op"].(string)),
		}
	}
	val.Set(reflect.ValueOf(stateVariable))
	return nil
}

func (s StorageVariable) MarshalBSON() ([]byte, error) {
	var cp = struct {
		Address string       `bson:"address"`
		Storage string       `bson:"storage"`
		Value   string       `bson:"value"`
		Type    StateVarType `bson:"type"`
	}{s.Address.Hex(), s.Storage.Hex(), s.Value.Hex(), s.Type()}
	return bson.Marshal(cp)
}

func (b BalanceVariable) MarshalBSON() ([]byte, error) {
	var cp = struct {
		Address string       `bson:"address"`
		Value   string       `bson:"value"`
		Type    StateVarType `bson:"type"`
	}{b.Address.Hex(), b.Value.String(), b.Type()}
	return bson.Marshal(cp)
}

func (c CodeVariable) MarshalBSON() ([]byte, error) {
	var cp = struct {
		Address string       `bson:"address"`
		Op      string       `bson:"op"`
		Type    StateVarType `bson:"type"`
	}{c.Address.Hex(), c.Op.String(), c.Type()}
	return bson.Marshal(cp)
}
