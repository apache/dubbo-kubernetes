package schema

import (
	"github.com/apache/dubbo-kubernetes/operator/pkg/config"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type schemaImpl struct {
	gvk           config.GroupVersionKind
	plural        string
	clusterScoped bool
}

func (s *schemaImpl) Kind() string {
	return s.gvk.Kind
}

func (s *schemaImpl) Group() string {
	return s.gvk.Group
}

func (s *schemaImpl) Version() string {
	return s.gvk.Version
}

func (s *schemaImpl) Plural() string {
	return s.plural
}

func (s *schemaImpl) GroupVersionKind() config.GroupVersionKind {
	return s.gvk
}

func (s *schemaImpl) InClusterScoped() bool {
	return s.clusterScoped
}

func (s *schemaImpl) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    s.Group(),
		Version:  s.Version(),
		Resource: s.Plural(),
	}
}

type Builder struct {
}

