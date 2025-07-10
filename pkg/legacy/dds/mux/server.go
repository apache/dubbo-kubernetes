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
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

import (
	"github.com/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/config/multizone"
	config_types "github.com/apache/dubbo-kubernetes/pkg/config/types"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	"github.com/apache/dubbo-kubernetes/pkg/dds/service"
)

const (
	grpcMaxConcurrentStreams = 1000000
	grpcKeepAliveTime        = 15 * time.Second
)

var muxServerLog = core.Log.WithName("dds-mux-server")

type OnGlobalToZoneSyncStartedFunc func(session mesh_proto.DDSSyncService_GlobalToZoneSyncClient, errorCh chan error)

func (f OnGlobalToZoneSyncStartedFunc) OnGlobalToZoneSyncStarted(session mesh_proto.DDSSyncService_GlobalToZoneSyncClient, errorCh chan error) {
	f(session, errorCh)
}

type OnZoneToGlobalSyncStartedFunc func(session mesh_proto.DDSSyncService_ZoneToGlobalSyncClient, errorCh chan error)

func (f OnZoneToGlobalSyncStartedFunc) OnZoneToGlobalSyncStarted(session mesh_proto.DDSSyncService_ZoneToGlobalSyncClient, errorCh chan error) {
	f(session, errorCh)
}

type server struct {
	config               multizone.DdsServerConfig
	CallbacksGlobal      OnGlobalToZoneSyncConnectFunc
	CallbacksZone        OnZoneToGlobalSyncConnectFunc
	filters              []Filter
	serviceServer        *service.GlobalDDSServiceServer
	ddsSyncServiceServer *DDSSyncServiceServer
	streamInterceptors   []grpc.StreamServerInterceptor
	unaryInterceptors    []grpc.UnaryServerInterceptor
}

func NewServer(
	filters []Filter,
	streamInterceptors []grpc.StreamServerInterceptor,
	unaryInterceptors []grpc.UnaryServerInterceptor,
	config multizone.DdsServerConfig,
	serviceServer *service.GlobalDDSServiceServer,
	ddsSyncServiceServer *DDSSyncServiceServer,
) component.Component {
	return &server{
		filters:              filters,
		config:               config,
		serviceServer:        serviceServer,
		ddsSyncServiceServer: ddsSyncServiceServer,
		streamInterceptors:   streamInterceptors,
		unaryInterceptors:    unaryInterceptors,
	}
}

func (s *server) Start(stop <-chan struct{}) error {
	grpcOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    grpcKeepAliveTime,
			Timeout: grpcKeepAliveTime,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcKeepAliveTime,
			PermitWithoutStream: true,
		}),
		grpc.MaxRecvMsgSize(int(s.config.MaxMsgSize)),
		grpc.MaxSendMsgSize(int(s.config.MaxMsgSize)),
	}
	if s.config.TlsCertFile != "" && s.config.TlsEnabled {
		cert, err := tls.LoadX509KeyPair(s.config.TlsCertFile, s.config.TlsKeyFile)
		if err != nil {
			return errors.Wrap(err, "failed to load TLS certificate")
		}
		tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
		if tlsCfg.MinVersion, err = config_types.TLSVersion(s.config.TlsMinVersion); err != nil {
			return err
		}
		if tlsCfg.MaxVersion, err = config_types.TLSVersion(s.config.TlsMaxVersion); err != nil {
			return err
		}
		if tlsCfg.CipherSuites, err = config_types.TLSCiphers(s.config.TlsCipherSuites); err != nil {
			return err
		}
		grpcOptions = append(grpcOptions, grpc.Creds(credentials.NewTLS(tlsCfg)))
	}
	for _, interceptor := range s.streamInterceptors {
		grpcOptions = append(grpcOptions, grpc.ChainStreamInterceptor(interceptor))
	}
	grpcOptions = append(
		grpcOptions,
		grpc.ChainUnaryInterceptor(s.unaryInterceptors...),
	)
	grpcServer := grpc.NewServer(grpcOptions...)

	mesh_proto.RegisterGlobalDDSServiceServer(grpcServer, s.serviceServer)
	mesh_proto.RegisterDDSSyncServiceServer(grpcServer, s.ddsSyncServiceServer)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.GrpcPort))
	if err != nil {
		return err
	}

	errChan := make(chan error)
	go func() {
		defer close(errChan)
		if err = grpcServer.Serve(lis); err != nil {
			muxServerLog.Error(err, "terminated with an error")
			errChan <- err
		} else {
			muxServerLog.Info("terminated normally")
		}
	}()
	muxServerLog.Info("starting", "interface", "0.0.0.0", "port", s.config.GrpcPort)

	select {
	case <-stop:
		muxServerLog.Info("stopping gracefully")
		grpcServer.GracefulStop()
		muxServerLog.Info("stopped")
		return nil
	case err := <-errChan:
		return err
	}
}

func (s *server) NeedLeaderElection() bool {
	return false
}
