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

package xds

import (
	"google.golang.org/protobuf/types/known/structpb"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model/rest"
)

var metadataLog = core.Log.WithName("xds-server").WithName("metadata-tracker")

const (
	FieldDataplaneProxyType         = "dataplane.proxyType"
	FieldDataplaneDataplaneResource = "dataplane.resource"
)

type DataplaneMetadata struct {
	Resource        model.Resource
	DynamicMetadata map[string]string
	ProxyType       mesh_proto.ProxyType
}

func (m *DataplaneMetadata) GetProxyType() mesh_proto.ProxyType {
	if m == nil || m.ProxyType == "" {
		return mesh_proto.DataplaneProxyType
	}
	return m.ProxyType
}

func DataplaneMetadataFromXdsMetadata(xdsMetadata *structpb.Struct, tmpDir string, dpKey model.ResourceKey) *DataplaneMetadata {
	metadata := DataplaneMetadata{}
	if xdsMetadata == nil {
		return &metadata
	}
	if field := xdsMetadata.Fields[FieldDataplaneProxyType]; field != nil {
		metadata.ProxyType = mesh_proto.ProxyType(field.GetStringValue())
	}
	if value := xdsMetadata.Fields[FieldDataplaneDataplaneResource]; value != nil {
		res, err := rest.YAML.UnmarshalCore([]byte(value.GetStringValue()))
		if err != nil {
			metadataLog.Error(err, "invalid value in dataplane metadata", "field", FieldDataplaneDataplaneResource, "value", value)
		} else {
			switch r := res.(type) {
			default:
				metadataLog.Error(err, "invalid dataplane resource type",
					"resource", r.Descriptor().Name,
					"field", FieldDataplaneDataplaneResource,
					"value", value)
			}
		}
	}
	return &metadata
}
