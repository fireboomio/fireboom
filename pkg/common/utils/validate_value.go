package utils

import (
	"golang.org/x/exp/slices"
	"reflect"
)

var validateLengthKinds = []reflect.Kind{reflect.Map, reflect.Slice}

// IsZeroValue 判断值是否为零值
func IsZeroValue(val any) bool {
	if nil == val {
		return true
	}
	if _, ok := val.(reflect.Kind); ok {
		return true
	}
	value := reflect.ValueOf(val)
	if value.IsZero() {
		return true
	}

	if slices.Contains(validateLengthKinds, value.Kind()) {
		return value.Len() == 0
	}

	return false
}
