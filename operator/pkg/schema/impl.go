package schema

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/config"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
)

type schemaImpl struct {
	gvk            config.GroupVersionKind
	plural         string
	clusterScoped  bool
	goPkg          string
	proto          string
	versionAliases []string
	apiVersion     string
	reflectType    reflect.Type
	statusType     reflect.Type
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
	return s.clusterScoped
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
	if s.reflectType == nil && getProtoMessageType(s.proto) == nil {
		err = multierror.Append(err, fmt.Errorf("proto message or reflect type not found: %v", s.proto))
	}
	return
}

type Builder struct {
	Identifier    string
	Plural        string
	ClusterScoped bool
	ProtoPackage  string
	Proto         string
	Kind          string
	Group         string
	Version       string
	ReflectType   reflect.Type
	StatusType    reflect.Type
	Builtin       bool
	Synthetic     bool
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
		goPkg:         b.ProtoPackage,
		proto:         b.Proto,
		apiVersion:    b.Group + "/" + b.Version,
		reflectType:   b.ReflectType,
		statusType:    b.StatusType,
	}
}

func (b Builder) Build() (Schema, error) {
	s := b.BuildNoValidate()
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (b Builder) MustBuild() Schema {
	s, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("MustBuild: %v", err))
	}
	return s
}

type Schema interface {
	fmt.Stringer
	GroupVersionResource() schema.GroupVersionResource
	GroupVersionKind() config.GroupVersionKind
	GroupVersionAliasKinds() []config.GroupVersionKind
	Validate() error
	IsClusterScoped() bool
}

var protoMessageType = protoregistry.GlobalTypes.FindMessageByName

func getProtoMessageType(protoMessageName string) reflect.Type {
	t, err := protoMessageType(protoreflect.FullName(protoMessageName))
	if err != nil || t == nil {
		return nil
	}
	return reflect.TypeOf(t.Zero().Interface())
}
