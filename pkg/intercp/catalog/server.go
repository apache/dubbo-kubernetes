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

package catalog

import (
	"context"
)

import (
	system_proto "github.com/apache/dubbo-kubernetes/api/system/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
)

var serverLog = core.Log.WithName("intercp").WithName("catalog").WithName("server")

type server struct {
	heartbeats *Heartbeats
	leaderInfo component.LeaderInfo

	system_proto.UnimplementedInterCpPingServiceServer
}

var _ system_proto.InterCpPingServiceServer = &server{}

func NewServer(heartbeats *Heartbeats, leaderInfo component.LeaderInfo) system_proto.InterCpPingServiceServer {
	return &server{
		heartbeats: heartbeats,
		leaderInfo: leaderInfo,
	}
}

func (s *server) Ping(_ context.Context, request *system_proto.PingRequest) (*system_proto.PingResponse, error) {
	serverLog.V(1).Info("received ping", "instanceID", request.InstanceId, "address", request.Address, "ready", request.Ready)
	instance := Instance{
		Id:          request.InstanceId,
		Address:     request.Address,
		InterCpPort: uint16(request.InterCpPort),
		Leader:      false,
	}
	if request.Ready {
		s.heartbeats.Add(instance)
	} else {
		s.heartbeats.Remove(instance)
	}
	return &system_proto.PingResponse{
		Leader: s.leaderInfo.IsLeader(),
	}, nil
}
