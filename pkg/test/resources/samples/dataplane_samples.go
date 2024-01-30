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

package samples

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/test/resources/builders"
)

func DataplaneBackendBuilder() *builders.DataplaneBuilder {
	return builders.Dataplane().
		WithAddress("192.168.0.1").
		WithServices("backend")
}

func DataplaneBackend() *mesh.DataplaneResource {
	return DataplaneBackendBuilder().Build()
}

func DataplaneWebBuilder() *builders.DataplaneBuilder {
	return builders.Dataplane().
		WithName("web-01").
		WithAddress("192.168.0.2").
		WithInboundOfTags(mesh_proto.ServiceTag, "web", mesh_proto.ProtocolTag, "http").
		AddOutboundToService("backend")
}

func DataplaneWeb() *mesh.DataplaneResource {
	return DataplaneWebBuilder().Build()
}

func IgnoredDataplaneBackendBuilder() *builders.DataplaneBuilder {
	return DataplaneBackendBuilder().With(func(resource *mesh.DataplaneResource) {
		resource.Spec.Networking.Inbound[0].State = mesh_proto.Dataplane_Networking_Inbound_Ignored
	})
}
