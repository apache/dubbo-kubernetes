package gvk

import (
	"github.com/apache/dubbo-kubernetes/operator/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	CustomResourceDefinition = config.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}
)

func ToGVR(g config.GroupVersionKind) (schema.GroupVersionResource, bool) {
	switch g {
	case CustomResourceDefinition:
		return gvr.CustomResourceDefinition, true
	}
	return schema.GroupVersionResource{}, false
}
func MustToGVR(g config.GroupVersionKind) schema.GroupVersionResource {
	r, ok := ToGVR(g)
	if !ok {
		panic("unknown kind: " + g.String())
	}
	return r
}

func FromGVR(g schema.GroupVersionResource) (config.GroupVersionKind, bool) {
	switch g {
	case gvr.CustomResourceDefinition:
		return CustomResourceDefinition, true
	}

	return config.GroupVersionKind{}, false
}

func MustFromGVR(g schema.GroupVersionResource) config.GroupVersionKind {
	r, ok := FromGVR(g)
	if !ok {
		panic("unknown kind: " + g.String())
	}
	return r
}
