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
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"reflect"
	"time"

	"google.golang.org/protobuf/reflect/protoreflect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ApiTypePrefix         = "type.googleapis.com/"
	AuthenticationTypeUrl = ApiTypePrefix + "dubbo.apache.org.v1alpha1.AuthenticationPolicyToClient"
	AuthorizationTypeUrl  = ApiTypePrefix + "dubbo.apache.org.v1alpha1.AuthorizationPolicyToClient"
	TagRouteTypeUrl       = ApiTypePrefix + "dubbo.apache.org.v1alpha1.TagRouteToClient"
	DynamicConfigTypeUrl  = ApiTypePrefix + "dubbo.apache.org.v1alpha1.DynamicConfigToClient"
	ServiceMappingTypeUrl = ApiTypePrefix + "dubbo.apache.org.v1alpha1.ServiceNameMappingToClient"
	ConditionRouteTypeUrl = ApiTypePrefix + "dubbo.apache.org.v1alpha1.ConditionRouteToClient"
)

// Meta is metadata attached to each configuration unit.
// The revision is optional, and if provided, identifies the
// last update operation on the object.
type Meta struct {
	// GroupVersionKind is a short configuration name that matches the content message type
	// (e.g. "route-dds")
	GroupVersionKind GroupVersionKind `json:"type,omitempty"`

	// UID
	UID string `json:"uid,omitempty"`

	// Name is a unique immutable identifier in a namespace
	Name string `json:"name,omitempty"`

	// Namespace defines the space for names (optional for some types),
	// applications may choose to use namespaces for a variety of purposes
	// (security domains, fault domains, organizational domains)
	Namespace string `json:"namespace,omitempty"`

	// Domain defines the suffix of the fully qualified name past the namespace.
	// Domain is not a part of the unique key unlike name and namespace.
	Domain string `json:"domain,omitempty"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	Annotations map[string]string `json:"annotations,omitempty"`

	// ResourceVersion is an opaque identifier for tracking updates to the config registry.
	// The implementation may use a change index or a commit log for the revision.
	// The config client should not make any assumptions about revisions and rely only on
	// exact equality to implement optimistic concurrency of read-write operations.
	//
	// The lifetime of an object of a particular revision depends on the underlying data store.
	// The data store may compaction old revisions in the interest of storage optimization.
	//
	// An empty revision carries a special meaning that the associated object has
	// not been stored and assigned a revision.
	ResourceVersion string `json:"resourceVersion,omitempty"`

	// CreationTimestamp records the creation time
	CreationTimestamp time.Time `json:"creationTimestamp,omitempty"`

	// OwnerReferences allows specifying in-namespace owning objects.
	OwnerReferences []metav1.OwnerReference `json:"ownerReferences,omitempty"`

	// A sequence number representing a specific generation of the desired state. Populated by the system. Read-only.
	Generation int64 `json:"generation,omitempty"`
}

// Config is a configuration unit consisting of the type of configuration, the
// key identifier that is unique per type, and the content represented as a
// protobuf message.
type Config struct {
	Meta

	// Spec holds the configuration object as a gogo protobuf message
	Spec Spec
}

// Spec defines the spec for the config.
type Spec interface{}

func ToProto(s Spec) (*anypb.Any, error) {
	if pb, ok := s.(protoreflect.ProtoMessage); ok {
		return MessageToAnyWithError(pb)
	}

	return nil, nil
}

// MessageToAnyWithError converts from proto message to proto Any
func MessageToAnyWithError(msg proto.Message) (*anypb.Any, error) {
	b, err := marshal(msg)
	if err != nil {
		return nil, err
	}
	return &anypb.Any{
		TypeUrl: "type.googleapis.com/" + string(msg.ProtoReflect().Descriptor().FullName()),
		Value:   b,
	}, nil
}

func marshal(msg proto.Message) ([]byte, error) {
	return proto.MarshalOptions{
		Deterministic: true,
	}.Marshal(msg)
}

type deepCopier interface {
	DeepCopyInterface() interface{}
}

func DeepCopy(s interface{}) interface{} {
	// If deep copy is defined, use that
	if dc, ok := s.(deepCopier); ok {
		return dc.DeepCopyInterface()
	}

	if _, ok := s.(protoreflect.ProtoMessage); ok {
		if pb, ok := s.(proto.Message); ok {
			return proto.Clone(pb)
		}
	}

	// If we don't have a deep copy method, we will have to do some reflection magic.
	js, err := json.Marshal(s)
	if err != nil {
		return nil
	}

	data := reflect.New(reflect.TypeOf(s).Elem()).Interface()
	err = json.Unmarshal(js, &data)
	if err != nil {
		return nil
	}
	return data
}

// Key function for the configuration objects
func Key(typ, name, namespace string) string {
	return fmt.Sprintf("%s/%s/%s", typ, namespace, name)
}

// Key is the unique identifier for a configuration object
// TODO: this is *not* unique - needs the version and group
func (meta *Meta) Key() string {
	return Key(meta.GroupVersionKind.Kind, meta.Name, meta.Namespace)
}

func (c *Config) DeepCopy() Config {
	var clone Config
	clone.Meta = c.Meta
	if c.Labels != nil {
		clone.Labels = make(map[string]string)
		for k, v := range c.Labels {
			clone.Labels[k] = v
		}
	}
	if c.Annotations != nil {
		clone.Annotations = make(map[string]string)
		for k, v := range c.Annotations {
			clone.Annotations[k] = v
		}
	}
	clone.Spec = DeepCopy(c.Spec)
	return clone
}

var _ fmt.Stringer = GroupVersionKind{}

type GroupVersionKind struct {
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

func (g GroupVersionKind) String() string {
	return g.CanonicalGroup() + "/" + g.Version + "/" + g.Kind
}

// GroupVersion returns the group/version similar to what would be found in the apiVersion field of a Kubernetes resource.
func (g GroupVersionKind) GroupVersion() string {
	if g.Group == "" {
		return g.Version
	}
	return g.Group + "/" + g.Version
}

// CanonicalGroup returns the group with defaulting applied. This means an empty group will
// be treated as "core", following Kubernetes API standards
func (g GroupVersionKind) CanonicalGroup() string {
	if g.Group != "" {
		return g.Group
	}
	return "core"
}
