package helpers

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/status-im/keycard-go/hexutils"
)

type AssembleResult struct {
	SourceFile string   `json:"sourceFile"`
	Bytecode   Bytecode `json:"bytecode"`
	Assembly   string   `json:"assembly"`
	Opcodes    string   `json:"opcodes"`
	SourceMap  string   `json:"sourceMap"`
}

func LoadAssembleResult(jsonFile string) (*AssembleResult, error) {
	file, err := os.Open(jsonFile)
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var r AssembleResult
	err = json.Unmarshal(bytes, &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

type Bytecode []byte

func (b *Bytecode) MarshalJSON() ([]byte, error) {
	return []byte(hexutils.BytesToHex(*b)), nil
}

func (b *Bytecode) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	*b = hexutils.HexToBytes(s)
	return nil
}
