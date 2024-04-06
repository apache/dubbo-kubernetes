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
	"time"
)

import (
	"github.com/pkg/errors"
)

import (
	system_proto "github.com/apache/dubbo-kubernetes/api/system/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
)

var heartbeatLog = core.Log.WithName("intercp").WithName("catalog").WithName("heartbeat")

type heartbeatComponent struct {
	catalog     Catalog
	getClientFn GetClientFn
	request     *system_proto.PingRequest
	interval    time.Duration

	leader *Instance
}

var _ component.Component = &heartbeatComponent{}

type GetClientFn = func(url string) (system_proto.InterCpPingServiceClient, error)

func NewHeartbeatComponent(
	catalog Catalog,
	instance Instance,
	interval time.Duration,
	newClientFn GetClientFn,
) (component.Component, error) {
	return &heartbeatComponent{
		catalog: catalog,
		request: &system_proto.PingRequest{
			InstanceId:  instance.Id,
			Address:     instance.Address,
			InterCpPort: uint32(instance.InterCpPort),
		},
		getClientFn: newClientFn,
		interval:    interval,
	}, nil
}

func (h *heartbeatComponent) Start(stop <-chan struct{}) error {
	heartbeatLog.Info("starting heartbeats to a leader")
	ticker := time.NewTicker(h.interval)
	ctx := context.Background()

	for {
		select {
		case <-ticker.C:
			if !h.heartbeat(ctx, true) {
				continue
			}
		case <-stop:
			// send final heartbeat to gracefully signal that the instance is going down
			_ = h.heartbeat(ctx, false)
			return nil
		}
	}
}

func (h *heartbeatComponent) heartbeat(ctx context.Context, ready bool) bool {
	heartbeatLog := heartbeatLog.WithValues(
		"instanceId", h.request.InstanceId,
		"ready", ready,
	)
	if h.leader == nil {
		if err := h.connectToLeader(ctx); err != nil {
			heartbeatLog.Error(err, "could not connect to leader")
			return false
		}
	}
	if h.leader.Id == h.request.InstanceId {
		heartbeatLog.V(1).Info("this instance is a leader. No need to send a heartbeat.")
		return true
	}
	heartbeatLog = heartbeatLog.WithValues(
		"leaderAddress", h.leader.Address,
	)
	heartbeatLog.V(1).Info("sending a heartbeat to a leader")
	h.request.Ready = ready
	client, err := h.getClientFn(h.leader.InterCpURL())
	if err != nil {
		heartbeatLog.Error(err, "could not get or create a client to a leader")
		h.leader = nil
		return false
	}
	resp, err := client.Ping(ctx, h.request)
	if err != nil {
		heartbeatLog.Error(err, "could not send a heartbeat to a leader")
		h.leader = nil
		return false
	}
	if !resp.Leader {
		heartbeatLog.V(1).Info("instance responded that it is no longer a leader")
		h.leader = nil
	}
	return true
}

func (h *heartbeatComponent) connectToLeader(ctx context.Context) error {
	newLeader, err := Leader(ctx, h.catalog)
	if err != nil {
		return err
	}
	h.leader = &newLeader
	if h.leader.Id == h.request.InstanceId {
		return nil
	}
	heartbeatLog.Info("leader has changed. Creating connection to the new leader.",
		"previousLeaderAddress", h.leader.Address,
		"newLeaderAddress", newLeader.Leader,
	)
	_, err = h.getClientFn(h.leader.InterCpURL())
	if err != nil {
		return errors.Wrap(err, "could not create a client to a leader")
	}
	return nil
}

func (h *heartbeatComponent) NeedLeaderElection() bool {
	return false
}
