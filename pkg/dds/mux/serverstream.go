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
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	envoy_sd "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc/metadata"
)

type ddsServerStream struct {
	ctx          context.Context
	bufferStream *bufferStream
}

func (k *ddsServerStream) Send(response *envoy_sd.DiscoveryResponse) error {
	err := k.bufferStream.Send(&mesh_proto.Message{Value: &mesh_proto.Message_Response{Response: response}})
	return err
}

func (k *ddsServerStream) Recv() (*envoy_sd.DiscoveryRequest, error) {
	res, err := k.bufferStream.Recv()
	if err != nil {
		return nil, err
	}
	return res.GetRequest(), nil
}

func (k *ddsServerStream) SetHeader(metadata.MD) error {
	panic("not implemented")
}

func (k *ddsServerStream) SendHeader(metadata.MD) error {
	panic("not implemented")
}

func (k *ddsServerStream) SetTrailer(metadata.MD) {
	panic("not implemented")
}

func (k *ddsServerStream) Context() context.Context {
	return k.ctx
}

func (k *ddsServerStream) SendMsg(m interface{}) error {
	panic("not implemented")
}

func (k *ddsServerStream) RecvMsg(m interface{}) error {
	panic("not implemented")
}
