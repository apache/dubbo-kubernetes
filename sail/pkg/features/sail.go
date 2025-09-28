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
	MultiRootMesh = env.Register("DUBBO_MULTIROOT_MESH", false,
		"If enabled, mesh will support certificates signed by more than one trustAnchor for DUBBO_MUTUAL mTLS").Get()
	NaviCertProvider = env.Register("PILOT_CERT_PROVIDER", constants.CertProviderDubbod,
		"The provider of Navi DNS certificate. K8S RA will be used for k8s.io/NAME. 'dubbod' value will sign"+
			" using Navi build in CA. Other values will not not generate TLS certs, but still "+
			" distribute ./etc/certs/root-cert.pem. Only used if custom certificates are not mounted.").Get()
	InformerWatchNamespace = env.Register("DUBBO_WATCH_NAMESPACE", "",
		"If set, limit Kubernetes watches to a single namespace. "+
			"Warning: only a single namespace can be set.").Get()
	ClusterName = env.Register("CLUSTER_ID", constants.DefaultClusterName,
		"Defines the cluster and service registry that this Dubbod instance belongs to").Get()
	EnableVtprotobuf = env.Register("ENABLE_VTPROTOBUF", true,
		"If true, will use optimized vtprotobuf based marshaling. Requires a build with -tags=vtprotobuf.").Get()
	KubernetesClientContentType = env.Register("DUBBO_KUBE_CLIENT_CONTENT_TYPE", "protobuf",
		"The content type to use for Kubernetes clients. Defaults to protobuf. Valid options: [protobuf, json]").Get()
	EnableCAServer = env.Register("ENABLE_CA_SERVER", true,
		"If this is set to false, will not create CA server in dubbod.").Get()
	// EnableCACRL ToDo (nilekh): remove this feature flag once it's stable
	EnableCACRL = env.Register(
		"SHIP_ENABLE_CA_CRL",
		true, // Default value (true = feature enabled by default)
		"If set to false, Dubbo will not watch for the ca-crl.pem file in the /etc/cacerts directory "+
			"and will not distribute CRL data to namespaces for proxies to consume.",
	).Get()
	SailCertProvider = env.Register("SAIL_CERT_PROVIDER", constants.CertProviderDubbod,
		"The provider of Pilot DNS certificate. K8S RA will be used for k8s.io/NAME. 'dubbod' value will sign"+
			" using Dubbo build in CA. Other values will not not generate TLS certs, but still "+
			" distribute ./etc/certs/root-cert.pem. Only used if custom certificates are not mounted.").Get()
	DubbodServiceCustomHost = env.Register("DUBBOD_CUSTOM_HOST", "",
		"Custom host name of dubbod that dubbod signs the server cert. "+
			"Multiple custom host names are supported, and multiple values are separated by commas.").Get()
)
