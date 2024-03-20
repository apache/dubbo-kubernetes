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

package envoy

import (
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"

	"google.golang.org/protobuf/types/known/structpb"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tags"
)

func EndpointMetadata(tags tags.Tags) *envoy_core.Metadata {
	tags = tags.WithoutTags(mesh_proto.ServiceTag) // service name is already in cluster name, we don't need it in metadata
	if len(tags) == 0 {
		return nil
	}
	fields := MetadataFields(tags)
	return &envoy_core.Metadata{
		FilterMetadata: map[string]*structpb.Struct{
			"envoy.lb": {
				Fields: fields,
			},
			"envoy.transport_socket_match": {
				Fields: fields,
			},
		},
	}
}

func LbMetadata(tags tags.Tags) *envoy_core.Metadata {
	tags = tags.WithoutTags(mesh_proto.ServiceTag) // service name is already in cluster name, we don't need it in metadata
	if len(tags) == 0 {
		return nil
	}
	fields := MetadataFields(tags)
	return &envoy_core.Metadata{
		FilterMetadata: map[string]*structpb.Struct{
			"envoy.lb": {
				Fields: fields,
			},
		},
	}
}

func MetadataFields(tags tags.Tags) map[string]*structpb.Value {
	fields := map[string]*structpb.Value{}
	for key, value := range tags {
		fields[key] = &structpb.Value{
			Kind: &structpb.Value_StringValue{
				StringValue: value,
			},
		}
	}
	return fields
}

const (
	TagsKey   = "io.dubbo.tags"
	LbTagsKey = "envoy.lb"
)

func ExtractTags(metadata *envoy_core.Metadata) tags.Tags {
	tags := tags.Tags{}
	for key, value := range metadata.GetFilterMetadata()[TagsKey].GetFields() {
		tags[key] = value.GetStringValue()
	}
	return tags
}

func ExtractLbTags(metadata *envoy_core.Metadata) tags.Tags {
	tags := tags.Tags{}
	for key, value := range metadata.GetFilterMetadata()[LbTagsKey].GetFields() {
		tags[key] = value.GetStringValue()
	}
	return tags
}
