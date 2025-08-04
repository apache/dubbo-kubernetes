package kubeclient

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"
	ktypes "github.com/apache/dubbo-kubernetes/pkg/kube/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	k8sioapicorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetWriteClient[T runtime.Object](c ClientGetter, namespace string) ktypes.WriteAPI[T] {
	switch any(ptr.Empty[T]()).(type) {
	case *k8sioapicorev1.ConfigMap:
		return c.Kube().CoreV1().ConfigMaps(namespace).(ktypes.WriteAPI[T])
	default:
		panic(fmt.Sprintf("Unknown type %T", ptr.Empty[T]()))
	}
}

func gvrToObject(g schema.GroupVersionResource) runtime.Object {
	switch g {
	case gvr.ConfigMap:
		return &k8sioapicorev1.ConfigMap{}
	default:
		panic(fmt.Sprintf("Unknown type %v", g))
	}
}
