package util

import (
	"reflect"
)

func kindOf(value interface{}) reflect.Kind {
	if value == nil {
		return reflect.Invalid
	}
	return reflect.TypeOf(value).Kind()
}

func IsSlice(value interface{}) bool {
	return kindOf(value) == reflect.Slice
}

func IsMap(value interface{}) bool {
	return kindOf(value) == reflect.Map
}
