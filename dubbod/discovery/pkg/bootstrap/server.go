//
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

package bootstrap

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/status"

	meshv1alpha1 "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/features"
	dubbogrpc "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/grpc"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/keycertbundle"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/networking/core"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/server"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/serviceregistry/aggregate"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/serviceregistry/provider"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/ca"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/ra"
	caserver "github.com/apache/dubbo-kubernetes/dubbod/security/pkg/server/ca"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/ctrlz"
	"github.com/apache/dubbo-kubernetes/pkg/filewatcher"
	"github.com/apache/dubbo-kubernetes/pkg/h2c"
	dubbokeepalive "github.com/apache/dubbo-kubernetes/pkg/keepalive"
	kubelib "github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/kube/multicluster"
	"github.com/apache/dubbo-kubernetes/pkg/kube/namespace"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	sec_model "github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/network"
	"github.com/apache/dubbo-kubernetes/pkg/spiffe"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/fsnotify/fsnotify"
	grpcprom "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.uber.org/atomic"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

const (
	// debounce file watcher events to minimize noise in logs
	watchDebounceDelay = 100 * time.Millisecond
)

type Server struct {
	XDSServer   *xds.DiscoveryServer
	clusterID   cluster.ID
	environment *model.Environment
	server      server.Instance
	kubeClient  kubelib.Client

	grpcServer        *grpc.Server
	grpcAddress       string
	secureGrpcServer  *grpc.Server
	secureGrpcAddress string

	httpServer  *http.Server // debug, monitoring and readiness Server.
	httpsServer *http.Server // webhooks HTTPS Server.
	httpAddr    string
	httpsAddr   string
	httpMux     *http.ServeMux
	httpsMux    *http.ServeMux // webhooks

	ConfigStores     []model.ConfigStoreController
	configController model.ConfigStoreController

	multiclusterController *multicluster.Controller

	fileWatcher filewatcher.FileWatcher

	internalStop     chan struct{}
	shutdownDuration time.Duration

	caServer                *caserver.Server
	cacertsWatcher          *fsnotify.Watcher
	dubbodCertBundleWatcher *keycertbundle.Watcher
	dubbodCert              *tls.Certificate
	RA                      ra.RegistrationAuthority
	CA                      *ca.DubboCA

	dnsNames []string

	certMu           sync.RWMutex
	internalDebugMux *http.ServeMux

	readinessProbes map[string]readinessProbe
	readinessFlags  *readinessFlags

	webhookInfo *webhookInfo

	krtDebugger *krt.DebugHandler

	RWConfigStore model.ConfigStoreController
	statusManager *status.Manager
}

type readinessFlags struct {
	InjectorReady         atomic.Bool
	configValidationReady atomic.Bool
}

type webhookInfo struct {
	mu sync.RWMutex
	wh *inject.Webhook
}

func (w *webhookInfo) getWebhookConfig() inject.Config {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.wh != nil && w.wh.Config != nil {
		return *w.wh.Config
	}
	return inject.Config{}
}

func (w *webhookInfo) addHandler(fn func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// Note: Dubbo's Webhook doesn't have RegisterInjectionHandler method
	// This is a placeholder for future implementation
	// For now, we'll call the handler directly if webhook is available
	if w.wh != nil {
		// Handler will be called when webhook config changes
		// This is a simplified implementation compared to Dubbo
		fn()
	}
}

type readinessProbe func() bool

func NewServer(args *DubboArgs, initFuncs ...func(*Server)) (*Server, error) {
	e := model.NewEnvironment()
	e.DomainSuffix = args.RegistryOptions.KubeOptions.DomainSuffix

	ac := aggregate.NewController(aggregate.Options{
		MeshHolder:      e,
		ConfigClusterID: getClusterID(args),
	})
	e.ServiceDiscovery = ac

	s := &Server{
		environment:             e,
		server:                  server.New(),
		clusterID:               getClusterID(args),
		httpMux:                 http.NewServeMux(),
		dubbodCertBundleWatcher: keycertbundle.NewWatcher(),
		fileWatcher:             filewatcher.NewWatcher(),
		internalStop:            make(chan struct{}),
		readinessProbes:         make(map[string]readinessProbe),
		readinessFlags:          &readinessFlags{},
		webhookInfo:             &webhookInfo{},
	}
	for _, fn := range initFuncs {
		fn(s)
	}

	s.XDSServer = xds.NewDiscoveryServer(e, args.RegistryOptions.KubeOptions.ClusterAliases, args.KrtDebugger)
	configGen := core.NewConfigGenerator(s.XDSServer.Cache)

	s.initReadinessProbes()

	s.initServers(args)

	if err := s.serveHTTP(); err != nil {
		return nil, fmt.Errorf("error serving http: %v", err)
	}

	if err := s.initKubeClient(args); err != nil {
		return nil, fmt.Errorf("error initializing kube client: %v", err)
	}

	s.initMeshGlobalConfiguration(args, s.fileWatcher)

	if s.kubeClient != nil {
		// Build a namespace watcher. This must have no filter, since this is our input to the filter itself.
		namespaces := kclient.New[*corev1.Namespace](s.kubeClient)
		filter := namespace.NewDiscoveryNamespacesFilter(namespaces, s.environment.Watcher, s.internalStop)
		s.kubeClient = kubelib.SetObjectFilter(s.kubeClient, filter)
	}

	s.initMeshHandlers(nil)

	s.environment.Init()

	// Options based on the current 'defaults' in dubbo.
	caOpts := &caOptions{
		TrustDomain:      s.environment.Mesh().TrustDomain,
		Namespace:        args.Namespace,
		ExternalCAType:   ra.CaExternalType(externalCaType),
		CertSignerDomain: features.CertSignerDomain,
	}

	if caOpts.ExternalCAType == ra.ExtCAK8s {
		caOpts.ExternalCASigner = k8sSigner
	}
	// CA signing certificate must be created first if needed.
	if err := s.maybeCreateCA(caOpts); err != nil {
		return nil, err
	}

	if err := s.initControllers(args); err != nil {
		return nil, err
	}

	InitGenerators(s.XDSServer, configGen)

	dubbodHost, _, err := e.GetDiscoveryAddress()
	if err != nil {
		return nil, err
	}

	if err := s.initDubbodCerts(args, string(dubbodHost)); err != nil {
		return nil, err
	}

	// Secure gRPC Server must be initialized after CA is created as may use a Aegis generated cert.
	if err := s.initSecureDiscoveryService(args, s.environment.Mesh().GetTrustDomain()); err != nil {
		return nil, fmt.Errorf("error initializing secure gRPC Listener: %v", err)
	}

	if s.kubeClient != nil {
		s.initSecureWebhookServer(args)
		wh, err := s.initInjector(args)
		if err != nil {
			return nil, fmt.Errorf("error initializing grpcxds injector: %v", err)
		}
		s.readinessFlags.InjectorReady.Store(true)
		s.webhookInfo.mu.Lock()
		s.webhookInfo.wh = wh
		s.webhookInfo.mu.Unlock()

		if err := s.initConfigValidation(args); err != nil {
			return nil, fmt.Errorf("error initializing config validator: %v", err)
		}
	}

	// Note: initRegistryEventHandlers is called in Start() after config controller starts
	s.initDiscoveryService()

	s.startCA(caOpts)

	if args.CtrlZOptions != nil {
		_, _ = ctrlz.Run(args.CtrlZOptions)
	}

	if s.kubeClient != nil {
		s.addStartFunc("kube client", func(stop <-chan struct{}) error {
			s.kubeClient.RunAndWait(stop)
			return nil
		})
	}

	return s, nil
}

func (s *Server) Start(stop <-chan struct{}) error {
	log.Infof("Starting Dubbod Server with primary cluster %s", s.clusterID)
	if err := s.server.Start(stop); err != nil {
		return err
	}

	if !s.waitForCacheSync(stop) {
		return fmt.Errorf("failed to sync cache")
	}

	s.XDSServer.CachesSynced()

	// Register event handlers after config controller has started and synced
	// This ensures that config changes are properly detected and handled
	s.initRegistryEventHandlers()

	if s.secureGrpcAddress != "" {
		grpcListener, err := net.Listen("tcp", s.secureGrpcAddress)
		if err != nil {
			return err
		}
		go func() {
			log.Infof("starting secure gRPC discovery service at %s", grpcListener.Addr())
			if err := s.secureGrpcServer.Serve(grpcListener); err != nil {
				log.Errorf("error serving secure GRPC server: %v", err)
			}
		}()
	}

	if s.grpcAddress != "" {
		grpcListener, err := net.Listen("tcp", s.grpcAddress)
		if err != nil {
			return err
		}
		go func() {
			log.Infof("starting gRPC discovery service at %s", grpcListener.Addr())
			if err := s.grpcServer.Serve(grpcListener); err != nil {
				log.Errorf("error serving GRPC server: %v", err)
			}
		}()
	}

	if s.httpsServer != nil {
		httpsListener, err := net.Listen("tcp", s.httpsServer.Addr)
		if err != nil {
			return err
		}
		go func() {
			log.Infof("starting webhook service at %s", httpsListener.Addr())
			if err := s.httpsServer.ServeTLS(httpsListener, "", ""); network.IsUnexpectedListenerError(err) {
				log.Errorf("error serving https server: %v", err)
			}
		}()
		s.httpsAddr = httpsListener.Addr().String()
	}

	s.waitForShutdown(stop)

	return nil
}

func (s *Server) startCA(caOpts *caOptions) {
	if s.CA == nil && s.RA == nil {
		return
	}
	// init the RA server if configured, else start init CA server
	if s.RA != nil {
		log.Infof("initializing CA server with RA")
		s.initCAServer(s.RA, caOpts)
	} else if s.CA != nil {
		log.Infof("initializing CA server with Dubbod CA")
		s.initCAServer(s.CA, caOpts)
	}
	s.addStartFunc("ca", func(stop <-chan struct{}) error {
		grpcServer := s.secureGrpcServer
		if s.secureGrpcServer == nil {
			grpcServer = s.grpcServer
		}
		log.Infof("Starting CA server")
		s.RunCA(grpcServer)
		return nil
	})
}

func (s *Server) initDiscoveryService() {
	log.Infof("starting discovery service")
	s.addStartFunc("xds server", func(stop <-chan struct{}) error {
		log.Infof("Starting ADS server")
		s.XDSServer.Start(stop)
		return nil
	})
}

func (s *Server) initSecureDiscoveryService(args *DubboArgs, trustDomain string) error {
	if args.ServerOptions.SecureGRPCAddr == "" {
		log.Info("The secure discovery port is disabled, multiplexing on httpAddr ")
		return nil
	}

	peerCertVerifier, err := s.createPeerCertVerifier(args.ServerOptions.TLSOptions, trustDomain)
	if err != nil {
		return err
	}
	if peerCertVerifier == nil {
		// Running locally without configured certs - no TLS mode
		log.Warnf("The secure discovery service is disabled")
		return nil
	}
	log.Info("initializing secure discovery service")

	cfg := &tls.Config{
		GetCertificate: s.getDubbodCertificate,
		ClientAuth:     tls.VerifyClientCertIfGiven,
		ClientCAs:      peerCertVerifier.GetGeneralCertPool(),
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			err := peerCertVerifier.VerifyPeerCert(rawCerts, verifiedChains)
			if err != nil {
				log.Infof("Could not verify certificate: %v", err)
			}
			return err
		},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: args.ServerOptions.TLSOptions.CipherSuits,
	}
	// Compliance for xDS server TLS.
	sec_model.EnforceGoCompliance(cfg)

	tlsCreds := credentials.NewTLS(cfg)

	s.secureGrpcAddress = args.ServerOptions.SecureGRPCAddr

	interceptors := []grpc.UnaryServerInterceptor{
		// setup server prometheus monitoring (as final interceptor in chain)
		grpcprom.UnaryServerInterceptor,
	}
	opts := dubbogrpc.ServerOptions(args.KeepaliveOptions, interceptors...)
	opts = append(opts, grpc.Creds(tlsCreds))

	s.secureGrpcServer = grpc.NewServer(opts...)
	s.XDSServer.Register(s.secureGrpcServer)
	reflection.Register(s.secureGrpcServer)

	s.addStartFunc("secure gRPC", func(stop <-chan struct{}) error {
		go func() {
			<-stop
			s.secureGrpcServer.Stop()
		}()
		return nil
	})

	return nil
}

func (s *Server) initRegistryEventHandlers() {
	log.Info("initializing registry event handlers")

	if s.configController == nil {
		log.Warnf("configController is nil, cannot register event handlers")
		return
	}

	log.Debugf("configController is available, registering event handlers")

	configHandler := func(prev config.Config, curr config.Config, event model.Event) {
		// Log ALL events at INFO level to ensure visibility
		log.Infof("configHandler: received event %s for config %v (prev.Name=%s, curr.Name=%s, prev.Namespace=%s, curr.Namespace=%s)",
			event, curr.GroupVersionKind, prev.Name, curr.Name, prev.Namespace, curr.Namespace)

		// Handle delete events - use prev config if curr is empty
		cfg := curr
		if event == model.EventDelete && curr.Name == "" {
			cfg = prev
		}

		// Build ConfigKey for the changed config
		// Find the schema to get the kind.Kind
		// First try Dubbo schemas, then try DubboGatewayAPI schemas
		schema, found := collections.Dubbo.FindByGroupVersionKind(cfg.GroupVersionKind)
		if !found && features.EnableGatewayAPI {
			schema, found = collections.DubboGatewayAPI().FindByGroupVersionKind(cfg.GroupVersionKind)
		}
		if !found {
			log.Warnf("configHandler: schema not found for %v, skipping", cfg.GroupVersionKind)
			return
		}

		// Map GVK to kind.Kind using schema identifier
		// This matches Dubbo's approach of using gvk.MustToKind, but we use schema.Identifier() instead
		schemaID := schema.Identifier()
		log.Infof("configHandler: processing config change, schema identifier=%s, GVK=%v, name=%s/%s, event=%s",
			schemaID, cfg.GroupVersionKind, cfg.Namespace, cfg.Name, event)

		var configKind kind.Kind
		switch schemaID {
		case "DestinationRule":
			configKind = kind.DestinationRule
		case "virtualService", "VirtualService":
			configKind = kind.VirtualService
		case "PeerAuthentication":
			configKind = kind.PeerAuthentication
		case "GatewayClass":
			configKind = kind.GatewayClass
		case "Gateway":
			configKind = kind.Gateway
		case "HTTPRoute":
			configKind = kind.HTTPRoute
		default:
			log.Debugf("configHandler: unknown schema identifier %s for %v, skipping", schemaID, cfg.GroupVersionKind)
			return
		}

		configKey := model.ConfigKey{
			Kind:      configKind,
			Name:      cfg.Name,
			Namespace: cfg.Namespace,
		}

		// Log the config change
		log.Infof("configHandler: %s event for %s/%s/%s", event, configKey.Kind, configKey.Namespace, configKey.Name)

		// Some configs (DestinationRule/VirtualService/PeerAuthentication/HTTPRoute) require Full push to ensure
		// PushContext is re-initialized and configuration is reloaded.
		// PeerAuthentication must rebuild AuthenticationPolicies to enable STRICT mTLS on LDS; without
		// a full push the cached PushContext would continue serving plaintext listeners.
		// HTTPRoute must rebuild HTTPRoute index to enable Gateway API routing.
		needsFullPush := configKind == kind.DestinationRule || configKind == kind.VirtualService || configKind == kind.PeerAuthentication || configKind == kind.HTTPRoute

		// Trigger ConfigUpdate to push changes to all connected proxies
		s.XDSServer.ConfigUpdate(&model.PushRequest{
			ConfigsUpdated: sets.New(configKey),
			Reason:         model.NewReasonStats(model.DependentResource),
			Full:           needsFullPush, // Full push for DestinationRule/VirtualService to reload PushContext
		})
	}
	schemas := collections.Dubbo.All()
	if features.EnableGatewayAPI {
		schemas = collections.DubboGatewayAPI().All()
	}
	log.Debugf("found %d schemas to register", len(schemas))
	registeredCount := 0
	for _, schema := range schemas {
		gvk := schema.GroupVersionKind()
		schemaID := schema.Identifier()
		log.Debugf("registering event handler for %s (GVK: %v)", schemaID, gvk)
		s.configController.RegisterEventHandler(gvk, configHandler)
		registeredCount++
		log.Debugf("successfully registered event handler for %s (GVK: %v)", schemaID, gvk)
	}
	log.Debugf("successfully registered event handlers for %d schemas", registeredCount)
}

func (s *Server) addReadinessProbe(name string, fn readinessProbe) {
	s.readinessProbes[name] = fn
}

func (s *Server) initReadinessProbes() {
	probes := map[string]readinessProbe{
		"discovery": func() bool {
			return s.XDSServer.IsServerReady()
		},
		"proxyless injector": func() bool {
			return s.readinessFlags.InjectorReady.Load()
		},
		"config validation": func() bool {
			return s.readinessFlags.configValidationReady.Load()
		},
	}
	for name, probe := range probes {
		s.addReadinessProbe(name, probe)
	}
}

func (s *Server) initMulticluster(args *DubboArgs) {
	if s.kubeClient == nil {
		return
	}
	s.multiclusterController = multicluster.NewController(s.kubeClient, args.Namespace, s.clusterID, s.environment.Watcher, func(r *rest.Config) {
		r.QPS = args.RegistryOptions.KubeOptions.KubernetesAPIQPS
		r.Burst = args.RegistryOptions.KubeOptions.KubernetesAPIBurst
	})
	// TODO ListRemoteClusters
	s.addStartFunc("multicluster controller", func(stop <-chan struct{}) error {
		return s.multiclusterController.Run(stop)
	})
}

func (s *Server) initMeshHandlers(changeHandler func(_ *meshv1alpha1.MeshGlobalConfig)) {
	log.Info("initializing mesh handlers")
	// When the mesh config or networks change, do a full push.
	s.environment.AddMeshHandler(func() {
		changeHandler(s.environment.Mesh())
		s.XDSServer.ConfigUpdate(&model.PushRequest{
			Full:   true,
			Reason: model.NewReasonStats(model.GlobalUpdate),
			Forced: true,
		})
	})
}

func (s *Server) initKubeClient(args *DubboArgs) error {
	if s.kubeClient != nil {
		// Already initialized by startup arguments
		return nil
	}
	hasK8SConfigStore := false
	if args.RegistryOptions.FileDir == "" {
		// If file dir is set - config controller will just use file.
		if _, err := os.Stat(args.MeshGlobalConfigFile); !os.IsNotExist(err) {
			meshGlobalConfig, err := mesh.ReadMeshGlobalConfig(args.MeshGlobalConfigFile)
			if err != nil {
				return fmt.Errorf("failed reading mesh config: %v", err)
			}
			if len(meshGlobalConfig.ConfigSources) == 0 && args.RegistryOptions.KubeConfig != "" {
				hasK8SConfigStore = true
			}
			for _, cs := range meshGlobalConfig.ConfigSources {
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
		// Used by validation
		kubeRestConfig, err := kubelib.DefaultRestConfig(args.RegistryOptions.KubeConfig, "", func(config *rest.Config) {
			config.QPS = args.RegistryOptions.KubeOptions.KubernetesAPIQPS
			config.Burst = args.RegistryOptions.KubeOptions.KubernetesAPIBurst
		})
		if err != nil {
			return fmt.Errorf("failed creating kube config: %v", err)
		}

		s.kubeClient, err = kubelib.NewClient(kubelib.NewClientConfigForRestConfig(kubeRestConfig), s.clusterID)
		if err != nil {
			return fmt.Errorf("failed creating kube client: %v", err)
		}
		s.kubeClient = kubelib.EnableCrdWatcher(s.kubeClient)
	}

	return nil
}

func (s *Server) initControllers(args *DubboArgs) error {
	log.Info("initializing controllers")

	s.initMulticluster(args)

	s.initSDSServer()

	if err := s.initConfigController(args); err != nil {
		return fmt.Errorf("error initializing config controller: %v", err)
	}
	if err := s.initServiceControllers(args); err != nil {
		return fmt.Errorf("error initializing service controllers: %v", err)
	}
	return nil
}

func (s *Server) initServers(args *DubboArgs) {
	s.initGrpcServer(args.KeepaliveOptions)
	multiplexGRPC := false
	if args.ServerOptions.GRPCAddr != "" {
		s.grpcAddress = args.ServerOptions.GRPCAddr
	} else {
		// This happens only if the GRPC port (15010) is disabled. We will multiplex
		// it on the HTTP port. Does not impact the HTTPS gRPC or HTTPS.
		log.Infof("multiplexing gRPC on http addr %v", args.ServerOptions.HTTPAddr)
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
	interceptors := []grpc.UnaryServerInterceptor{
		// setup server prometheus monitoring (as final interceptor in chain)
		grpcprom.UnaryServerInterceptor,
	}
	grpcOptions := dubbogrpc.ServerOptions(options, interceptors...)
	s.grpcServer = grpc.NewServer(grpcOptions...)
	s.XDSServer.Register(s.grpcServer)
	reflection.Register(s.grpcServer)
}

func (s *Server) createPeerCertVerifier(tlsOptions TLSOptions, trustDomain string) (*spiffe.PeerCertVerifier, error) {
	customTLSCertsExists, _, _, caCertPath := hasCustomTLSCerts(tlsOptions)
	if !customTLSCertsExists && s.CA == nil && !s.isK8SSigning() {
		// Running locally without configured certs - no TLS mode
		return nil, nil
	}
	peerCertVerifier := spiffe.NewPeerCertVerifier()
	var rootCertBytes []byte
	var err error
	if caCertPath != "" {
		if rootCertBytes, err = os.ReadFile(caCertPath); err != nil {
			return nil, err
		}
	} else {
		if s.RA != nil {
			if strings.HasPrefix(features.DubboCertProvider, constants.CertProviderKubernetesSignerPrefix) {
				signerName := strings.TrimPrefix(features.DubboCertProvider, constants.CertProviderKubernetesSignerPrefix)
				caBundle, _ := s.RA.GetRootCertFromMeshGlobalConfig(signerName)
				rootCertBytes = append(rootCertBytes, caBundle...)
			} else {
				rootCertBytes = append(rootCertBytes, s.RA.GetCAKeyCertBundle().GetRootCertPem()...)
			}
		}
		if s.CA != nil {
			rootCertBytes = append(rootCertBytes, s.CA.GetCAKeyCertBundle().GetRootCertPem()...)
		}
	}

	if len(rootCertBytes) != 0 {
		// TODO: trustDomain here is static and will not update if it dynamically changes in mesh config
		err := peerCertVerifier.AddMappingFromPEM(trustDomain, rootCertBytes)
		if err != nil {
			return nil, fmt.Errorf("add root CAs into peerCertVerifier failed: %v", err)
		}
	}

	return peerCertVerifier, nil
}

// maybeCreateCA creates and initializes the built-in CA if needed.
func (s *Server) maybeCreateCA(caOpts *caOptions) error {
	// CA signing certificate must be created only if CA is enabled.
	if features.EnableCAServer {
		log.Info("creating CA and initializing public key")
		var err error
		if useRemoteCerts.Get() {
			if err = s.loadCACerts(caOpts, LocalCertDir.Get()); err != nil {
				return fmt.Errorf("failed to load remote CA certs: %v", err)
			}
		}
		// May return nil, if the CA is missing required configs - This is not an error.
		// This is currently only used for K8S signing.
		if caOpts.ExternalCAType != "" {
			if s.RA, err = s.createDubboRA(caOpts); err != nil {
				return fmt.Errorf("failed to create RA: %v", err)
			}
		}
		// If K8S signs - we don't need to use the built-in dubbo CA.
		if !s.isK8SSigning() {
			if s.CA, err = s.createDubboCA(caOpts); err != nil {
				return fmt.Errorf("failed to create CA: %v", err)
			}
		}
	}
	return nil
}

func (s *Server) getDubbodCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	s.certMu.RLock()
	defer s.certMu.RUnlock()
	if s.dubbodCert != nil {
		return s.dubbodCert, nil
	}
	return nil, fmt.Errorf("cert not initialized")
}

func (s *Server) WaitUntilCompletion() {
	s.server.Wait()
}

func (s *Server) serveHTTP() error {
	// At this point we are ready - start Http Listener so that it can respond to readiness events.
	httpListener, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return err
	}
	go func() {
		log.Infof("starting HTTP service at %s", httpListener.Addr())
		if err := s.httpServer.Serve(httpListener); network.IsUnexpectedListenerError(err) {
			log.Errorf("error serving http server: %v", err)
		}
	}()
	s.httpAddr = httpListener.Addr().String()
	return nil
}

// addStartFunc appends a function to be run. These are run synchronously in order,
// so the function should start a go routine if it needs to do anything blocking
func (s *Server) addStartFunc(name string, fn server.Component) {
	s.server.RunComponent(name, fn)
}

func getClusterID(args *DubboArgs) cluster.ID {
	clusterID := args.RegistryOptions.KubeOptions.ClusterID
	if clusterID == "" {
		if hasKubeRegistry(args.RegistryOptions.Registries) {
			clusterID = cluster.ID(provider.Kubernetes)
		}
	}
	return clusterID
}

func (s *Server) initSDSServer() {
	if s.kubeClient == nil {
		return
	}
	if !features.EnableXDSIdentityCheck {
		// Make sure we have security
		log.Warnf("skipping Kubernetes credential reader; DUBBO_ENABLE_XDS_IDENTITY_CHECK must be set to true for this feature.")
	} else {
		// TODO ConfigUpdated Multicluster get secret and configmap
	}
}

// isK8SSigning returns whether K8S (as a RA) is used to sign certs instead of private keys known by Dubbod
func (s *Server) isK8SSigning() bool {
	return s.RA != nil && strings.HasPrefix(features.DubboCertProvider, constants.CertProviderKubernetesSignerPrefix)
}

func (s *Server) cachesSynced() bool {
	// TODO multiclusterController HasSynced
	if !s.ServiceController().HasSynced() {
		return false
	}
	if !s.configController.HasSynced() {
		return false
	}
	return true
}

func (s *Server) waitForShutdown(stop <-chan struct{}) {
	go func() {
		<-stop
		close(s.internalStop)
		_ = s.fileWatcher.Close()

		if s.cacertsWatcher != nil {
			_ = s.cacertsWatcher.Close()
		}
		// Stop gRPC services.  If gRPC services fail to stop in the shutdown duration,
		// force stop them. This does not happen normally.
		stopped := make(chan struct{})
		go func() {
			// Some grpcServer implementations do not support GracefulStop. Unfortunately, this is not
			// exposed; they just panic. To avoid this, we will recover and do a standard Stop when its not
			// support.
			defer func() {
				if r := recover(); r != nil {
					s.grpcServer.Stop()
					if s.secureGrpcServer != nil {
						s.secureGrpcServer.Stop()
					}
					close(stopped)
				}
			}()
			s.grpcServer.GracefulStop()
			if s.secureGrpcServer != nil {
				s.secureGrpcServer.GracefulStop()
			}
			close(stopped)
		}()

		t := time.NewTimer(s.shutdownDuration)
		select {
		case <-t.C:
			s.grpcServer.Stop()
			if s.secureGrpcServer != nil {
				s.secureGrpcServer.Stop()
			}
		case <-stopped:
			t.Stop()
		}

		// Stop HTTP services.
		ctx, cancel := context.WithTimeout(context.Background(), s.shutdownDuration)
		defer cancel()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Error(err)
		}
		if s.httpsServer != nil {
			if err := s.httpsServer.Shutdown(ctx); err != nil {
				log.Error(err)
			}
		}

		// Shutdown the DiscoveryServer.
		s.XDSServer.Shutdown()
	}()
}

func (s *Server) pushContextReady(expected int64) bool {
	committed := s.XDSServer.CommittedUpdates.Load()
	if committed < expected {
		log.Debugf("Waiting for pushcontext to process inbound updates, inbound: %v, committed : %v", expected, committed)
		return false
	}
	return true
}

func (s *Server) waitForCacheSync(stop <-chan struct{}) bool {
	start := time.Now()
	log.Info("Waiting for caches to be synced")
	if !kubelib.WaitForCacheSync("server", stop, s.cachesSynced) {
		log.Errorf("Failed waiting for cache sync")
		return false
	}
	log.Infof("All controller caches have been synced up in %v", time.Since(start))
	expected := s.XDSServer.InboundUpdates.Load()
	return kubelib.WaitForCacheSync("push context", stop, func() bool { return s.pushContextReady(expected) })
}

func (s *Server) initDubbodCerts(args *DubboArgs, host string) error {
	// Skip all certificates
	var err error

	s.dnsNames = getDNSNames(args, host)
	if hasCustomCertArgsOrWellKnown, tlsCertPath, tlsKeyPath, caCertPath := hasCustomTLSCerts(args.ServerOptions.TLSOptions); hasCustomCertArgsOrWellKnown {
		// Use the DNS certificate provided via args or in well known location.
		err = s.initFileCertificateWatches(TLSOptions{
			CaCertFile: caCertPath,
			KeyFile:    tlsKeyPath,
			CertFile:   tlsCertPath,
		})
		if err != nil {
			// Not crashing dubbod - This typically happens if certs are missing and in tests.
			log.Errorf("error initializing certificate watches: %v", err)
			return nil
		}
	} else if features.EnableCAServer && features.DubboCertProvider == constants.CertProviderDubbod {
		log.Infof("initializing Dubbod DNS certificates host: %s, custom host: %s", host, features.DubbodServiceCustomHost)
		err = s.initDNSCertsDubbod()
	} else if features.DubboCertProvider == constants.CertProviderKubernetes {
		log.Warnf("DUBBO_CERT_PROVIDER=kubernetes is no longer supported by upstream K8S")
	} else if strings.HasPrefix(features.DubboCertProvider, constants.CertProviderKubernetesSignerPrefix) {
		log.Infof("initializing Dubbod DNS certificates using K8S RA:%s  host: %s, custom host: %s", features.DubboCertProvider,
			host, features.DubbodServiceCustomHost)
		err = s.initDNSCertsK8SRA()
	} else {
		log.Warnf("DUBBO_CERT_PROVIDER=%s is not implemented", features.DubboCertProvider)
	}

	if err == nil {
		err = s.initDubbodCertLoader()
	}

	return err
}

func (s *Server) dubbodReadyHandler(w http.ResponseWriter, _ *http.Request) {
	for name, fn := range s.readinessProbes {
		if ready := fn(); !ready {
			log.Warnf("%s is not ready", name)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) shouldStartNsController() bool {
	if s.isK8SSigning() {
		// Need to distribute the roots from MeshGlobalConfig
		return true
	}
	if s.CA == nil {
		return false
	}

	// For no CA we don't distribute it either, as there is no cert
	if features.DubboCertProvider == constants.CertProviderNone {
		return false
	}

	return true
}

func (s *Server) initStatusManager(_ *DubboArgs) {
	s.addStartFunc("status manager", func(stop <-chan struct{}) error {
		s.statusManager = status.NewManager(s.RWConfigStore)
		s.statusManager.Start(stop)
		return nil
	})
}

func getDNSNames(_ *DubboArgs, host string) []string {
	// Append custom hostname if there is any
	customHost := features.DubbodServiceCustomHost
	var cHosts []string

	if customHost != "" {
		cHosts = strings.Split(customHost, ",")
	}
	sans := sets.New(cHosts...)
	sans.Insert(host)
	dnsNames := sets.SortedList(sans)
	log.Infof("Discover server subject alt names: %v", dnsNames)
	return dnsNames
}

func hasCustomTLSCerts(tlsOptions TLSOptions) (ok bool, tlsCertPath, tlsKeyPath, caCertPath string) {
	// load from tls args as priority
	if hasCustomTLSCertArgs(tlsOptions) {
		return true, tlsOptions.CertFile, tlsOptions.KeyFile, tlsOptions.CaCertFile
	}

	if ok = checkPathsExist(constants.DefaultDubboTLSCert, constants.DefaultDubboTLSKey, constants.DefaultDubboTLSCaCert); ok {
		tlsCertPath = constants.DefaultDubboTLSCert
		tlsKeyPath = constants.DefaultDubboTLSKey
		caCertPath = constants.DefaultDubboTLSCaCert
		return
	}

	if ok = checkPathsExist(constants.DefaultDubboTLSCert, constants.DefaultDubboTLSKey, constants.DefaultDubboTLSCaCertAlternatePath); ok {
		tlsCertPath = constants.DefaultDubboTLSCert
		tlsKeyPath = constants.DefaultDubboTLSKey
		caCertPath = constants.DefaultDubboTLSCaCertAlternatePath
		return
	}

	return
}

func checkPathsExist(paths ...string) bool {
	for _, path := range paths {
		fInfo, err := os.Stat(path)

		if err != nil || fInfo.IsDir() {
			return false
		}
	}
	return true
}

func hasCustomTLSCertArgs(tlsOptions TLSOptions) bool {
	return tlsOptions.CaCertFile != "" && tlsOptions.CertFile != "" && tlsOptions.KeyFile != ""
}

func (s *Server) initDubbodCertLoader() error {
	if err := s.loadDubbodCert(); err != nil {
		return fmt.Errorf("first time load DubbodCert failed: %v", err)
	}
	_, watchCh := s.dubbodCertBundleWatcher.AddWatcher()
	s.addStartFunc("reload certs", func(stop <-chan struct{}) error {
		go s.reloadDubbodCert(watchCh, stop)
		return nil
	})
	return nil
}

func (s *Server) loadDubbodCert() error {
	keyCertBundle := s.dubbodCertBundleWatcher.GetKeyCertBundle()
	keyPair, err := tls.X509KeyPair(keyCertBundle.CertPem, keyCertBundle.KeyPem)
	if err != nil {
		return fmt.Errorf("dubbod loading x509 key pairs failed: %v", err)
	}
	for _, c := range keyPair.Certificate {
		x509Cert, err := x509.ParseCertificates(c)
		if err != nil {
			// This can rarely happen, just in case.
			return fmt.Errorf("x509 cert - ParseCertificates() error: %v", err)
		}
		for _, c := range x509Cert {
			log.Infof("x509 cert - Issuer: %q, Subject: %q, SN: %x, NotBefore: %q, NotAfter: %q",
				c.Issuer, c.Subject, c.SerialNumber,
				c.NotBefore.Format(time.RFC3339), c.NotAfter.Format(time.RFC3339))
		}
	}

	log.Info("Dubbod certificates are reloaded")
	s.certMu.Lock()
	s.dubbodCert = &keyPair
	s.certMu.Unlock()
	return nil
}

func (s *Server) reloadDubbodCert(watchCh <-chan struct{}, stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		case <-watchCh:
			if err := s.loadDubbodCert(); err != nil {
				log.Errorf("reload dubbod cert failed: %v", err)
			}
		}
	}
}

func (s *Server) addTerminatingStartFunc(name string, fn server.Component) {
	s.server.RunComponentAsyncAndWait(name, fn)
}
