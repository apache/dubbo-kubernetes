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

package model

const (
	APITypePrefix = "type.googleapis.com/"
	ClusterType   = APITypePrefix + "cluster.v1.Cluster"
	EndpointType  = APITypePrefix + "endpoint.v1.ClusterLoadAssignment"
	ListenerType  = APITypePrefix + "listener.v1.Listener"
	RouteType     = APITypePrefix + "route.v1.RouteConfiguration"
	SecretType    = APITypePrefix + "extensions.transport_sockets.tls.v1.Secret"

	HealthInfoType  = APITypePrefix + "dubbo.v1.HealthInformation"
	ProxyConfigType = APITypePrefix + "dubbo.mesh.v1alpha1.ProxyConfig"
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
	case ProxyConfigType:
		return "PCDS"
	default:
		return typeURL
	}
}

func GetMetricType(typeURL string) string {
	switch typeURL {
	case ClusterType:
		return "cds"
	case ListenerType:
		return "lds"
	case RouteType:
		return "rds"
	case EndpointType:
		return "eds"
	case SecretType:
		return "sds"
	case ProxyConfigType:
		return "pcds"
	default:
		return typeURL
	}
}
