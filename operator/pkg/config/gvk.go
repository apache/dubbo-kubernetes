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

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	gogojsonpb "github.com/gogo/protobuf/jsonpb"
	gogoproto "github.com/gogo/protobuf/proto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func (g GroupVersionKind) Kubernetes() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   g.Group,
		Version: g.Version,
		Kind:    g.Kind,
	}
}

func CanoncalGroup(group string) string {
	if group != "" {
		return group
	}
	return "core"
}

func FromK8sGVK(gvk schema.GroupVersionKind) GroupVersionKind {
	return GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind,
	}
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
