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

package bootstrap

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/navigator/pkg/features"
	"github.com/apache/dubbo-kubernetes/navigator/pkg/model"
	"github.com/apache/dubbo-kubernetes/navigator/pkg/server"
	"github.com/apache/dubbo-kubernetes/navigator/pkg/serviceregistry/providers"
	"github.com/apache/dubbo-kubernetes/navigator/pkg/xds"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/filewatcher"
	"github.com/apache/dubbo-kubernetes/pkg/h2c"
	dubbokeepalive "github.com/apache/dubbo-kubernetes/pkg/keepalive"
	"github.com/apache/dubbo-kubernetes/pkg/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/network"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	cluster2 "istio.io/istio/pkg/cluster"
	kubelib "istio.io/istio/pkg/kube"
	"k8s.io/client-go/rest"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type Server struct {
	XDSServer         *xds.DiscoveryServer
	clusterID         cluster.ID
	environment       *model.Environment
	server            server.Instance
	kubeClient        kubelib.Client
	grpcServer        *grpc.Server
	grpcAddress       string
	secureGrpcServer  *grpc.Server
	secureGrpcAddress string
	httpServer        *http.Server // debug, monitoring and readiness Server.
	httpAddr          string
	httpsServer       *http.Server // webhooks HTTPS Server.
	httpsAddr         string
	httpMux           *http.ServeMux
	httpsMux          *http.ServeMux // webhooks
	fileWatcher       filewatcher.FileWatcher
	internalStop      chan struct{}
}

func NewServer(args *NaviArgs, initFuncs ...func(*Server)) (*Server, error) {
	e := model.NewEnvironment()
	s := &Server{
		environment:  e,
		server:       server.New(),
		clusterID:    getClusterID(args),
		httpMux:      http.NewServeMux(),
		fileWatcher:  filewatcher.NewWatcher(),
		internalStop: make(chan struct{}),
	}
	for _, fn := range initFuncs {
		fn(s)
	}
	s.XDSServer = xds.NewDiscoveryServer(e, args.RegistryOptions.KubeOptions.ClusterAliases)
	// TODO configGen
	// TODO initReadinessProbes
	s.initServers(args)

	if err := s.serveHTTP(); err != nil {
		return nil, fmt.Errorf("error serving http: %v", err)
	}

	if err := s.initKubeClient(args); err != nil {
		return nil, fmt.Errorf("error initializing kube client: %v", err)
	}
	s.initMeshConfiguration(args, s.fileWatcher)

	return s, nil
}

func (s *Server) Start(stop <-chan struct{}) error {
	if err := s.server.Start(stop); err != nil {
		return err
	}
	// TODO waitForCacheSync
	// TODO XDSserver CacheSynced

	if s.secureGrpcAddress != "" {
		grpcListener, err := net.Listen("tcp", s.secureGrpcAddress)
		if err != nil {
			return err
		}
		go func() {
			fmt.Printf("starting secure gRPC discovery service at %s", grpcListener.Addr())
			if err := s.secureGrpcServer.Serve(grpcListener); err != nil {
				fmt.Errorf("error serving secure GRPC server: %v", err)
			}
		}()
	}

	if s.grpcAddress != "" {
		grpcListener, err := net.Listen("tcp", s.grpcAddress)
		if err != nil {
			return err
		}
		go func() {
			fmt.Printf("starting gRPC discovery service at %s", grpcListener.Addr())
			if err := s.grpcServer.Serve(grpcListener); err != nil {
				fmt.Errorf("error serving GRPC server: %v", err)
			}
		}()
	}

	if s.httpsServer != nil {
		httpsListener, err := net.Listen("tcp", s.httpsServer.Addr)
		if err != nil {
			return err
		}
		go func() {
			fmt.Printf("starting webhook service at %s", httpsListener.Addr())
			if err := s.httpsServer.ServeTLS(httpsListener, "", ""); network.IsUnexpectedListenerError(err) {
				fmt.Errorf("error serving https server: %v", err)
			}
		}()
		s.httpsAddr = httpsListener.Addr().String()
	}
	return nil
}

func (s *Server) initKubeClient(args *NaviArgs) error {
	if s.kubeClient != nil {
		return nil
	}
	hasK8SConfigStore := false
	if args.RegistryOptions.FileDir == "" {
		// If file dir is set - config controller will just use file.
		if _, err := os.Stat(args.MeshConfigFile); !os.IsNotExist(err) {
			meshConfig, err := mesh.ReadMeshConfig(args.MeshConfigFile)
			if err != nil {
				return fmt.Errorf("failed reading mesh config: %v", err)
			}
			if len(meshConfig.ConfigSources) == 0 && args.RegistryOptions.KubeConfig != "" {
				hasK8SConfigStore = true
			}
			for _, cs := range meshConfig.ConfigSources {
				if cs.Address == string(Kubernetes)+"://" {
					hasK8SConfigStore = true
					break
				}
			}
		} else if args.RegistryOptions.KubeConfig != "" {
			hasK8SConfigStore = true
		}
	}

	if hasK8SConfigStore || hasKubeRegistry(args.RegistryOptions.Registries) {
		kubeRestConfig, err := kubelib.DefaultRestConfig(args.RegistryOptions.KubeConfig, "", func(config *rest.Config) {
			config.QPS = args.RegistryOptions.KubeOptions.KubernetesAPIQPS
			config.Burst = args.RegistryOptions.KubeOptions.KubernetesAPIBurst
		})

		if err != nil {
			return fmt.Errorf("failed creating kube config: %v", err)
		}

		s.kubeClient, err = kubelib.NewClient(kubelib.NewClientConfigForRestConfig(kubeRestConfig), cluster2.ID(s.clusterID))
		if err != nil {
			return fmt.Errorf("failed creating kube client: %v", err)
		}
	}

	return nil
}

func (s *Server) initServers(args *NaviArgs) {
	s.initGrpcServer(args.KeepaliveOptions)
	multiplexGRPC := false
	if args.ServerOptions.GRPCAddr != "" {
		s.grpcAddress = args.ServerOptions.GRPCAddr
	} else {
		// This happens only if the GRPC port (15010) is disabled. We will multiplex
		// it on the HTTP port. Does not impact the HTTPS gRPC or HTTPS.
		fmt.Printf("multiplexing gRPC on http addr %v", args.ServerOptions.HTTPAddr)
		multiplexGRPC = true
	}
	h2s := &http2.Server{
		MaxConcurrentStreams: uint32(features.MaxConcurrentStreams),
	}
	multiplexHandler := h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("content-type"), "application/grpc") {
			s.grpcServer.ServeHTTP(w, r)
			return
		}
		s.httpMux.ServeHTTP(w, r)
	}), h2s)
	s.httpServer = &http.Server{
		Addr:        args.ServerOptions.HTTPAddr,
		Handler:     s.httpMux,
		IdleTimeout: 90 * time.Second, // matches http.DefaultTransport keep-alive timeout
		ReadTimeout: 30 * time.Second,
	}
	if multiplexGRPC {
		s.httpServer.ReadTimeout = 0
		s.httpServer.ReadHeaderTimeout = 30 * time.Second
		s.httpServer.Handler = multiplexHandler
	}
}

func (s *Server) initGrpcServer(options *dubbokeepalive.Options) {
}

func (s *Server) serveHTTP() error {
	// At this point we are ready - start Http Listener so that it can respond to readiness events.
	httpListener, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return err
	}
	go func() {
		fmt.Printf("starting HTTP service at %s", httpListener.Addr())
		if err := s.httpServer.Serve(httpListener); network.IsUnexpectedListenerError(err) {
			fmt.Errorf("error serving http server: %v", err)
		}
	}()
	s.httpAddr = httpListener.Addr().String()
	return nil
}

func getClusterID(args *NaviArgs) cluster.ID {
	clusterID := args.RegistryOptions.KubeOptions.ClusterID
	if clusterID == "" {
		if hasKubeRegistry(args.RegistryOptions.Registries) {
			clusterID = cluster.ID(providers.Kubernetes)
		}
	}
	return clusterID
}
