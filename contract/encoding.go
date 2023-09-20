package contract

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ParseEvent parse the log to get event arguments according to the abi.Event specification.
// It returns the decoded arguments of the event and any error that may have occurred.
// The indexed and non-indexed arguments are merged together in the return value following
// the order of event definition.
func ParseEvent(eventAbi abi.Event, log *types.Log) ([]interface{}, error) {
	var err error
	var args []interface{}

	var nonIndexedArgs []interface{}
	if len(eventAbi.Inputs.NonIndexed()) > 0 {
		nonIndexedArgs, err = eventAbi.Inputs.Unpack(log.Data)
		if err != nil {
			return nil, err
		}
	}

	indexedIndex := 0
	nonIndexedIndex := 0
	for _, inputSpec := range eventAbi.Inputs {
		if inputSpec.Indexed {
			// non-packable indexed arguments are left as nil
			var arg interface{}
			if indexedIndex+1 >= len(log.Topics) {
				return nil, fmt.Errorf("not enough topics for indexed argument %d", indexedIndex)
			}
			data := log.Topics[indexedIndex+1].Bytes()
			switch inputSpec.Type.T {
			case abi.IntTy, abi.UintTy:
				arg = abi.ReadInteger(inputSpec.Type, data)
			case abi.BoolTy:
				arg, err = readBool(data)
				if err != nil {
					return nil, err
				}
			case abi.AddressTy:
				arg = common.BytesToAddress(data)
			case abi.HashTy:
				arg = common.BytesToHash(data)
			case abi.FixedBytesTy:
				if inputSpec.Type.Size <= 32 {
					arg = data
				}
			}
			args = append(args, arg)
			indexedIndex++
		} else {
			if nonIndexedIndex >= len(nonIndexedArgs) {
				return nil, fmt.Errorf("not enough non-indexed arguments for non-indexed argument %d", nonIndexedIndex)
			}
			args = append(args, nonIndexedArgs[nonIndexedIndex])
			nonIndexedIndex++
		}
	}

	return args, err
}

var ErrBadBool = errors.New("abi: improperly encoded boolean value")

func readBool(word []byte) (bool, error) {
	for _, b := range word[:31] {
		if b != 0 {
			return false, ErrBadBool
		}
	}
	switch word[31] {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, ErrBadBool
	}
}

var ErrNoMatchedEvent = errors.New("no matched event")
var ErrNoFallbackFunction = errors.New("no fallback function")
var ErrNoReceiveFunction = errors.New("no receive function")

// UnpackLog unpacks the raw EVM log according to the ABI definition.
// It returns the corresponding event specification,
// arguments of the event and any error that may have occurred.
// If there is no matching event in the abi, ErrNoMatchedEvent is returned.
func UnpackLog(abi abi.ABI, log *types.Log) (*abi.Event, []interface{}, error) {
	if len(log.Topics) == 0 {
		// no topics, this is not a solidity event
		return nil, nil, ErrNoMatchedEvent
	}

	ev, err := abi.EventByID(log.Topics[0])
	if err != nil {
		return nil, nil, ErrNoMatchedEvent
	}

	args, err := ParseEvent(*ev, log)
	return ev, args, err
}

// UnpackInput unpacks the input of a message call according to the given abi specification.
// The input should include the function signature that is being called.
// It returns the specification of the function being called,
// arguments (as an array) and any error that may have occurred.
func UnpackInput(abi abi.ABI, input []byte) (fn *abi.Method, args []interface{}, err error) {
	if len(input) == 0 {
		goto receive
	}
	if len(input) < 4 {
		goto fallback
	} else {
		sig := input[:4]
		fn, err = abi.MethodById(sig)
		if err != nil {
			// method not found
			goto fallback
		}
		args, err = fn.Inputs.Unpack(input[4:])
		if err != nil {
			return nil, nil, err
		}
		return fn, args, nil
	}

fallback:
	// no valid function signature
	if abi.HasFallback() {
		return &abi.Fallback, []interface{}{input}, nil
	} else {
		return nil, nil, ErrNoFallbackFunction
	}
receive:
	if abi.HasReceive() {
		return &abi.Receive, []interface{}{}, nil
	} else {
		return nil, nil, ErrNoReceiveFunction
	}
}
