/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package model

import (
	"encoding/json"
	"path"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"sigs.k8s.io/yaml"

	utilproto "github.com/apache/dubbo-kubernetes/pkg/common/util/proto"
	"github.com/apache/dubbo-kubernetes/pkg/core_legacy/resources"
)

func ToJSON(spec resources.ResourceSpec) ([]byte, error) {
	if msg, ok := spec.(proto.Message); ok {
		return utilproto.ToJSON(msg)
	} else {
		return json.Marshal(spec)
	}
}

func ToYAML(spec resources.ResourceSpec) ([]byte, error) {
	if msg, ok := spec.(proto.Message); ok {
		return utilproto.ToYAML(msg)
	} else {
		return yaml.Marshal(spec)
	}
}

func ToAny(spec resources.ResourceSpec) (*anypb.Any, error) {
	if msg, ok := spec.(proto.Message); ok {
		return utilproto.MarshalAnyDeterministic(msg)
	} else {
		bytes, err := json.Marshal(spec)
		if err != nil {
			return nil, err
		}
		return &anypb.Any{
			Value: bytes,
		}, nil
	}
}

func FromJSON(src []byte, spec resources.ResourceSpec) error {
	if msg, ok := spec.(proto.Message); ok {
		return utilproto.FromJSON(src, msg)
	} else {
		return json.Unmarshal(src, spec)
	}
}

func FromYAML(src []byte, spec resources.ResourceSpec) error {
	if msg, ok := spec.(proto.Message); ok {
		return utilproto.FromYAML(src, msg)
	} else {
		return yaml.Unmarshal(src, spec)
	}
}

func FromAny(src *anypb.Any, spec resources.ResourceSpec) error {
	if msg, ok := spec.(proto.Message); ok {
		return utilproto.UnmarshalAnyTo(src, msg)
	} else {
		return json.Unmarshal(src.Value, spec)
	}
}

func FullName(spec resources.ResourceSpec) string {
	specType := reflect.TypeOf(spec).Elem()
	return path.Join(specType.PkgPath(), specType.Name())
}

func Equal(x, y resources.ResourceSpec) bool {
	xMsg, xOk := x.(proto.Message)
	yMsg, yOk := y.(proto.Message)
	if xOk != yOk {
		return false
	}

	if xOk {
		return proto.Equal(xMsg, yMsg)
	} else {
		return reflect.DeepEqual(x, y)
	}
}

func IsEmpty(spec resources.ResourceSpec) bool {
	if msg, ok := spec.(proto.Message); ok {
		return proto.Size(msg) == 0
	} else {
		return reflect.ValueOf(spec).Elem().IsZero()
	}
}
