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
	"os"
	"path/filepath"
	"strings"

	dubboagent "github.com/apache/dubbo-kubernetes/pkg/dubbo-agent"
	meshconfig "istio.io/api/mesh/v1alpha1"
)

const xdsHeaderPrefix = "XDS_HEADER_"

func NewAgentOptions(proxy *ProxyArgs, cfg *meshconfig.ProxyConfig, sds dubboagent.SDSServiceFactory) *dubboagent.AgentOptions {
	o := &dubboagent.AgentOptions{
		XDSRootCerts:               xdsRootCA,
		CARootCerts:                caRootCA,
		XDSHeaders:                 map[string]string{},
		XdsUdsPath:                 filepath.Join(cfg.ConfigPath, "XDS"),
		GRPCBootstrapPath:          grpcBootstrapEnv,
		ProxyType:                  proxy.Type,
		ServiceNode:                proxy.ServiceNode(),
		EnableDynamicProxyConfig:   enableProxyConfigXdsEnv,
		SDSFactory:                 sds,
		WorkloadIdentitySocketFile: workloadIdentitySocketFile,
		ProxyIPAddresses:           proxy.IPAddresses,
		ProxyDomain:                proxy.DNSDomain,
		DubbodSAN:                  dubbodSAN.Get(),
	}
	extractXDSHeadersFromEnv(o)
	return o
}

func extractXDSHeadersFromEnv(o *dubboagent.AgentOptions) {
	envs := os.Environ()
	for _, e := range envs {
		if strings.HasPrefix(e, xdsHeaderPrefix) {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) != 2 {
				continue
			}
			o.XDSHeaders[parts[0][len(xdsHeaderPrefix):]] = parts[1]
		}
	}
}
