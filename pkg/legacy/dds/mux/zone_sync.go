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

package mux

import (
	"context"
)

import (
	"github.com/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/dds"
	"github.com/apache/dubbo-kubernetes/pkg/dds/service"
	"github.com/apache/dubbo-kubernetes/pkg/dds/util"
	"github.com/apache/dubbo-kubernetes/pkg/events"
)

type Filter interface {
	InterceptServerStream(stream grpc.ServerStream) error
	InterceptClientStream(stream grpc.ClientStream) error
}

type OnGlobalToZoneSyncConnectFunc func(stream mesh_proto.DDSSyncService_GlobalToZoneSyncServer, errorCh chan error)

func (f OnGlobalToZoneSyncConnectFunc) OnGlobalToZoneSyncConnect(stream mesh_proto.DDSSyncService_GlobalToZoneSyncServer, errorChan chan error) {
	f(stream, errorChan)
}

type OnZoneToGlobalSyncConnectFunc func(stream mesh_proto.DDSSyncService_ZoneToGlobalSyncServer, errorCh chan error)

func (f OnZoneToGlobalSyncConnectFunc) OnZoneToGlobalSyncConnect(stream mesh_proto.DDSSyncService_ZoneToGlobalSyncServer, errorCh chan error) {
	f(stream, errorCh)
}

var clientLog = core.Log.WithName("dds-delta-client")

type DDSSyncServiceServer struct {
	globalToZoneCb OnGlobalToZoneSyncConnectFunc
	zoneToGlobalCb OnZoneToGlobalSyncConnectFunc
	filters        []Filter
	extensions     context.Context
	eventBus       events.EventBus
	mesh_proto.UnimplementedDDSSyncServiceServer
	context context.Context
}

func (g *DDSSyncServiceServer) mustEmbedUnimplementedDDSSyncServiceServer() {
	panic("implement me")
}

func NewDDSSyncServiceServer(ctx context.Context, globalToZoneCb OnGlobalToZoneSyncConnectFunc, zoneToGlobalCb OnZoneToGlobalSyncConnectFunc, filters []Filter, extensions context.Context, eventBus events.EventBus) *DDSSyncServiceServer {
	return &DDSSyncServiceServer{
		context:        ctx,
		globalToZoneCb: globalToZoneCb,
		zoneToGlobalCb: zoneToGlobalCb,
		filters:        filters,
		extensions:     extensions,
		eventBus:       eventBus,
	}
}

func (g *DDSSyncServiceServer) GlobalToZoneSync(stream mesh_proto.DDSSyncService_GlobalToZoneSyncServer) error {
	zone, err := util.ClientIDFromIncomingCtx(stream.Context())
	if err != nil {
		return err
	}
	for _, filter := range g.filters {
		if err := filter.InterceptServerStream(stream); err != nil {
			return errors.Wrap(err, "closing DDS stream following a callback error")
		}
	}

	shouldDisconnectStream := g.watchZoneHealthCheck(stream.Context(), zone)
	defer shouldDisconnectStream.Close()

	processingErrorsCh := make(chan error)
	go g.globalToZoneCb.OnGlobalToZoneSyncConnect(stream, processingErrorsCh)
	select {
	case <-shouldDisconnectStream.Recv():
		clientLog.Info("ending stream, zone health check failed")
		return status.Error(codes.Canceled, "stream canceled - zone hc failed")
	case <-stream.Context().Done():
		clientLog.Info("GlobalToZoneSync rpc stream stopped")
		return status.Error(codes.Canceled, "stream canceled - stream stopped")
	case <-g.context.Done():
		clientLog.Info("app context done")
		return status.Error(codes.Unavailable, "stream unavailable")
	case err := <-processingErrorsCh:
		if status.Code(err) == codes.Unimplemented {
			return errors.Wrap(err, "GlobalToZoneSync rpc stream failed, because Global CP does not implement this rpc. Upgrade Global CP.")
		}
		clientLog.Error(err, "GlobalToZoneSync rpc stream failed prematurely, will restart in background")
		return status.Error(codes.Internal, "stream failed")
	}
}

func (g *DDSSyncServiceServer) ZoneToGlobalSync(stream mesh_proto.DDSSyncService_ZoneToGlobalSyncServer) error {
	zone, err := util.ClientIDFromIncomingCtx(stream.Context())
	if err != nil {
		return err
	}
	logger := clientLog.WithValues("clientID", zone)
	for _, filter := range g.filters {
		if err := filter.InterceptServerStream(stream); err != nil {
			return errors.Wrap(err, "closing DDS stream following a callback error")
		}
	}

	shouldDisconnectStream := g.watchZoneHealthCheck(stream.Context(), zone)
	defer shouldDisconnectStream.Close()

	processingErrorsCh := make(chan error)
	go g.zoneToGlobalCb.OnZoneToGlobalSyncConnect(stream, processingErrorsCh)
	select {
	case <-shouldDisconnectStream.Recv():
		logger.Info("ending stream, zone health check failed")
		return nil
	case <-stream.Context().Done():
		logger.Info("ZoneToGlobalSync rpc stream stopped")
		return nil
	case err := <-processingErrorsCh:
		if status.Code(err) == codes.Unimplemented {
			return errors.Wrap(err, "ZoneToGlobalSync rpc stream failed, because Global CP does not implement this rpc. Upgrade Global CP.")
		}
		logger.Error(err, "ZoneToGlobalSync rpc stream failed prematurely, will restart in background")
		return status.Error(codes.Internal, "stream failed")
	}
}

func (g *DDSSyncServiceServer) watchZoneHealthCheck(streamContext context.Context, zone string) events.Listener {
	shouldDisconnectStream := events.NewNeverListener()

	if dds.ContextHasFeature(streamContext, dds.FeatureZonePingHealth) {
		shouldDisconnectStream = g.eventBus.Subscribe(func(e events.Event) bool {
			disconnectEvent, ok := e.(service.ZoneWentOffline)
			return ok && disconnectEvent.Zone == zone
		})
		g.eventBus.Send(service.ZoneOpenedStream{Zone: zone})
	}

	return shouldDisconnectStream
}
