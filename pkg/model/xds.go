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

package model

const (
	APITypePrefix              = "type.googleapis.com/"
	ClusterType                = APITypePrefix + "envoy.config.cluster.v3.Cluster"
	EndpointType               = APITypePrefix + "envoy.config.endpoint.v3.ClusterLoadAssignment"
	ListenerType               = APITypePrefix + "envoy.config.listener.v3.Listener"
	RouteType                  = APITypePrefix + "envoy.config.route.v3.RouteConfiguration"
	SecretType                 = APITypePrefix + "envoy.extensions.transport_sockets.tls.v3.Secret"
	ExtensionConfigurationType = APITypePrefix + "envoy.config.core.v3.TypedExtensionConfig"

	NameTableType   = APITypePrefix + "dubbo.networking.nds.v1.NameTable"
	HealthInfoType  = APITypePrefix + "dubbo.v1.HealthInformation"
	ProxyConfigType = APITypePrefix + "dubbo.mesh.v1alpha1.ProxyConfig"
	DebugType       = "dubbo.io/debug"
	AddressType     = APITypePrefix + "dubbo.workload.Address"
	WorkloadType    = APITypePrefix + "dubbo.workload.Workload"
)

func GetShortType(typeURL string) string {
	switch typeURL {
	case ClusterType:
		return "CDS"
	case ListenerType:
		return "LDS"
	case RouteType:
		return "RDS"
	case EndpointType:
		return "EDS"
	case SecretType:
		return "SDS"
	case NameTableType:
		return "NDS"
	case ProxyConfigType:
		return "PCDS"
	case ExtensionConfigurationType:
		return "ECDS"
	case AddressType, WorkloadType:
		return "WDS"
	default:
		return typeURL
	}
}
