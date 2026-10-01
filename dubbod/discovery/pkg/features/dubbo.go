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

package features

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/env"
)

var (
	ValidationWebhookConfigName = env.Register("VALIDATION_WEBHOOK_CONFIG_NAME", "dubbo-dubbo-system",
		"If not empty, the controller will automatically patch validatingwebhookconfiguration when the CA certificate changes. "+
			"Only works in kubernetes environment.").Get()
	SharedMeshConfig = env.Register("SHARED_MESH_CONFIG", "",
		"Additional config map to load for shared MeshConfig settings. The standard mesh config will take precedence.").Get()

	ClusterName = env.Register("CLUSTER_ID", constants.DefaultClusterName,
		"Defines the cluster and service registry that this Dubbod instance belongs to").Get()
	EnableVtprotobuf = env.Register("ENABLE_VTPROTOBUF", true,
		"If true, will use optimized vtprotobuf based marshaling. Requires a build with -tags=vtprotobuf.").Get()

	EnableCAServer = env.Register("ENABLE_CA_SERVER", true,
		"If this is set to false, will not create CA server in dubbod.").Get()
	// EnableCACRL ToDo (nilekh): remove this feature flag once it's stable
	EnableCACRL = env.Register(
		"DUBBO_ENABLE_CA_CRL",
		true, // Default value (true = feature enabled by default)
		"If set to false, Dubbo will not watch for the ca-crl.pem file in the /etc/cacerts directory "+
			"and will not distribute CRL data to namespaces for proxies to consume.",
	).Get()
	DubboCertProvider = env.Register("DUBBO_CERT_PROVIDER", constants.CertProviderDubbod,
		"The provider of Dubbo DNS certificate. K8S RA will be used for k8s.io/NAME. 'dubbod' value will sign"+
			" using Dubbo build in CA. Other values will not not generate TLS certs, but still "+
			" distribute ./etc/certs/root-cert.pem. Only used if custom certificates are not mounted.").Get()
	DubbodServiceCustomHost = env.Register("DUBBOD_CUSTOM_HOST", "",
		"Custom host name of dubbod that dubbod signs the server cert. "+
			"Multiple custom host names are supported, and multiple values are separated by commas.").Get()
	InjectionWebhookConfigName = env.Register("INJECTION_WEBHOOK_CONFIG_NAME", "dubbo-inherent-injector",
		"Name of the mutatingwebhookconfiguration to patch, if dubboctl is not used.").Get()
	ManagedGatewayController = env.Register("DUBBO_GATEWAY_API_CONTROLLER_NAME", "dubbo.apache.org/gateway-controller",
		"Gateway API controller name. dubbod will only reconcile Gateway API resources referencing a GatewayClass with this controller name").Get()
	GatewayAPIDefaultGatewayClass = env.Register("DUBBO_GATEWAY_API_DEFAULT_GATEWAYCLASS_NAME", "dubbo",
		"Name of the default GatewayClass").Get()
	TransitImage = env.Register("DUBBO_TRANSIT_IMAGE", "dubml/transit:latest",
		"Container image used for managed Dubbo Gateway API data-plane deployments").Get()
	TransitReplicas = env.Register("DUBBO_TRANSIT_REPLICAS", 2,
		"Default replica count for managed Dubbo Gateway API data-plane deployments. Two or more keeps a"+
			" gateway serving while one replica is drained; individual gateways override it with the"+
			" gateway.dubbo.apache.org/replicas annotation").Get()
	StatusMaxWorkers = env.Register("DUBBO_STATUS_MAX_WORKERS", 100, "The maximum number of workers"+
		" for status update").Get()
)
