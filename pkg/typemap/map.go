package typemap

import (
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"reflect"
)

type TypeMap struct {
	inner map[reflect.Type]any
}

func NewTypeMap() TypeMap {
	return TypeMap{make(map[reflect.Type]any)}
}

func Set[T any](t TypeMap, v T) {
	interfaceType := reflect.TypeOf((*T)(nil)).Elem()
	t.inner[interfaceType] = v
}

func Get[T any](t TypeMap) *T {
	v, f := t.inner[reflect.TypeFor[T]()]
	if f {
		return ptr.Of(v.(T))
	}
	return nil
}
