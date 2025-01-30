package util

import (
	"bytes"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"sigs.k8s.io/yaml"
)

func UnmarshalWithJSONPB(y string, out proto.Message, allowUnknownField bool) error {
	if y == "" {
		return nil
	}
	jb, err := yaml.YAMLToJSON([]byte(y))
	if err != nil {
		return err
	}
	u := jsonpb.Unmarshaler{AllowUnknownFields: allowUnknownField}
	err = u.Unmarshal(bytes.NewReader(jb), out)
	if err != nil {
		return err
	}
	return nil
}

func ToYAMLWithJSONPB(val proto.Message) string {
	m := jsonpb.Marshaler{EnumsAsInts: true}
	js, err := m.MarshalToString(val)
	if err != nil {
		return err.Error()
	}
	yb, err := yaml.JSONToYAML([]byte(js))
	if err != nil {
		return err.Error()
	}
	return string(yb)
}
