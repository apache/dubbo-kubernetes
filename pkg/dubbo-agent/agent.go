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

package dubboagent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/log"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/cmd/planet-agent/config"
	"github.com/apache/dubbo-kubernetes/pkg/model"

	mesh "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/nodeagent/cache"
	"github.com/apache/dubbo-kubernetes/pkg/backoff"
	"github.com/apache/dubbo-kubernetes/pkg/bootstrap"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo-agent/grpcxds"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo-agent/pixiu"
	"github.com/apache/dubbo-kubernetes/pkg/filewatcher"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	serviceNodeSeparator = "~"
)

const (
	AegisCACertPath = "./var/run/secrets/dubbo"
)

const (
	MetadataClientCertKey   = "DUBBO_META_TLS_CLIENT_KEY"
	MetadataClientCertChain = "DUBBO_META_TLS_CLIENT_CERT_CHAIN"
	MetadataClientRootCert  = "DUBBO_META_TLS_CLIENT_ROOT_CERT"
)

type SDSServiceFactory = func(_ *security.Options, _ security.SecretManager) SDSService

type SDSService interface {
	OnSecretUpdate(resourceName string)
	Stop()
}

type Proxy struct {
	ID          string
	DNSDomain   string
	IPAddresses []string
	Type        model.NodeType
	ipMode      model.IPMode
}

type Agent struct {
	proxyConfig              *mesh.ProxyConfig
	cfg                      *AgentOptions
	EnableDynamicProxyConfig bool
	secOpts                  *security.Options
	sdsServer                SDSService
	secretCache              *cache.SecretManagerClient
	sdsMu                    sync.Mutex

	xdsProxy    *XdsProxy
	fileWatcher filewatcher.FileWatcher
	statusSrv   *http.Server

	// Pixiu agent for router mode (Gateway Pods)
	pixiuAgent *pixiu.Agent

	wg sync.WaitGroup
}

type AgentOptions struct {
	WorkloadIdentitySocketFile string
	GRPCBootstrapPath          string
	XDSHeaders                 map[string]string
	XdsUdsPath                 string
	XDSRootCerts               string
	ProxyIPAddresses           []string
	ProxyDomain                string
	EnableDynamicProxyConfig   bool
	ServiceNode                string
	MetadataDiscovery          *bool
	CARootCerts                string
	DubbodSAN                  string
	DownstreamGrpcOptions      []grpc.ServerOption
	ProxyType                  model.NodeType
	SDSFactory                 func(options *security.Options, workloadSecretCache security.SecretManager) SDSService
}

func NewAgent(proxyConfig *mesh.ProxyConfig, agentOpts *AgentOptions, sopts *security.Options) *Agent {
	return &Agent{
		proxyConfig: proxyConfig,
		cfg:         agentOpts,
		secOpts:     sopts,
		fileWatcher: filewatcher.NewWatcher(),
	}
}

func (a *Agent) Run(ctx context.Context) (func(), error) {
	// TODO initLocalDNSServer?

	if a.cfg.WorkloadIdentitySocketFile != filepath.Base(a.cfg.WorkloadIdentitySocketFile) {
		return nil, fmt.Errorf("workload identity socket file override must be a filename, not a path: %s", a.cfg.WorkloadIdentitySocketFile)
	}

	configuredAgentSocketPath := security.GetWorkloadSDSSocketListenPath(a.cfg.WorkloadIdentitySocketFile)

	isDubboSDS := configuredAgentSocketPath == security.GetDubboSDSServerSocketPath()

	socketExists, err := checkSocket(ctx, configuredAgentSocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check SDS socket: %v", err)
	}
	if socketExists {
		log.Infof("Existing workload SDS socket found at %s. Default Dubbo SDS Server will only serve files", configuredAgentSocketPath)
		a.secOpts.ServeOnlyFiles = true
	} else if !isDubboSDS {
		return nil, fmt.Errorf("agent configured for non-default SDS socket path: %s but no socket found", configuredAgentSocketPath)
	}

	log.Info("Starting default Dubbo SDS Server")
	err = a.initSdsServer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start default Dubbo SDS server: %v", err)
	}
	a.xdsProxy, err = initXdsProxy(a)
	if err != nil {
		return nil, fmt.Errorf("failed to start xds proxy: %v", err)
	}

	var bootstrapNode *core.Node
	if a.cfg.GRPCBootstrapPath != "" {
		log.Infof("Starting planet-agent with GRPC bootstrap path: %s", a.cfg.GRPCBootstrapPath)
		node, err := a.generateGRPCBootstrapWithNode()
		if err != nil {
			return nil, fmt.Errorf("failed generating gRPC XDS bootstrap: %v", err)
		}
		// Prepare Node for upstream connection, but don't set it yet
		// We'll set it after status port starts and certificates are generated
		if node != nil && a.xdsProxy != nil {
			bootstrapNode = &core.Node{
				Id:       node.ID,
				Locality: node.Locality,
			}
			if node.Metadata != nil {
				bytes, _ := json.Marshal(node.Metadata)
				rawMeta := map[string]any{}
				if err := json.Unmarshal(bytes, &rawMeta); err == nil {
					if metaStruct, err := structpb.NewStruct(rawMeta); err == nil {
						bootstrapNode.Metadata = metaStruct
					}
				}
			}
		}
	} else {
		log.Warn("GRPC_XDS_BOOTSTRAP not set, bootstrap file will not be generated")
	}
	if a.proxyConfig.ControlPlaneAuthPolicy != mesh.AuthenticationPolicy_NONE {
		rootCAForXDS, err := a.FindRootCAForXDS()
		if err != nil {
			return nil, fmt.Errorf("failed to find root XDS CA: %v", err)
		}
		go a.startFileWatcher(ctx, rootCAForXDS, func() {
			if err := a.xdsProxy.initDubbodDialOptions(a); err != nil {
				log.Warnf("Failed to update xds proxy dial options: %v", err)
			} else {
				log.Info("Updated xds proxy dial options after certificate change")
			}
		})

		// also watch CA root for CA client; rebuild SDS/CA client when it changes
		if caRoot, err := a.FindRootCAForCA(); err == nil && caRoot != "" {
			go a.startFileWatcher(ctx, caRoot, func() {
				log.Info("CA root changed, rebuilding CA client and SDS server")
				a.rebuildSDSWithNewCAClient()
			})
		}
	}

	// start status HTTP server for health checks
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	a.statusSrv = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.proxyConfig.StatusPort),
		Handler: mux,
	}
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		log.Infof("Opening status port %d", a.proxyConfig.StatusPort)
		if err := a.statusSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("status server error: %v", err)
		}
	}()

	// Now set bootstrap node to trigger upstream connection.
	// This ensures upstream connection logs appear after certificate logs.
	if bootstrapNode != nil && a.xdsProxy != nil {
		a.xdsProxy.SetBootstrapNode(bootstrapNode)
	}

	// Initialize and start Pixiu for router mode (Gateway Pods)
	if a.cfg.ProxyType == model.Router {
		if err := a.initializePixiuAgent(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize Pixiu agent: %v", err)
		}
	}

	return a.wg.Wait, nil
}

func (a *Agent) Close() {
	if a.xdsProxy != nil {
		a.xdsProxy.close()
	}
	if a.sdsServer != nil {
		a.sdsServer.Stop()
	}
	if a.secretCache != nil {
		a.secretCache.Close()
	}
	if a.fileWatcher != nil {
		_ = a.fileWatcher.Close()
	}
	if a.statusSrv != nil {
		_ = a.statusSrv.Shutdown(context.Background())
	}
}

func (a *Agent) FindRootCAForXDS() (string, error) {
	var rootCAPath string

	if a.cfg.XDSRootCerts == security.SystemRootCerts {
		// Special case input for root cert configuration to use system root certificates
		return "", nil
	} else if a.cfg.XDSRootCerts != "" {
		// Using specific platform certs or custom roots
		rootCAPath = a.cfg.XDSRootCerts
	} else if fileExists(security.DefaultRootCertFilePath) {
		// Old style - mounted cert. This is used for XDS auth only,
		// not connecting to CA_ADDR because this mode uses external
		// agent (Secret refresh, etc)
		return security.DefaultRootCertFilePath, nil
	} else if a.secOpts.ProvCert != "" {
		// This was never completely correct - PROV_CERT are only intended for auth with CA_ADDR,
		// and should not be involved in determining the root CA.
		// For VMs, the root cert file used to auth may be populated afterwards.
		// Thus, return directly here and skip checking for existence.
		return a.secOpts.ProvCert + "/root-cert.pem", nil
	} else if a.secOpts.FileMountedCerts {
		// FileMountedCerts - Load it from Proxy Metadata.
		rootCAPath = a.proxyConfig.ProxyMetadata[MetadataClientRootCert]
	} else if a.secOpts.PlanetCertProvider == constants.CertProviderNone {
		return "", fmt.Errorf("root CA file for XDS required but configured provider as none")
	} else {
		rootCAPath = path.Join(AegisCACertPath, constants.CACertNamespaceConfigMapDataName)
	}

	// Additional checks for root CA cert existence. Fail early, instead of obscure envoy errors
	if fileExists(rootCAPath) {
		return rootCAPath, nil
	}

	return "", fmt.Errorf("root CA file for XDS does not exist %s", rootCAPath)
}

func (a *Agent) GetKeyCertsForXDS() (string, string) {
	var key, cert string
	if a.secOpts.ProvCert != "" {
		key, cert = getKeyCertInner(a.secOpts.ProvCert)
	} else if a.secOpts.FileMountedCerts {
		key = a.proxyConfig.ProxyMetadata[MetadataClientCertKey]
		cert = a.proxyConfig.ProxyMetadata[MetadataClientCertChain]
	}
	return key, cert
}

func (a *Agent) GetKeyCertsForCA() (string, string) {
	var key, cert string
	if a.secOpts.ProvCert != "" {
		key, cert = getKeyCertInner(a.secOpts.ProvCert)
	}
	return key, cert
}

func (a *Agent) FindRootCAForCA() (string, error) {
	var rootCAPath string

	if a.cfg.CARootCerts == security.SystemRootCerts {
		return "", nil
	} else if a.cfg.CARootCerts != "" {
		rootCAPath = a.cfg.CARootCerts
	} else if a.secOpts.PlanetCertProvider == constants.CertProviderCustom {
		rootCAPath = security.DefaultRootCertFilePath // ./etc/certs/root-cert.pem
	} else if a.secOpts.ProvCert != "" {
		// This was never completely correct - PROV_CERT are only intended for auth with CA_ADDR,
		// and should not be involved in determining the root CA.
		// For VMs, the root cert file used to auth may be populated afterwards.
		// Thus, return directly here and skip checking for existence.
		return a.secOpts.ProvCert + "/root-cert.pem", nil
	} else if a.secOpts.PlanetCertProvider == constants.CertProviderNone {
		return "", fmt.Errorf("root CA file for CA required but configured provider as none")
	} else {
		rootCAPath = path.Join(AegisCACertPath, constants.CACertNamespaceConfigMapDataName)
	}

	if fileExists(rootCAPath) {
		return rootCAPath, nil
	}

	return "", fmt.Errorf("root CA file for CA does not exist %s", rootCAPath)
}

func (a *Agent) startFileWatcher(ctx context.Context, filePath string, handler func()) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		log.Warnf("Failed to get absolute path for %s: %v", filePath, err)
		return
	}
	// Ensure parent directory exists for filewatcher
	parentDir := filepath.Dir(absPath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			log.Warnf("Failed to create parent directory %s for file watcher: %v", parentDir, err)
			return
		}
	}
	// Filewatcher watches the parent directory, so the file doesn't need to exist yet
	if err := a.fileWatcher.Add(absPath); err != nil {
		// If the file is already being watched, this is expected and should be silently skipped
		// Only log as warning if it's a different error
		if strings.Contains(err.Error(), "is already being watched") {
			log.Debugf("File watcher already exists for %s, skipping", absPath)
			return
		}
		log.Warnf("Failed to add file watcher %s: %v", absPath, err)
		return
	}

	log.Debugf("Add file %s watcher", absPath)
	for {
		select {
		case gotEvent := <-a.fileWatcher.Events(absPath):
			log.Debugf("Receive file %s event %v", absPath, gotEvent)
			handler()
		case err := <-a.fileWatcher.Errors(absPath):
			log.Warnf("Watch file %s error: %v", absPath, err)
		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) initSdsServer(ctx context.Context) error {
	var err error
	if security.CheckWorkloadCertificate(security.WorkloadIdentityCertChainPath, security.WorkloadIdentityKeyPath, security.WorkloadIdentityRootCertPath) {
		log.Info("workload certificate files detected, creating secret manager without caClient")
		a.secOpts.RootCertFilePath = security.WorkloadIdentityRootCertPath
		a.secOpts.CertChainFilePath = security.WorkloadIdentityCertChainPath
		a.secOpts.KeyFilePath = security.WorkloadIdentityKeyPath
		a.secOpts.FileMountedCerts = true
	}

	createCaClient := !a.secOpts.FileMountedCerts && !a.secOpts.ServeOnlyFiles
	a.secretCache, err = a.newSecretManager(createCaClient)
	if err != nil {
		return fmt.Errorf("failed to start workload secret manager %v", err)
	}

	a.sdsServer = a.cfg.SDSFactory(a.secOpts, a.secretCache)
	return a.registerSecretHandler(ctx)
}

func (a *Agent) rebuildSDSWithNewCAClient() {
	a.sdsMu.Lock()
	defer a.sdsMu.Unlock()
	if a.sdsServer != nil {
		log.Info("Stopping existing SDS server for CA client rebuild")
		a.sdsServer.Stop()
	}
	if a.secretCache != nil {
		log.Info("Closing existing SecretManagerClient")
		a.secretCache.Close()
	}
	// recreate secret manager with CA client enabled
	sc, err := a.newSecretManager(true)
	if err != nil {
		log.Errorf("failed to recreate secret manager with new CA client: %v", err)
		return
	}
	a.secretCache = sc
	a.sdsServer = a.cfg.SDSFactory(a.secOpts, a.secretCache)
	if err := a.registerSecretHandler(context.Background()); err != nil {
		log.Errorf("failed to refresh workload certificates after CA rebuild: %v", err)
	} else {
		log.Info("SDS server and CA client rebuilt successfully")
	}
}

func (a *Agent) registerSecretHandler(ctx context.Context) error {
	if a.secretCache == nil {
		return nil
	}
	handler := func(resourceName string) {
		if a.sdsServer != nil {
			a.sdsServer.OnSecretUpdate(resourceName)
		}
		if resourceName == security.WorkloadKeyCertResourceName || resourceName == security.RootCertReqResourceName {
			go func() {
				if err := a.ensureWorkloadCertificates(context.Background()); err != nil {
					log.Warnf("failed to refresh workload certificates after %s update: %v", resourceName, err)
				}
			}()
		}
	}
	a.secretCache.RegisterSecretHandler(handler)
	return a.ensureWorkloadCertificates(ctx)
}

func (a *Agent) ensureWorkloadCertificates(ctx context.Context) error {
	if a.secretCache == nil || a.secOpts == nil || a.secOpts.OutputKeyCertToDir == "" {
		// Nothing to write
		return nil
	}
	generate := func(resource string) error {
		b := backoff.NewExponentialBackOff(backoff.DefaultOption())
		return b.RetryWithContext(ctx, func() error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			_, err := a.secretCache.GenerateSecret(resource)
			if err != nil {
				log.Warnf("failed to generate %s: %v", resource, err)
			}
			return err
		})
	}
	if err := generate(security.WorkloadKeyCertResourceName); err != nil {
		return err
	}
	return generate(security.RootCertReqResourceName)
}

func (a *Agent) generateGRPCBootstrapWithNode() (*model.Node, error) {
	// Convert relative path to absolute path for bootstrap file
	bootstrapPath := a.cfg.GRPCBootstrapPath
	absBootstrapPath, err := filepath.Abs(bootstrapPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for bootstrap file: %v", err)
	}
	log.Infof("Generating gRPC bootstrap file at: %s (absolute: %s)", bootstrapPath, absBootstrapPath)

	// generate metadata
	node, err := a.generateNodeMetadata()
	if err != nil {
		return nil, fmt.Errorf("failed generating node metadata: %v", err)
	}

	// GRPC bootstrap requires this. Original implementation injected this via env variable, but
	// this interfere with envoy, we should be able to use both envoy for TCP/HTTP and proxyless.
	node.Metadata.Generator = "grpc"

	if err := os.MkdirAll(filepath.Dir(absBootstrapPath), 0o700); err != nil {
		return nil, err
	}

	// Use absolute XdsUdsPath from xdsProxy (already converted in initXdsProxy)
	absUdsPath := a.xdsProxy.xdsUdsPath

	_, err = grpcxds.GenerateBootstrapFile(grpcxds.GenerateBootstrapOptions{
		Node:             node,
		XdsUdsPath:       absUdsPath,
		DiscoveryAddress: a.proxyConfig.DiscoveryAddress,
		CertDir:          a.secOpts.OutputKeyCertToDir,
	}, absBootstrapPath)
	if err != nil {
		return nil, err
	}
	log.Infof("gRPC bootstrap file generated successfully at: %s", absBootstrapPath)
	return node, nil
}

func (a *Agent) generateGRPCBootstrap() error {
	_, err := a.generateGRPCBootstrapWithNode()
	return err
}

func (a *Agent) newSecretManager(createCaClient bool) (*cache.SecretManagerClient, error) {
	if !createCaClient {
		log.Info("Workload is using file mounted certificates. Skipping connecting to CA")
		return cache.NewSecretManagerClient(nil, a.secOpts)
	}
	log.Infof("CA Endpoint %s, provider %s", a.secOpts.CAEndpoint, a.secOpts.CAProviderName)

	caClient, err := createCAClient(a.secOpts, a)
	if err != nil {
		return nil, err
	}
	return cache.NewSecretManagerClient(caClient, a.secOpts)
}

func (a *Agent) generateNodeMetadata() (*model.Node, error) {
	var planetSAN []string
	if a.proxyConfig.ControlPlaneAuthPolicy == mesh.AuthenticationPolicy_MUTUAL_TLS {
		planetSAN = []string{config.GetPlanetSan(a.proxyConfig.DiscoveryAddress)}
	}

	credentialSocketExists, err := checkSocket(context.TODO(), security.CredentialNameSocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check credential SDS socket: %v", err)
	}
	if credentialSocketExists {
		log.Info("Credential SDS socket found")
	}

	return bootstrap.GetNodeMetaData(bootstrap.MetadataOptions{
		ID:                     a.cfg.ServiceNode,
		Envs:                   os.Environ(),
		InstanceIPs:            a.cfg.ProxyIPAddresses,
		StsPort:                a.secOpts.STSPort,
		ProxyConfig:            a.proxyConfig,
		PlanetSubjectAltName:   planetSAN,
		CredentialSocketExists: credentialSocketExists,
		XDSRootCert:            a.cfg.XDSRootCerts,
		MetadataDiscovery:      a.cfg.MetadataDiscovery,
	})
}

func (node *Proxy) DiscoverIPMode() {
	node.ipMode = model.DiscoverIPMode(node.IPAddresses)
}

func (node *Proxy) ServiceNode() string {
	ip := ""
	if len(node.IPAddresses) > 0 {
		ip = node.IPAddresses[0]
	}
	return strings.Join([]string{
		string(node.Type), ip, node.ID, node.DNSDomain,
	}, serviceNodeSeparator)
}

func getKeyCertInner(certPath string) (string, string) {
	key := path.Join(certPath, constants.KeyFilename)
	cert := path.Join(certPath, constants.CertChainFilename)
	return key, cert
}

func fileExists(path string) bool {
	if fi, err := os.Stat(path); err == nil && fi.Mode().IsRegular() {
		return true
	}
	return false
}

func socketHealthCheck(ctx context.Context, socketPath string) error {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second))
	defer cancel()

	conn, err := grpc.DialContext(ctx, fmt.Sprintf("unix:%s", socketPath),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.FailOnNonTempDialError(true),
		grpc.WithReturnConnectionError(),
		grpc.WithBlock(),
	)
	if err != nil {
		return err
	}
	err = conn.Close()
	if err != nil {
		log.Infof("connection is not closed: %v", err)
	}

	return nil
}

func socketFileExists(path string) bool {
	if fi, err := os.Stat(path); err == nil && !fi.Mode().IsRegular() {
		return true
	}
	return false
}

func checkSocket(ctx context.Context, socketPath string) (bool, error) {
	socketExists := socketFileExists(socketPath)
	if !socketExists {
		return false, nil
	}

	err := socketHealthCheck(ctx, socketPath)
	if err != nil {
		log.Debugf("SDS socket detected but not healthy: %v", err)
		err = os.Remove(socketPath)
		if err != nil {
			return false, fmt.Errorf("existing SDS socket could not be removed: %v", err)
		}
		return false, nil
	}

	return true, nil
}

// initializePixiuAgent initializes and starts Pixiu agent for router mode
func (a *Agent) initializePixiuAgent(ctx context.Context) error {
	configPath := os.Getenv("PROXY_CONFIG_PATH")
	if configPath == "" {
		configPath = "/etc/pixiu/config/pixiu.yaml"
	}

	binaryPath := os.Getenv("PROXY_BINARY_PATH")
	if binaryPath == "" {
		possiblePaths := []string{
			"/usr/local/bin/pixiugateway",
			"pixiugateway",
		}

		found := false
		for _, path := range possiblePaths {
			if strings.Contains(path, "/") {
				// Check absolute path
				if _, err := os.Stat(path); err == nil {
					binaryPath = path
					found = true
					break
				}
			} else {
				// Check in PATH
				if fullPath, err := exec.LookPath(path); err == nil {
					binaryPath = fullPath
					found = true
					break
				}
			}
		}

		if !found {
			return fmt.Errorf("gateway binary pixiugateway not found. Please set PROXY_BINARY_PATH environment variable or ensure pixiu binary is installed at /usr/local/bin/pixiugateway")
		}
	}

	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("gateway binary not found at %s: %v", binaryPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create proxy config directory: %v", err)
	}

	pixiuProxy := pixiu.NewProxy(pixiu.ProxyConfig{
		ConfigPath:    configPath,
		ConfigCleanup: false, // Don't cleanup config file
		BinaryPath:    binaryPath,
	})

	terminationDrainDuration := 45 * time.Second // Default drain duration
	if a.proxyConfig.DrainDuration != nil {
		terminationDrainDuration = a.proxyConfig.DrainDuration.AsDuration()
	}
	minDrainDuration := 5 * time.Second

	a.pixiuAgent = pixiu.NewAgent(pixiuProxy, terminationDrainDuration, minDrainDuration)

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.pixiuAgent.Run(ctx)
	}()

	log.Infof("pixiu agent initialized for router mode, config path: %s, binary: %s", configPath, binaryPath)
	return nil
}
