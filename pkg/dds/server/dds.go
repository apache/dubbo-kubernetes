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

package server

import (
	"context"
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	envoy_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/delta/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"

	envoy_server "github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

type Server interface {
	ZoneToGlobal(stream stream.DeltaStream) error
	mesh_proto.DDSSyncServiceServer
}

func NewServer(config envoy_cache.Cache, callbacks envoy_server.Callbacks) Server {
	deltaServer := delta.NewServer(context.Background(), config, callbacks)
	return &server{Server: deltaServer}
}

type server struct {
	delta.Server
	mesh_proto.UnimplementedDDSSyncServiceServer
}

func (s *server) GlobalToZoneSync(stream mesh_proto.DDSSyncService_GlobalToZoneSyncServer) error {
	errorStream := NewErrorRecorderStream(stream)
	err := s.Server.DeltaStreamHandler(errorStream, "")
	if err == nil {
		err = errorStream.Err()
	}
	return err
}

func (s *server) ZoneToGlobalSync(stream mesh_proto.DDSSyncService_ZoneToGlobalSyncServer) error {
	panic("not implemented")
}

func (s *server) ZoneToGlobal(stream stream.DeltaStream) error {
	errorStream := NewErrorRecorderStream(stream)
	err := s.Server.DeltaStreamHandler(errorStream, "")
	if err == nil {
		err = errorStream.Err()
	}
	return err
}
