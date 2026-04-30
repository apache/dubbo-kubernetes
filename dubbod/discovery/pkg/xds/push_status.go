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

package xds

import (
	"encoding/json"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	v1 "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds/v1"
)

type pushStatusReport struct {
	PushVersion      string           `json:"pushVersion"`
	Uptime           string           `json:"uptime"`
	Services         int              `json:"services"`
	EndpointServices int              `json:"endpointServices"`
	Clients          pushClientStatus `json:"clients"`
	Queue            pushQueueStats   `json:"queue"`
	Updates          pushUpdateStatus `json:"updates"`
	ProxyStatus      json.RawMessage  `json:"proxyStatus"`
}

type pushClientStatus struct {
	ConnectedEndpoints       int `json:"connectedEndpoints"`
	ProxylessGRPC            int `json:"proxylessGrpc"`
	ProxylessGRPCEDSWatchers int `json:"proxylessGrpcEdsWatchers"`
	ProxylessGRPCUnknownEDS  int `json:"proxylessGrpcUnknownEds"`
	WatchedEDSResources      int `json:"watchedEdsResources"`
}

type pushUpdateStatus struct {
	Inbound   int64 `json:"inbound"`
	Committed int64 `json:"committed"`
}

func (s *DiscoveryServer) pushStatusJSON(push *model.PushContext) ([]byte, error) {
	return json.Marshal(s.pushStatusReport(push))
}

func (s *DiscoveryServer) pushStatusReport(push *model.PushContext) pushStatusReport {
	proxyStatus := json.RawMessage([]byte("{}"))
	if push != nil {
		if data, err := push.StatusJSON(); err == nil && string(data) != "null" {
			proxyStatus = json.RawMessage(data)
		}
	}
	updates := pushUpdateStatus{}
	if s.InboundUpdates != nil {
		updates.Inbound = s.InboundUpdates.Load()
	}
	if s.CommittedUpdates != nil {
		updates.Committed = s.CommittedUpdates.Load()
	}

	report := pushStatusReport{
		PushVersion:      "unknown",
		Uptime:           time.Since(s.DiscoveryStartTime).String(),
		Clients:          summarizePushClients(s.AllClients()),
		Queue:            s.pushQueue.Stats(),
		Updates:          updates,
		ProxyStatus:      proxyStatus,
		EndpointServices: endpointServiceCount(s.Env),
	}
	if push != nil {
		report.PushVersion = push.PushVersion
		report.Services = len(push.GetAllServices())
	}
	return report
}

func summarizePushClients(clients []*Connection) pushClientStatus {
	out := pushClientStatus{ConnectedEndpoints: len(clients)}
	for _, con := range clients {
		if con == nil || con.proxy == nil {
			continue
		}
		con.proxy.RLock()
		isProxylessGRPC := con.proxy.Metadata != nil && con.proxy.Metadata.Generator == "grpc"
		watched := con.proxy.WatchedResources[v1.EndpointType]
		if isProxylessGRPC {
			out.ProxylessGRPC++
			if watched == nil {
				out.ProxylessGRPCUnknownEDS++
			} else {
				out.ProxylessGRPCEDSWatchers++
				out.WatchedEDSResources += len(watched.ResourceNames)
			}
		}
		con.proxy.RUnlock()
	}
	return out
}

func endpointServiceCount(env *model.Environment) int {
	if env == nil || env.EndpointIndex == nil {
		return 0
	}
	return len(env.EndpointIndex.AllServices())
}
