// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package activation

import (
	"fmt"
	"net"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/activation/demandpb"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/activation/externalscaler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// keepaliveOptions hold the StreamIsActive connections open through idle
// periods. Those streams carry nothing between activations, and a middlebox
// that drops idle connections would silently stop KEDA from ever hearing that
// a request arrived.
var keepaliveOptions = keepalive.ServerParameters{
	Time:    30 * time.Second,
	Timeout: 10 * time.Second,
}

// enforcementPolicy accepts the pings KEDA's client sends on an otherwise idle
// activation stream. Without permitting them the server would close the very
// connections it needs to keep.
var enforcementPolicy = keepalive.EnforcementPolicy{
	MinTime:             10 * time.Second,
	PermitWithoutStream: true,
}

// Server is the KEDA-facing endpoint: a gRPC service KEDA dials for every
// ScaledObject whose external trigger points at this control plane.
type Server struct {
	scaler   *Scaler
	registry *Registry
	grpc     *grpc.Server
}

// NewServer builds the activation endpoint and the demand registry behind it.
func NewServer() *Server {
	registry := NewRegistry()
	scaler := NewScaler(registry)

	server := grpc.NewServer(
		grpc.KeepaliveParams(keepaliveOptions),
		grpc.KeepaliveEnforcementPolicy(enforcementPolicy),
	)
	externalscaler.RegisterExternalScalerServer(server, scaler)
	// Gateways report into the same registry the scaler reads, on the same
	// listener: a gateway that can reach this replica can always be heard by
	// the KEDA stream this replica is serving.
	demandpb.RegisterActivationDemandServer(server, NewDemandService(registry))

	return &Server{scaler: scaler, registry: registry, grpc: server}
}

// Scaler exposes the KEDA subscription state the policy controller reports.
func (s *Server) Scaler() *Scaler { return s.scaler }

// Registry exposes the demand store gateways report into.
func (s *Server) Registry() *Registry { return s.registry }

// Serve listens on addr until stop is closed. An empty address disables the
// endpoint, which is how a cluster without KEDA installed runs unchanged.
func (s *Server) Serve(addr string, stop <-chan struct{}) error {
	if addr == "" {
		logger.Info("activation scaler disabled; no listen address configured")
		return nil
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("unable to listen on activation scaler socket: %v", err)
	}

	go func() {
		logger.Infof("starting activation scaler at %s", listener.Addr())
		if err := s.grpc.Serve(listener); err != nil {
			logger.Errorf("error serving activation scaler: %v", err)
		}
	}()

	go func() {
		<-stop
		// Graceful: an activation stream that is mid-send is the only way KEDA
		// learns about a waiting request, so it is worth letting it finish.
		s.grpc.GracefulStop()
	}()

	return nil
}
