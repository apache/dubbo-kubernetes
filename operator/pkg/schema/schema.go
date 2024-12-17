package schema

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/config"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type schemaImpl struct {
	gvk            config.GroupVersionKind
	plural         string
	clusterScoped  bool
	goPkg          string
	proto          string
	versionAliases []string
}

func (s *schemaImpl) GroupVersionAliasKinds() []config.GroupVersionKind {
	gvks := make([]config.GroupVersionKind, len(s.versionAliases))
	for i, va := range s.versionAliases {
		gvks[i] = s.gvk
		gvks[i].Version = va
	}
	gvks = append(gvks, s.GroupVersionKind())
	return gvks
}

func (s *schemaImpl) IsClusterScoped() bool {
	//TODO implement me
	panic("implement me")
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
	Plural        string
	ClusterScoped bool
	ProtoPkg      string
	Proto         string
	Kind          string
	Group         string
	Version       string
}

func (b Builder) BuildNoValidate() Schema {
	return &schemaImpl{
		gvk: config.GroupVersionKind{
			Group:   b.Group,
			Version: b.Version,
			Kind:    b.Kind,
		},
		plural:        b.Plural,
		clusterScoped: b.ClusterScoped,
		goPkg:         b.ProtoPkg,
		proto:         b.Proto,
	}
}

func (b Builder) Build() (Schema, error) {
	s := b.BuildNoValidate()
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

type Schema interface {
	fmt.Stringer
	GroupVersionResource() schema.GroupVersionResource
	GroupVersionKind() config.GroupVersionKind
	GroupVersionAliasKinds() []config.GroupVersionKind
	Validate() error
	IsClusterScoped() bool
}
