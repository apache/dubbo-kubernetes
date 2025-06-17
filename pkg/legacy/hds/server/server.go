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
	"sync/atomic"
)

import (
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_service_health "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"
	envoy_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	envoy_stream "github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"

	"github.com/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

import (
	hds_cache "github.com/apache/dubbo-kubernetes/pkg/hds/cache"
	hds_callbacks "github.com/apache/dubbo-kubernetes/pkg/hds/callbacks"
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
)

type Stream interface {
	grpc.ServerStream

	Send(specifier *envoy_service_health.HealthCheckSpecifier) error
	Recv() (*envoy_service_health.HealthCheckRequestOrEndpointHealthResponse, error)
}

type server struct {
	streamCount int64
	ctx         context.Context
	callbacks   hds_callbacks.Callbacks
	cache       envoy_cache.Cache
}

func New(ctx context.Context, config envoy_cache.Cache, callbacks hds_callbacks.Callbacks) envoy_service_health.HealthDiscoveryServiceServer {
	return &server{
		ctx:       ctx,
		callbacks: callbacks,
		cache:     config,
	}
}

func (s *server) StreamHealthCheck(stream envoy_service_health.HealthDiscoveryService_StreamHealthCheckServer) error {
	return s.StreamHandler(stream)
}

// StreamHandler converts a blocking read call to channels and initiates stream processing
func (s *server) StreamHandler(stream Stream) error {
	// a channel for receiving incoming requests
	reqOrRespCh := make(chan *envoy_service_health.HealthCheckRequestOrEndpointHealthResponse)
	go func() {
		defer close(reqOrRespCh)
		for {
			req, err := stream.Recv()
			if err != nil {
				return
			}
			select {
			case reqOrRespCh <- req:
			case <-stream.Context().Done():
				return
			case <-s.ctx.Done():
				return
			}
		}
	}()

	return s.process(stream, reqOrRespCh)
}

func (s *server) process(stream Stream, reqOrRespCh chan *envoy_service_health.HealthCheckRequestOrEndpointHealthResponse) error {
	streamID := atomic.AddInt64(&s.streamCount, 1)
	lastVersion := ""

	var watchCancellation func()
	defer func() {
		if watchCancellation != nil {
			watchCancellation()
		}
		if s.callbacks != nil {
			s.callbacks.OnStreamClosed(streamID)
		}
	}()

	send := func(resp envoy_cache.Response) error {
		if resp == nil {
			return errors.New("missing response")
		}

		out, err := resp.GetDiscoveryResponse()
		if err != nil {
			return err
		}
		if len(out.Resources) == 0 {
			return nil
		}

		hcs := &envoy_service_health.HealthCheckSpecifier{}
		if err := util_proto.UnmarshalAnyTo(out.Resources[0], hcs); err != nil {
			return err
		}
		lastVersion, err = resp.GetVersion()
		if err != nil {
			return err
		}
		return stream.Send(hcs)
	}

	if s.callbacks != nil {
		if err := s.callbacks.OnStreamOpen(stream.Context(), streamID); err != nil {
			return err
		}
	}

	responseChan := make(chan envoy_cache.Response, 1)
	node := &envoy_core.Node{}
	for {
		select {
		case <-s.ctx.Done():
			return nil
		case resp, more := <-responseChan:
			if !more {
				return status.Error(codes.Unavailable, "healthChecks watch failed")
			}
			if err := send(resp); err != nil {
				return err
			}
			if watchCancellation != nil {
				watchCancellation()
			}
			watchCancellation = s.cache.CreateWatch(&envoy_cache.Request{
				Node:          node,
				TypeUrl:       hds_cache.HealthCheckSpecifierType,
				ResourceNames: []string{"hcs"},
				VersionInfo:   lastVersion,
			}, envoy_stream.NewStreamState(false, nil), responseChan)
		case reqOrResp, more := <-reqOrRespCh:
			if !more {
				return nil
			}
			if reqOrResp == nil {
				return status.Errorf(codes.Unavailable, "empty request")
			}
			if req := reqOrResp.GetHealthCheckRequest(); req != nil {
				if req.Node != nil {
					node = req.Node
				} else {
					req.Node = node
				}
				if s.callbacks != nil {
					if err := s.callbacks.OnHealthCheckRequest(streamID, req); err != nil {
						return err
					}
				}
			}
			if resp := reqOrResp.GetEndpointHealthResponse(); resp != nil {
				if s.callbacks != nil {
					if err := s.callbacks.OnEndpointHealthResponse(streamID, resp); err != nil {
						return err
					}
				}
			}
			if watchCancellation != nil {
				watchCancellation()
			}
			watchCancellation = s.cache.CreateWatch(&envoy_cache.Request{
				Node:          node,
				TypeUrl:       hds_cache.HealthCheckSpecifierType,
				ResourceNames: []string{"hcs"},
				VersionInfo:   lastVersion,
			}, envoy_stream.NewStreamState(false, nil), responseChan)
		}
	}
}

func (s *server) FetchHealthCheck(ctx context.Context, response *envoy_service_health.HealthCheckRequestOrEndpointHealthResponse) (*envoy_service_health.HealthCheckSpecifier, error) {
	panic("not implemented")
}
