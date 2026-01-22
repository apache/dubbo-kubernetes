//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package protomarshal

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/protobuf/jsonpb"             // nolint: depguard
	legacyproto "github.com/golang/protobuf/proto"  // nolint: staticcheck
	"google.golang.org/protobuf/encoding/protojson" // nolint: depguard
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"sigs.k8s.io/yaml"
)

var (
	strictUnmarshaler = jsonpb.Unmarshaler{}
)

type ComparableMessage interface {
	comparable
	proto.Message
}

func ToJSON(msg proto.Message) (string, error) {
	return ToJSONWithIndent(msg, "")
}

func Marshal(msg proto.Message) ([]byte, error) {
	res, err := ToJSONWithIndent(msg, "")
	if err != nil {
		return nil, err
	}
	return []byte(res), err
}

func MarshalIndent(msg proto.Message, indent string) ([]byte, error) {
	res, err := ToJSONWithIndent(msg, indent)
	if err != nil {
		return nil, err
	}
	return []byte(res), err
}

func MarshalProtoNames(msg proto.Message) ([]byte, error) {
	if msg == nil {
		return nil, errors.New("unexpected nil message")
	}

	// Marshal from proto to json bytes
	m := jsonpb.Marshaler{OrigName: true}
	buf := &bytes.Buffer{}
	err := m.Marshal(buf, legacyproto.MessageV1(msg))
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Unmarshal(b []byte, m proto.Message) error {
	return strictUnmarshaler.Unmarshal(bytes.NewReader(b), legacyproto.MessageV1(m))
}

func ToJSONWithIndent(msg proto.Message, indent string) (string, error) {
	return ToJSONWithOptions(msg, indent, false)
}

func ToJSONWithOptions(msg proto.Message, indent string, enumsAsInts bool) (string, error) {
	if msg == nil {
		return "", errors.New("unexpected nil message")
	}

	// Marshal from proto to json bytes
	m := jsonpb.Marshaler{Indent: indent, EnumsAsInts: enumsAsInts}
	return m.MarshalToString(legacyproto.MessageV1(msg))
}

func ToYAML(msg proto.Message) (string, error) {
	js, err := ToJSON(msg)
	if err != nil {
		return "", err
	}
	yml, err := yaml.JSONToYAML([]byte(js))
	return string(yml), err
}

func ApplyJSON(js string, pb proto.Message) error {
	reader := strings.NewReader(js)
	m := jsonpb.Unmarshaler{}
	if err := m.Unmarshal(reader, legacyproto.MessageV1(pb)); err != nil {
		fmt.Printf("Failed to decode proto: %q. Trying decode with AllowUnknownFields=true", err)
		m.AllowUnknownFields = true
		reader.Reset(js)
		return m.Unmarshal(reader, legacyproto.MessageV1(pb))
	}
	return nil
}

func ApplyJSONStrict(js string, pb proto.Message) error {
	reader := strings.NewReader(js)
	m := jsonpb.Unmarshaler{}
	return m.Unmarshal(reader, legacyproto.MessageV1(pb))
}

func ApplyYAML(yml string, pb proto.Message) error {
	js, err := yaml.YAMLToJSON([]byte(yml))
	if err != nil {
		return err
	}
	return ApplyJSON(string(js), pb)
}

func Clone[T proto.Message](obj T) T {
	return proto.Clone(obj).(T)
}

func MessageToStructSlow(msg proto.Message) (*structpb.Struct, error) {
	if msg == nil {
		return nil, errors.New("nil message")
	}

	b, err := (&protojson.MarshalOptions{UseProtoNames: true}).Marshal(msg)
	if err != nil {
		return nil, err
	}

	pbs := &structpb.Struct{}
	if err := protojson.Unmarshal(b, pbs); err != nil {
		return nil, err
	}

	return pbs, nil
}
