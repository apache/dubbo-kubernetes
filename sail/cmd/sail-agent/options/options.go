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

package options

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/env"
	"github.com/apache/dubbo-kubernetes/pkg/jwt"
	"github.com/apache/dubbo-kubernetes/pkg/security"
)

var (
	ProxyConfigEnv = env.Register(
		"PROXY_CONFIG",
		"",
		"The proxy configuration. This will be set by the injection - gateways will use file mounts.",
	).Get()
	dubbodSAN = env.Register("DUBBOD_SAN", "",
		"Override the ServerName used to validate Istiod certificate. "+
			"Can be used as an alternative to setting /etc/hosts for VMs - discovery address will be an IP:port")
	jwtPolicy = env.Register("JWT_POLICY", jwt.PolicyThirdParty,
		"The JWT validation policy.")
	workloadIdentitySocketFile = env.Register("WORKLOAD_IDENTITY_SOCKET_FILE", security.DefaultWorkloadIdentitySocketFile,
		fmt.Sprintf("SPIRE workload identity SDS socket filename. If set, an SDS socket with this name must exist at %s", security.WorkloadIdentityPath)).Get()
	credFetcherTypeEnv = env.Register("CREDENTIAL_FETCHER_TYPE", security.JWT,
		"The type of the credential fetcher. Currently supported types include GoogleComputeEngine").Get()
	credIdentityProvider = env.Register("CREDENTIAL_IDENTITY_PROVIDER", "GoogleComputeEngine",
		"The identity provider for credential. Currently default supported identity provider is GoogleComputeEngine").Get()
	caProviderEnv = env.Register("CA_PROVIDER", "Aegis", "name of authentication provider").Get()
	caEndpointEnv = env.Register("CA_ADDR", "", "Address of the spiffe certificate provider. Defaults to discoveryAddress").Get()
)
