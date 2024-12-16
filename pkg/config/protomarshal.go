package config

import "google.golang.org/protobuf/proto"

func MarshalIndent(msg proto.Message, indent string) ([]byte, error) {
}

func ToJSONWithIndent(msg proto.Message, indent string) (string, error) {
	return "", nil
}

func ToJSONWithOptions(msg proto.Message, indent string) (string, error) {
	return "", nil
}
