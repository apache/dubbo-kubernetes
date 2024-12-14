package apis

import (
	"bytes"
	"encoding/json"

	github_com_golang_protobuf_jsonpb "github.com/golang/protobuf/jsonpb" // nolint: depguard
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// nolint
var _ github_com_golang_protobuf_jsonpb.JSONPBUnmarshaler = &IntOrString{}

func (i *IntOrString) UnmarshalJSON(value []byte) error {
	if value[0] == '"' {
		i.Type = int64(intstr.String)
		var s string
		err := json.Unmarshal(value, &s)
		if err != nil {
			return err
		}
		i.StrVal = &wrapperspb.StringValue{Value: s}
		return nil
	}
	i.Type = int64(intstr.Int)
	var s int32
	err := json.Unmarshal(value, &s)
	if err != nil {
		return err
	}
	i.IntVal = &wrapperspb.Int32Value{Value: s}
	return nil
}

func (i *IntOrString) MarshalJSONPB(_ *github_com_golang_protobuf_jsonpb.Marshaler) ([]byte, error) {
	return i.MarshalJSON()
}

func (i *IntOrString) MarshalJSON() ([]byte, error) {
	if i.IntVal != nil {
		return json.Marshal(i.IntVal.GetValue())
	}
	return json.Marshal(i.StrVal.GetValue())
}

func (i *IntOrString) UnmarshalJSONPB(_ *github_com_golang_protobuf_jsonpb.Unmarshaler, value []byte) error {
	return i.UnmarshalJSON(value)
}

func (i *IntOrString) ToKubernetes() intstr.IntOrString {
	if i.IntVal != nil {
		return intstr.FromInt32(i.GetIntVal().GetValue())
	}
	return intstr.FromString(i.GetStrVal().GetValue())
}

// MarshalJSON is a custom marshaler for Values
func (in *Values) MarshalJSON() ([]byte, error) {
	str, err := OperatorMarshaler.MarshalToString(in)
	return []byte(str), err
}

// UnmarshalJSON is a custom unmarshaler for Values
func (in *Values) UnmarshalJSON(b []byte) error {
	return OperatorUnmarshaler.Unmarshal(bytes.NewReader(b), in)
}

var (
	OperatorMarshaler   = &github_com_golang_protobuf_jsonpb.Marshaler{}
	OperatorUnmarshaler = &github_com_golang_protobuf_jsonpb.Unmarshaler{}
)
