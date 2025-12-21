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

import "github.com/apache/dubbo-kubernetes/pkg/env"

var (
	CACertConfigMapName = env.Register("PLANET_CA_CERT_CONFIGMAP", "dubbo-ca-root-cert",
		"The name of the ConfigMap that stores the Root CA Certificate that is used by dubbod").Get()
	EnableLeaderElection = env.Register("ENABLE_LEADER_ELECTION", true,
		"If enabled (default), starts a leader election client and gains leadership before executing controllers. "+
			"If false, it assumes that only one instance of dubbod is running and skips leader election.").Get()
	EnableEnhancedSubsetRuleMerge = env.Register("ENABLE_ENHANCED_DESTINATIONRULE_MERGE", true,
		"If enabled, Dubbo merge subsetrules considering their exportTo fields,"+
			" they will be kept as independent rules if the exportTos are not equal.").Get()
	EnableGatewayAPI = env.Register("PLANET_ENABLE_GATEWAY_API", true,
		"If this is set to true, support for Kubernetes gateway-api (github.com/kubernetes-sigs/gateway-api) will "+
			" be enabled. In addition to this being enabled, the gateway-api CRDs need to be installed.").Get()
	EnableGatewayAPIStatus = env.Register("PLANET_ENABLE_GATEWAY_API_STATUS", true,
		"If this is set to true, gateway-api resources will have status written to them").Get()
	EnableGatewayAPIGatewayClassController = env.Register("PLANET_ENABLE_GATEWAY_API_GATEWAYCLASS_CONTROLLER", true,
		"If this is set to true, dubbod will create and manage its default GatewayClasses").Get()
	EnableGatewayAPIDeploymentController = env.Register("PLANET_ENABLE_GATEWAY_API_DEPLOYMENT_CONTROLLER", true,
		"If this is set to true, gateway-api resources will automatically provision in cluster deployment, services, etc").Get()
)
