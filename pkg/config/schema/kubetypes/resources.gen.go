package kubetypes

import (
	"github.com/apache/dubbo-kubernetes/operator/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	k8sioapicorev1 "k8s.io/api/core/v1"
)

func getGvk(obj any) (config.GroupVersionKind, bool) {
	switch obj.(type) {
	case *k8sioapicorev1.ConfigMap:
		return gvk.ConfigMap, true
	default:
		return config.GroupVersionKind{}, false
	}

}
