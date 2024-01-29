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

package router

import (
	"fmt"
	"net"
)

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

import (
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/handlers/grpc_handlers"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/interceptors"
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
)

type GRPCRouter struct {
	PlainServer      *grpc.Server
	PlainServerPort  int
	SecureServer     *grpc.Server
	SecureServerPort int
}
type GrpcServer struct {
	PlainServer      *grpc.Server
	PlainServerPort  int
	SecureServer     *grpc.Server
	SecureServerPort int
}

func newGrpcServer(config dubbo_cp.Config) *GRPCRouter {
	router := &GRPCRouter{
		PlainServerPort:  config.Bufman.Server.GrpcPlainPort,
		SecureServerPort: config.Bufman.Server.GrpcSecurePort,
	}
	router.PlainServer = grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors.Auth()))
	reflection.Register(router.PlainServer)

	router.SecureServer = grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors.Auth()))
	reflection.Register(router.SecureServer)
	return router
}

func (r *GRPCRouter) NeedLeaderElection() bool {
	return false
}

func (r *GRPCRouter) Start(stop <-chan struct{}) error {
	plainLis, err := net.Listen("tcp", fmt.Sprintf(":%d", r.PlainServerPort))
	if err != nil {
		return err
	}
	secureLis, err := net.Listen("tcp", fmt.Sprintf(":%d", r.SecureServerPort))
	if err != nil {
		return err
	}
	plainErrChan := make(chan error)
	secureErrChan := make(chan error)
	go func() {
		defer close(plainErrChan)
		if err = r.PlainServer.Serve(plainLis); err != nil {
			logger.Sugar().Error(err, "bufman terminated with an error")
			plainErrChan <- err
		} else {
			logger.Sugar().Info("bufman terminated normally")
		}
	}()
	go func() {
		defer close(secureErrChan)
		if err = r.SecureServer.Serve(secureLis); err != nil {
			logger.Sugar().Error(err, "bufman terminated with an error")
			secureErrChan <- err
		} else {
			logger.Sugar().Info("terminated normally")
		}
	}()

	select {
	case <-stop:
		logger.Sugar().Info("bufman stopping gracefully")
		r.PlainServer.GracefulStop()
		r.SecureServer.GracefulStop()
		return nil
	case err := <-secureErrChan:
		return err
	case err := <-plainErrChan:
		return err
	}
}

func InitGRPCRouter(config dubbo_cp.Config) *GRPCRouter {
	r := newGrpcServer(config)

	register(r.PlainServer)
	register(r.SecureServer)

	return r
}

func register(server *grpc.Server) {
	// UserService
	registryv1alpha1.RegisterUserServiceServer(server, grpc_handlers.NewUserServiceHandler())

	// TokenService
	registryv1alpha1.RegisterTokenServiceServer(server, grpc_handlers.NewTokenServiceHandler())

	// AuthnService
	registryv1alpha1.RegisterAuthnServiceServer(server, grpc_handlers.NewAuthnServiceHandler())

	// RepositoryService
	registryv1alpha1.RegisterRepositoryServiceServer(server, grpc_handlers.NewRepositoryServiceHandler())

	// PushService
	registryv1alpha1.RegisterPushServiceServer(server, grpc_handlers.NewPushServiceHandler())

	// CommitService
	registryv1alpha1.RegisterRepositoryCommitServiceServer(server, grpc_handlers.NewCommitServiceHandler())

	// TagService
	registryv1alpha1.RegisterRepositoryTagServiceServer(server, grpc_handlers.NewTagServiceHandler())

	// ResolveService
	registryv1alpha1.RegisterResolveServiceServer(server, grpc_handlers.NewResolveServiceHandler())

	// DownloadService
	registryv1alpha1.RegisterDownloadServiceServer(server, grpc_handlers.NewDownloadServiceHandler())

	// DocService
	registryv1alpha1.RegisterDocServiceServer(server, grpc_handlers.NewDocServiceHandler())
}
