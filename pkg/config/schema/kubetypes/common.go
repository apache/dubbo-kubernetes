package kubetypes

import (
	"github.com/apache/dubbo-kubernetes/operator/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"github.com/apache/dubbo-kubernetes/pkg/typemap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func MustGVKFromType[T runtime.Object]() (cfg config.GroupVersionKind) {
	if gvk, ok := getGvk(ptr.Empty[T]()); ok {
		return gvk
	}
	if rp := typemap.Get[RegisterType[T]](registeredTypes); rp != nil {
		return (*rp).GetGVK()
	}
	panic("unknown kind: " + cfg.String())
}

func MustToGVR[T runtime.Object](cfg config.GroupVersionKind) schema.GroupVersionResource {
	if r, ok := gvk.ToGVR(cfg); ok {
		return r
	}
	if rp := typemap.Get[RegisterType[T]](registeredTypes); rp != nil {
		return (*rp).GetGVR()
	}
	panic("unknown kind: " + cfg.String())
}

var registeredTypes = typemap.NewTypeMap()

type RegisterType[T runtime.Object] interface {
	GetGVK() config.GroupVersionKind
	GetGVR() schema.GroupVersionResource
	Object() T
}
