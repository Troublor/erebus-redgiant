package helpers

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type ContractInfo struct {
	Abi abi.ABI `json:"abi"`
	Evm struct {
		Assembly string `json:"assembly"`
		Bytecode struct {
			Object    Bytecode `json:"object"`
			Opcodes   string   `json:"opcodes"`
			SourceMap string   `json:"sourceMap"`
		} `json:"bytecode"`
		DeployedBytecode struct {
			Object    Bytecode `json:"object"`
			Opcodes   string   `json:"opcodes"`
			SourceMap string   `json:"sourceMap"`
		} `json:"deployedBytecode"`
	}
}

type CompilationResult struct {
	Contracts map[string]map[string]ContractInfo `json:"contracts"`
}

func LoadCompilationResult(jsonFile string) (*CompilationResult, error) {
	var r CompilationResult
	file, err := os.Open(jsonFile)
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
