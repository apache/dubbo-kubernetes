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
)

import (
	envoy_sd "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"

	"google.golang.org/grpc/metadata"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
)

type ServerStream interface {
	stream.DeltaStream
}

type serverStream struct {
	stream mesh_proto.DDSSyncService_ZoneToGlobalSyncClient
}

// NewServerStream converts client stream to a server's DeltaStream, so it can be used in DeltaStreamHandler
func NewServerStream(stream mesh_proto.DDSSyncService_ZoneToGlobalSyncClient) ServerStream {
	s := &serverStream{
		stream: stream,
	}
	return s
}

func (k *serverStream) Send(response *envoy_sd.DeltaDiscoveryResponse) error {
	err := k.stream.Send(response)
	return err
}

func (k *serverStream) Recv() (*envoy_sd.DeltaDiscoveryRequest, error) {
	res, err := k.stream.Recv()
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (k *serverStream) SetHeader(metadata.MD) error {
	panic("not implemented")
}

func (k *serverStream) SendHeader(metadata.MD) error {
	panic("not implemented")
}

func (k *serverStream) SetTrailer(metadata.MD) {
	panic("not implemented")
}

func (k *serverStream) Context() context.Context {
	return k.stream.Context()
}

func (k *serverStream) SendMsg(m interface{}) error {
	panic("not implemented")
}

func (k *serverStream) RecvMsg(m interface{}) error {
	panic("not implemented")
}
