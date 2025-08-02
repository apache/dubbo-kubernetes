package kubetypes

import (
	"github.com/apache/dubbo-kubernetes/operator/pkg/config"
)

func getGvk(obj any) (config.GroupVersionKind, bool) {
	switch obj.(type) {
	default:
		return config.GroupVersionKind{}, false
	}
}
