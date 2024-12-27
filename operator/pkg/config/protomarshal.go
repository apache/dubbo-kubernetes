package config

import (
	"errors"
	"github.com/golang/protobuf/jsonpb"
	legacyproto "github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/proto"
)

func MarshalIndent(msg proto.Message, indent string) ([]byte, error) {
	res, err := ToJSONWithIndent(msg, indent)
	if err != nil {
		return nil, err
	}
	return []byte(res), err
}

func ToJSONWithIndent(msg proto.Message, indent string) (string, error) {
	return ToJSONWithOptions(msg, indent, false)
}

func ToJSONWithOptions(msg proto.Message, indent string, enumsAsInts bool) (string, error) {
	if msg == nil {
		return "", errors.New("unexpected nil message")
	}
	m := jsonpb.Marshaler{Indent: indent, EnumsAsInts: enumsAsInts}
	return m.MarshalToString(legacyproto.MessageV1(msg))
}
