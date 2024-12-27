package render

import (
	"github.com/apache/dubbo-kubernetes/operator/pkg/component"
	"github.com/apache/dubbo-kubernetes/operator/pkg/manifest"
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
)

func postProcess(comp component.Component, manifests []manifest.Manifest, vals values.Map) ([]manifest.Manifest, error) {
	return manifests, nil
}
