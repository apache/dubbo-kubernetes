package schema

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/config"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type schemaImpl struct {
	gvk           config.GroupVersionKind
	plural        string
	clusterScoped bool
	goPkg         string
	proto         string
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

func (s *schemaImpl) String() string {
	return fmt.Sprintf("[Schema](%s, %q, %s)", s.Kind(), s.goPkg, s.proto)
}

func (s *schemaImpl) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    s.Group(),
		Version:  s.Version(),
		Resource: s.Plural(),
	}
}

func (s *schemaImpl) Validate() (err error) {
	return
}

type Builder struct {
}

func (b Builder) BuildNoValidate() Schema {
}

func (b Builder) Build() (Schema, error) {
}

type Schema interface {
	fmt.Stringer
	GroupVersionResource() schema.GroupVersionResource
	GroupVersionKind() schema.GroupVersionKind
	GroupVersionAliasKinds() []config.GroupVersionKind
	Validate() error
	IsClusterScoped() bool
}
