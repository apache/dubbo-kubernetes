package util

import (
	"bytes"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch"
	yaml2 "github.com/ghodss/yaml"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"sigs.k8s.io/yaml"
	"strings"
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

func OverlayYAML(base, overlay string) (string, error) {
	if strings.TrimSpace(base) == "" {
		return overlay, nil
	}
	if strings.TrimSpace(overlay) == "" {
		return base, nil
	}
	bj, err := yaml2.YAMLToJSON([]byte(base))
	if err != nil {
		return "", fmt.Errorf("yamlToJSON error in base: %s\n%s", err, bj)
	}
	oj, err := yaml2.YAMLToJSON([]byte(overlay))
	if err != nil {
		return "", fmt.Errorf("yamlToJSON error in overlay: %s\n%s", err, oj)
	}
	if base == "" {
		bj = []byte("{}")
	}
	if overlay == "" {
		oj = []byte("{}")
	}

	merged, err := jsonpatch.MergePatch(bj, oj)
	if err != nil {
		return "", fmt.Errorf("json merge error (%s) for base object: \n%s\n override object: \n%s", err, bj, oj)
	}
	my, err := yaml2.JSONToYAML(merged)
	if err != nil {
		return "", fmt.Errorf("jsonToYAML error (%s) for merged object: \n%s", err, merged)
	}

	return string(my), nil
}
