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
	"io"
	"math/rand"
	"time"
)

import (
	"github.com/pkg/errors"

	"github.com/sethvargo/go-retry"

	"golang.org/x/exp/slices"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/dds"
	"github.com/apache/dubbo-kubernetes/pkg/dds/util"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	util_grpc "github.com/apache/dubbo-kubernetes/pkg/util/grpc"
)

var log = core.Log.WithName("dds-service")

type StreamInterceptor interface {
	InterceptServerStream(stream grpc.ServerStream) error
}

type GlobalDDSServiceServer struct {
	envoyAdminRPCs          EnvoyAdminRPCs
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
	envoyAdminRPCs EnvoyAdminRPCs,
	resManager manager.ResourceManager,
	instanceID string,
	filters []StreamInterceptor,
	extensions context.Context,
	upsertCfg config_store.UpsertConfig,
	eventBus events.EventBus,
	zoneHealthCheckInterval time.Duration,
) *GlobalDDSServiceServer {
	return &GlobalDDSServiceServer{
		context:                 ctx,
		envoyAdminRPCs:          envoyAdminRPCs,
		resManager:              resManager,
		instanceID:              instanceID,
		filters:                 filters,
		extensions:              extensions,
		upsertCfg:               upsertCfg,
		eventBus:                eventBus,
		zoneHealthCheckInterval: zoneHealthCheckInterval,
	}
}

func (g *GlobalDDSServiceServer) StreamXDSConfigs(stream mesh_proto.GlobalDDSService_StreamXDSConfigsServer) error {
	return g.streamEnvoyAdminRPC(ConfigDumpRPC, g.envoyAdminRPCs.XDSConfigDump, stream, func() (util_grpc.ReverseUnaryMessage, error) {
		return stream.Recv()
	})
}

func (g *GlobalDDSServiceServer) StreamStats(stream mesh_proto.GlobalDDSService_StreamStatsServer) error {
	return g.streamEnvoyAdminRPC(StatsRPC, g.envoyAdminRPCs.Stats, stream, func() (util_grpc.ReverseUnaryMessage, error) {
		return stream.Recv()
	})
}

func (g *GlobalDDSServiceServer) StreamClusters(stream mesh_proto.GlobalDDSService_StreamClustersServer) error {
	return g.streamEnvoyAdminRPC(ClustersRPC, g.envoyAdminRPCs.Clusters, stream, func() (util_grpc.ReverseUnaryMessage, error) {
		return stream.Recv()
	})
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

func (g *GlobalDDSServiceServer) streamEnvoyAdminRPC(
	rpcName string,
	rpc util_grpc.ReverseUnaryRPCs,
	stream grpc.ServerStream,
	recv func() (util_grpc.ReverseUnaryMessage, error),
) error {
	zone, err := util.ClientIDFromIncomingCtx(stream.Context())
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	zoneID := ZoneClientIDFromCtx(stream.Context(), zone)

	shouldDisconnectStream := events.NewNeverListener()

	md, _ := metadata.FromIncomingContext(stream.Context())
	features := md.Get(dds.FeaturesMetadataKey)

	if slices.Contains(features, dds.FeatureZonePingHealth) {
		shouldDisconnectStream = g.eventBus.Subscribe(func(e events.Event) bool {
			disconnectEvent, ok := e.(ZoneWentOffline)
			return ok && disconnectEvent.Zone == zone
		})
		g.eventBus.Send(ZoneOpenedStream{Zone: zone})
	}

	defer shouldDisconnectStream.Close()

	for _, filter := range g.filters {
		if err := filter.InterceptServerStream(stream); err != nil {
			switch status.Code(err) {
			case codes.InvalidArgument, codes.Unauthenticated, codes.PermissionDenied:
				log.Info("stream interceptor terminating the stream", "cause", err)
			default:
				log.Error(err, "stream interceptor terminating the stream")
			}
			return err
		}
	}
	log.Info("Envoy Admin RPC stream started")
	rpc.ClientConnected(zoneID.String(), stream)
	if err := g.storeStreamConnection(stream.Context(), zone, rpcName, g.instanceID); err != nil {
		if errors.Is(err, context.Canceled) {
			return status.Error(codes.Canceled, "stream was cancelled")
		}
		log.Error(err, "could not store stream connection")
		return status.Error(codes.Internal, "could not store stream connection")
	}
	log.Info("stored stream connection")
	streamResult := make(chan error, 1)
	go func() {
		for {
			resp, err := recv()
			if err == io.EOF {
				log.Info("stream stopped")
				streamResult <- nil
				return
			}
			if status.Code(err) == codes.Canceled {
				log.Info("stream cancelled")
				streamResult <- nil
				return
			}
			if err != nil {
				log.Error(err, "could not receive a message")
				streamResult <- status.Error(codes.Internal, "could not receive a message")
				return
			}
			log.V(1).Info("Envoy Admin RPC response received", "requestId", resp.GetRequestId())
			if err := rpc.ResponseReceived(zoneID.String(), resp); err != nil {
				log.Error(err, "could not mark the response as received")
				streamResult <- status.Error(codes.InvalidArgument, "could not mark the response as received")
				return
			}
		}
	}()
	select {
	case <-g.context.Done():
		log.Info("app context done")
		return status.Error(codes.Unavailable, "stream unavailable")
	case <-shouldDisconnectStream.Recv():
		log.Info("ending stream, zone health check failed")
		return status.Error(codes.Canceled, "stream canceled")
	case res := <-streamResult:
		return res
	}
}

type ZoneWentOffline struct {
	Zone string
}

type ZoneOpenedStream struct {
	Zone string
}

func (g *GlobalDDSServiceServer) storeStreamConnection(ctx context.Context, zone string, rpcName string, instance string) error {
	key := model.ResourceKey{Name: zone}

	// wait for Zone to be created, only then we can create Zone Insight
	err := retry.Do(
		ctx,
		retry.WithMaxRetries(30, retry.NewConstant(1*time.Second)),
		func(ctx context.Context) error {
			return retry.RetryableError(g.resManager.Get(ctx, system.NewZoneResource(), core_store.GetBy(key)))
		},
	)
	if err != nil {
		return err
	}

	// Add delay for Upsert. If Global CP is behind an HTTP load balancer,
	// it might be the case that each Envoy Admin stream will land on separate instance.
	// In this case, all instances will try to update Zone Insight which will result in conflicts.
	// Since it's unusual to immediately execute envoy admin rpcs after zone is connected, 0-10s delay should be fine.
	// #nosec G404 - math rand is enough
	time.Sleep(time.Duration(rand.Int31n(10000)) * time.Millisecond)

	zoneInsight := system.NewZoneInsightResource()
	return manager.Upsert(ctx, g.resManager, key, zoneInsight, func(resource model.Resource) error {
		if zoneInsight.Spec.EnvoyAdminStreams == nil {
			zoneInsight.Spec.EnvoyAdminStreams = &system_proto.EnvoyAdminStreams{}
		}
		switch rpcName {
		case ConfigDumpRPC:
			zoneInsight.Spec.EnvoyAdminStreams.ConfigDumpGlobalInstanceId = instance
		case StatsRPC:
			zoneInsight.Spec.EnvoyAdminStreams.StatsGlobalInstanceId = instance
		case ClustersRPC:
			zoneInsight.Spec.EnvoyAdminStreams.ClustersGlobalInstanceId = instance
		}
		return nil
	}, manager.WithConflictRetry(g.upsertCfg.ConflictRetryBaseBackoff.Duration, g.upsertCfg.ConflictRetryMaxTimes, g.upsertCfg.ConflictRetryJitterPercent)) // we need retry because zone sink or other RPC may also update the insight.
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
