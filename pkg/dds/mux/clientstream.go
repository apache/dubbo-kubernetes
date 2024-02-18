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
	envoy_sd "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"google.golang.org/grpc/metadata"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
)

type ddsClientStream struct {
	ctx          context.Context
	bufferStream *bufferStream
}

func (k *ddsClientStream) Send(request *envoy_sd.DiscoveryRequest) error {
	err := k.bufferStream.Send(&mesh_proto.Message{Value: &mesh_proto.Message_Request{Request: request}})
	return err
}

func (k *ddsClientStream) Recv() (*envoy_sd.DiscoveryResponse, error) {
	res, err := k.bufferStream.Recv()
	if err != nil {
		return nil, err
	}
	return res.GetResponse(), nil
}

func (k *ddsClientStream) Header() (metadata.MD, error) {
	panic("not implemented")
}

func (k *ddsClientStream) Trailer() metadata.MD {
	panic("not implemented")
}

func (k *ddsClientStream) CloseSend() error {
	panic("not implemented")
}

func (k *ddsClientStream) Context() context.Context {
	return k.ctx
}

func (k *ddsClientStream) SendMsg(m interface{}) error {
	panic("not implemented")
}

func (k *ddsClientStream) RecvMsg(m interface{}) error {
	panic("not implemented")
}
