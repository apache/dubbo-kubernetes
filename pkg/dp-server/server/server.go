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
	"fmt"
	dp_server "github.com/apache/dubbo-kubernetes/pkg/config/dp-server"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"net/http"
)

var log = core.Log.WithName("dp-server")

type DpServer struct {
	config      dp_server.DpServerConfig
	PlainServer *grpc.Server
	httpMux     *http.ServeMux
}

var _ component.Component = &DpServer{}

func NewDpServer(config dp_server.DpServerConfig) *DpServer {
	srv := &DpServer{
		config:  config,
		httpMux: http.NewServeMux(),
	}
	srv.PlainServer = grpc.NewServer()
	reflection.Register(srv.PlainServer)

	return srv
}

func (d *DpServer) Start(stop <-chan struct{}) error {
	plainLis, err := net.Listen("tcp", fmt.Sprintf(":%d", d.config.Port))
	if err != nil {
		return err
	}
	plainErrChan := make(chan error)
	go func() {
		defer close(plainErrChan)
		if err = d.PlainServer.Serve(plainLis); err != nil {
			logger.Sugar().Error(err, "[cp-server] terminated with an error")
			plainErrChan <- err
		} else {
			logger.Sugar().Info("[cp-server] terminated normally")
		}
	}()

	select {
	case <-stop:
		logger.Sugar().Info("[cp-server] stopping gracefully")
		d.PlainServer.GracefulStop()
		return nil
	case err := <-plainErrChan:
		return err
	}
}

func (d *DpServer) NeedLeaderElection() bool {
	return false
}

func (d *DpServer) GrpcServer() *grpc.Server {
	return d.PlainServer
}

func (d *DpServer) HTTPMux() *http.ServeMux {
	return d.httpMux
}
