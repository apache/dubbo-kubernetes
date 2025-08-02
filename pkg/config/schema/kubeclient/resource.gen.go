package kubeclient

import (
	"fmt"
	ktypes "github.com/apache/dubbo-kubernetes/pkg/kube/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"k8s.io/apimachinery/pkg/runtime"
)

func GetWriteClient[T runtime.Object](c ClientGetter, namespace string) ktypes.WriteAPI[T] {
	switch any(ptr.Empty[T]()).(type) {
	default:
		panic(fmt.Sprintf("Unknown type %T", ptr.Empty[T]()))
	}
}
