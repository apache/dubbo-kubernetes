package manifest

import (
	"github.com/apache/dubbo-kubernetes/operator/pkg/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Manifest struct {
	*unstructured.Unstructured
	Content string
}

type ManifestSet struct {
	Components component.Name
	Manifests  []Manifest
}
