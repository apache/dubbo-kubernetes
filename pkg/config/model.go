package config

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type GroupVersionKind struct {
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

var _ fmt.Stringer = GroupVersionKind{}

func (g GroupVersionKind) String() string {
}

func (g GroupVersionKind) CanonicalGroup() string {
	return CanoncalGroup(g.Group)
}

func CanoncalGroup(group string) string {
	if group != "" {
		return group
	}
	return "core"
}

type Spec any

func ToJSON(s Spec) ([]byte, error) {
	return toJSON()
}

func toJSON(s Spec, pretty bool) ([]byte, error) {
	indent := ""
	if pretty {
		indent = "    "
	}
	if _, ok := s.(protoreflect.ProtoMessage); ok {
		if pb, ok := s.(proto.Message); ok {
		}
	}
	return nil, nil
}
