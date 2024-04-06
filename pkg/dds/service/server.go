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

package service

import (
	"context"
	"fmt"
	"time"
)

import (
	"github.com/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	system_proto "github.com/apache/dubbo-kubernetes/api/system/v1alpha1"
	config_store "github.com/apache/dubbo-kubernetes/pkg/config/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/dds/util"
	"github.com/apache/dubbo-kubernetes/pkg/events"
)

var log = core.Log.WithName("dds-service")

type StreamInterceptor interface {
	InterceptServerStream(stream grpc.ServerStream) error
}

type GlobalDDSServiceServer struct {
	resManager              manager.ResourceManager
	instanceID              string
	filters                 []StreamInterceptor
	extensions              context.Context
	upsertCfg               config_store.UpsertConfig
	eventBus                events.EventBus
	zoneHealthCheckInterval time.Duration
	mesh_proto.UnimplementedGlobalDDSServiceServer
	context context.Context
}

func NewGlobalDDSServiceServer(
	ctx context.Context,
	resManager manager.ResourceManager,
	instanceID string, filters []StreamInterceptor,
	extensions context.Context,
	upsertCfg config_store.UpsertConfig,
	eventBus events.EventBus,
	zoneHealthCheckInterval time.Duration,
) *GlobalDDSServiceServer {
	return &GlobalDDSServiceServer{
		context:                 ctx,
		resManager:              resManager,
		instanceID:              instanceID,
		filters:                 filters,
		extensions:              extensions,
		upsertCfg:               upsertCfg,
		eventBus:                eventBus,
		zoneHealthCheckInterval: zoneHealthCheckInterval,
	}
}

func (g *GlobalDDSServiceServer) HealthCheck(ctx context.Context, _ *mesh_proto.ZoneHealthCheckRequest) (*mesh_proto.ZoneHealthCheckResponse, error) {
	zone, err := util.ClientIDFromIncomingCtx(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	zoneID := ZoneClientIDFromCtx(ctx, zone)
	log := log.WithValues("clientID", zoneID.String())

	insight := system.NewZoneInsightResource()
	if err := manager.Upsert(ctx, g.resManager, model.ResourceKey{Name: zone, Mesh: model.NoMesh}, insight, func(resource model.Resource) error {
		if insight.Spec.HealthCheck == nil {
			insight.Spec.HealthCheck = &system_proto.HealthCheck{}
		}

		insight.Spec.HealthCheck.Time = timestamppb.Now()
		return nil
	}, manager.WithConflictRetry(
		g.upsertCfg.ConflictRetryBaseBackoff.Duration, g.upsertCfg.ConflictRetryMaxTimes, g.upsertCfg.ConflictRetryJitterPercent,
	)); err != nil && !errors.Is(err, context.Canceled) {
		log.Error(err, "couldn't update zone insight", "zone", zone)
	}

	return &mesh_proto.ZoneHealthCheckResponse{
		Interval: durationpb.New(g.zoneHealthCheckInterval),
	}, nil
}

type ZoneWentOffline struct {
	Zone string
}

type ZoneOpenedStream struct {
	Zone string
}

type ZoneClientID struct {
	Zone string
}

func (id ZoneClientID) String() string {
	return fmt.Sprintf("%s", id.Zone)
}

func ZoneClientIDFromCtx(ctx context.Context, zone string) ZoneClientID {
	return ZoneClientID{Zone: zone}
}
