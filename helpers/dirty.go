package helpers

import (
	"reflect"
	"unsafe"

	"github.com/ethereum/go-ethereum/core/vm"
)

//go:linkname InstructionSet github.com/ethereum/go-ethereum/core/vm.newLondonInstructionSet
func InstructionSet() vm.JumpTable

func GetUnexportedField(obj interface{}, fieldName string) interface{} {
	objT := reflect.ValueOf(obj).Elem()
	field := objT.FieldByName(fieldName)
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}
