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
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"
)

import (
	"github.com/bakito/go-log-logr-adapter/adapter"

	http_prometheus "github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"
	"github.com/slok/go-http-metrics/middleware/std"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

import (
	dp_server "github.com/apache/dubbo-kubernetes/pkg/config/dp-server"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
)

var log = core.Log.WithName("dp-server")

const (
	grpcMaxConcurrentStreams = 1000000
	grpcKeepAliveTime        = 15 * time.Second
)

type Filter func(writer http.ResponseWriter, request *http.Request) bool

type DpServer struct {
	config         dp_server.DpServerConfig
	httpMux        *http.ServeMux
	grpcServer     *grpc.Server
	filter         Filter
	promMiddleware middleware.Middleware
}

var _ component.Component = &DpServer{}

func NewDpServer(config dp_server.DpServerConfig, filter Filter) *DpServer {
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
	}
	grpcServer := grpc.NewServer(grpcOptions...)

	promMiddleware := middleware.New(middleware.Config{
		Recorder: http_prometheus.NewRecorder(http_prometheus.Config{
			Prefix: "dp_server",
		}),
	})

	return &DpServer{
		config:         config,
		httpMux:        http.NewServeMux(),
		grpcServer:     grpcServer,
		filter:         filter,
		promMiddleware: promMiddleware,
	}
}

func (d *DpServer) Start(stop <-chan struct{}) error {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12} // To make gosec pass this is always set after
	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", d.config.Port),
		Handler:   http.HandlerFunc(d.handle),
		TLSConfig: tlsConfig,
		ErrorLog:  adapter.ToStd(log),
	}

	errChan := make(chan error)

	go func() {
		defer close(errChan)
		var err error
		if d.config.TlsCertFile == "" || d.config.TlsKeyFile == "" {
			// h2c is used for HTTP/2 cleartext upgrade
			// so that we can normally run xds server on gRPC
			server.Handler = h2c.NewHandler(server.Handler, &http2.Server{})
			err = server.ListenAndServe()
		} else {
			err = server.ListenAndServeTLS(d.config.TlsCertFile, d.config.TlsKeyFile)
		}
		if err != nil {
			if err != http.ErrServerClosed {
				log.Error(err, "terminated with an error")
				errChan <- err
				return
			}
		}
		log.Info("terminated normally")
	}()
	log.Info("starting", "interface", "0.0.0.0", "port", d.config.Port, "tls", true)

	select {
	case <-stop:
		log.Info("stopping")
		return server.Shutdown(context.Background())
	case err := <-errChan:
		return err
	}
}

func (d *DpServer) NeedLeaderElection() bool {
	return false
}

func (d *DpServer) handle(writer http.ResponseWriter, request *http.Request) {
	if !d.filter(writer, request) {
		return
	}
	// add filter function that will be in runtime, and we will implement it in kong-mesh
	if request.ProtoMajor == 2 && strings.Contains(request.Header.Get("Content-Type"), "application/grpc") {
		d.grpcServer.ServeHTTP(writer, request)
	} else {
		// we only want to measure HTTP not GRPC requests because they can mess up metrics
		// for example ADS bi-directional stream counts as one really long request
		std.Handler("", d.promMiddleware, d.httpMux).ServeHTTP(writer, request)
	}
}

func (d *DpServer) HTTPMux() *http.ServeMux {
	return d.httpMux
}

func (d *DpServer) GrpcServer() *grpc.Server {
	return d.grpcServer
}

func (d *DpServer) SetFilter(filter Filter) {
	d.filter = filter
}
