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

package dubbo

import (
	envoy_admin_v3 "github.com/envoyproxy/go-control-plane/envoy/admin/v3"
)

func Sanitize(configDump *envoy_admin_v3.ConfigDump) error {
	for _, config := range configDump.Configs {
		if config.MessageIs(&envoy_admin_v3.BootstrapConfigDump{}) {
			bootstrapConfigDump := &envoy_admin_v3.BootstrapConfigDump{}
			if err := config.UnmarshalTo(bootstrapConfigDump); err != nil {
				return err
			}

			for _, grpcService := range bootstrapConfigDump.GetBootstrap().GetDynamicResources().GetAdsConfig().GetGrpcServices() {
				for i, initMeta := range grpcService.InitialMetadata {
					if initMeta.Key == "authorization" {
						grpcService.InitialMetadata[i].Value = "[redacted]"
					}
				}
			}

			for _, grpcService := range bootstrapConfigDump.GetBootstrap().GetHdsConfig().GetGrpcServices() {
				for i, initMeta := range grpcService.InitialMetadata {
					if initMeta.Key == "authorization" {
						grpcService.InitialMetadata[i].Value = "[redacted]"
					}
				}
			}

			if err := config.MarshalFrom(bootstrapConfigDump); err != nil {
				return err
			}
		}
	}
	return nil
}
