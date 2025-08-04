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

package features

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/env"
)

var (
	SharedMeshConfig = env.Register("SHARED_MESH_CONFIG", "",
		"Additional config map to load for shared MeshConfig settings. The standard mesh config will take precedence.").Get()
	EnableUnsafeAssertions = env.Register(
		"UNSAFE_NAVIGATOR_ENABLE_RUNTIME_ASSERTIONS",
		false,
		"If enabled, addition runtime asserts will be performed. "+
			"These checks are both expensive and panic on failure. As a result, this should be used only for testing.",
	).Get()
	InformerWatchNamespace = env.Register("DUBBO_WATCH_NAMESPACE", "",
		"If set, limit Kubernetes watches to a single namespace. "+
			"Warning: only a single namespace can be set.").Get()
	ClusterName = env.Register("CLUSTER_ID", constants.DefaultClusterName,
		"Defines the cluster and service registry that this Dubbod instance belongs to").Get()
	EnableVtprotobuf = env.Register("ENABLE_VTPROTOBUF", true,
		"If true, will use optimized vtprotobuf based marshaling. Requires a build with -tags=vtprotobuf.").Get()
	KubernetesClientContentType = env.Register("DUBBO_KUBE_CLIENT_CONTENT_TYPE", "protobuf",
		"The content type to use for Kubernetes clients. Defaults to protobuf. Valid options: [protobuf, json]").Get()
)
