package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	gogojsonpb "github.com/gogo/protobuf/jsonpb"
	gogoproto "github.com/gogo/protobuf/proto"
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
	return g.CanonicalGroup() + "/" + g.Version + "/" + g.Kind
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
	return toJSON(s, false)
}

func toJSON(s Spec, pretty bool) ([]byte, error) {
	indent := ""
	if pretty {
		indent = "    "
	}
	if _, ok := s.(protoreflect.ProtoMessage); ok {
		if pb, ok := s.(proto.Message); ok {
			b, err := MarshalIndent(pb, indent)
			return b, err
		}
	}
	b := &bytes.Buffer{}
	if pb, ok := s.(gogoproto.Message); ok {
		err := (&gogojsonpb.Marshaler{Indent: indent}).Marshal(b, pb)
		return b.Bytes(), err
	}
	if pretty {
		return json.MarshalIndent(s, "", indent)
	}
	return json.Marshal(s)
}
