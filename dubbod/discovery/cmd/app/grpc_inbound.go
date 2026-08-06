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

package app

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	neturl "net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	xdsresolver "github.com/kdubbo/xds-api/grpc/resolver"
	"github.com/spf13/cobra"
)

type grpcInboundOptions struct {
	listen            string
	upstream          string
	bootstrapPath     string
	runtimeConfig     string
	mtlsMode          string
	trustDomain       string
	allowedPrincipals string
	acceptTimeout     time.Duration
	connectTimeout    time.Duration
	reloadInterval    time.Duration
}

type grpcInboundMTLSMode string

const (
	grpcInboundMTLSModeDisable    grpcInboundMTLSMode = "DISABLE"
	grpcInboundMTLSModePermissive grpcInboundMTLSMode = "PERMISSIVE"
	grpcInboundMTLSModeStrict     grpcInboundMTLSMode = "STRICT"
)

// grpcInboundReloadInterval bounds how stale the workload certificate and the
// runtime config may be. Certificates are rotated well ahead of expiry by the
// control plane, so polling is enough and avoids the inotify blind spot around
// kubelet's atomic symlink swap of the mounted secret.
const grpcInboundReloadInterval = 30 * time.Second

// grpcInboundAcceptTimeout caps how long a peer may take to get through the
// first read and the TLS handshake. Every mesh pod can reach this port, so
// without a deadline a peer that connects and never writes pins a goroutine
// and a file descriptor indefinitely.
const grpcInboundAcceptTimeout = 10 * time.Second

func newGRPCInboundCommand() *cobra.Command {
	opts := &grpcInboundOptions{
		listen:            firstNonEmpty(os.Getenv("DUBBO_GRPC_INBOUND_LISTEN"), fmt.Sprintf(":%d", inject.ProxylessGRPCInboundPort)),
		upstream:          firstNonEmpty(os.Getenv("DUBBO_GRPC_INBOUND_UPSTREAM"), "127.0.0.1:80"),
		bootstrapPath:     os.Getenv("GRPC_XDS_BOOTSTRAP"),
		runtimeConfig:     firstNonEmpty(os.Getenv(inject.ProxylessGRPCConfigEnvName), inject.ProxylessGRPCConfigPath),
		mtlsMode:          os.Getenv("DUBBO_GRPC_INBOUND_MTLS_MODE"),
		trustDomain:       firstNonEmpty(os.Getenv("DUBBO_GRPC_INBOUND_TRUST_DOMAIN"), os.Getenv("TRUST_DOMAIN")),
		allowedPrincipals: os.Getenv("DUBBO_GRPC_INBOUND_ALLOWED_PRINCIPALS"),
		acceptTimeout:     durationSecondsFromEnv("DUBBO_GRPC_INBOUND_ACCEPT_TIMEOUT", grpcInboundAcceptTimeout),
		connectTimeout:    durationSecondsFromEnv("DUBBO_GRPC_INBOUND_CONNECT_TIMEOUT", 5*time.Second),
		reloadInterval:    durationSecondsFromEnv("DUBBO_GRPC_INBOUND_RELOAD_INTERVAL", grpcInboundReloadInterval),
	}
	c := &cobra.Command{
		Use:   "grpc-inbound",
		Short: "run an inbound mTLS data-plane proxy for proxyless workloads",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			log.SetDefaultScope(grpcInboundLogScope)
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return opts.run(cmd.Context())
		},
	}
	c.Flags().StringVar(&opts.listen, "listen", opts.listen, "mTLS listener address")
	c.Flags().StringVar(&opts.upstream, "upstream", opts.upstream, "local plaintext upstream address")
	c.Flags().StringVar(&opts.bootstrapPath, "bootstrap", opts.bootstrapPath, "gRPC xDS bootstrap file")
	c.Flags().StringVar(&opts.runtimeConfig, "runtime-config", opts.runtimeConfig, "proxyless runtime config file")
	c.Flags().StringVar(&opts.mtlsMode, "mtls-mode", opts.mtlsMode, "override inbound mTLS mode: DISABLE, PERMISSIVE, or STRICT")
	c.Flags().StringVar(&opts.trustDomain, "trust-domain", opts.trustDomain, "trust domain peers must belong to; defaults to the trust domain of the workload certificate")
	c.Flags().StringVar(&opts.allowedPrincipals, "allowed-principals", opts.allowedPrincipals,
		"comma-separated peer identities allowed to connect, as spiffe:// URIs or ns/<namespace>/sa/<serviceaccount>; empty allows any peer in the trust domain")
	c.Flags().DurationVar(&opts.acceptTimeout, "accept-timeout", opts.acceptTimeout, "deadline for the first read and the TLS handshake; 0 disables it")
	c.Flags().DurationVar(&opts.connectTimeout, "connect-timeout", opts.connectTimeout, "timeout for connecting to the local upstream")
	c.Flags().DurationVar(&opts.reloadInterval, "reload-interval", opts.reloadInterval, "how often to reload the workload certificate and runtime config")
	return c
}

func (o *grpcInboundOptions) run(ctx context.Context) error {
	if o.bootstrapPath == "" {
		return fmt.Errorf("grpc-inbound requires GRPC_XDS_BOOTSTRAP or --bootstrap")
	}
	if o.listen == "" {
		return fmt.Errorf("grpc-inbound listen address is required")
	}
	if o.upstream == "" {
		return fmt.Errorf("grpc-inbound upstream address is required")
	}
	bootstrap, err := xdsresolver.ParseBootstrap(o.bootstrapPath)
	if err != nil {
		return err
	}
	certs, err := newGRPCInboundCertStore(bootstrap)
	if err != nil {
		return err
	}
	modes := newGRPCInboundModeStore(o.runtimeConfig, upstreamPort(o.upstream))
	if err := modes.reload(); err != nil {
		// A missing or unreadable runtime config is not fatal: the store falls
		// back to STRICT until a successful load, and an explicit --mtls-mode
		// still overrides it.
		log.Warnf("grpc-inbound: initial runtime config load failed: %v", err)
	}
	peers, err := o.peerPolicy(certs)
	if err != nil {
		return err
	}
	lis, err := net.Listen("tcp", o.listen)
	if err != nil {
		return fmt.Errorf("listen grpc-inbound %s: %w", o.listen, err)
	}
	defer lis.Close()
	go grpcInboundReloadLoop(ctx, o.reloadInterval, certs, modes)
	return serveGRPCInbound(ctx, lis, certs.tlsConfig(peers), o.upstream, o.effectiveMTLSMode(modes), o.acceptTimeout, o.connectTimeout)
}

func (o *grpcInboundOptions) peerPolicy(certs *grpcInboundCertStore) (*grpcInboundPeerPolicy, error) {
	trustDomain := firstNonEmpty(strings.TrimSpace(o.trustDomain), certs.trustDomain())
	allowed, err := parseGRPCInboundPrincipals(o.allowedPrincipals, trustDomain)
	if err != nil {
		return nil, err
	}
	if trustDomain == "" {
		log.Warnf("grpc-inbound: no trust domain configured and the workload certificate carries no SPIFFE identity; peer identity is not checked")
	}
	return &grpcInboundPeerPolicy{trustDomain: trustDomain, allowed: allowed}, nil
}

// grpcInboundReloadLoop keeps the workload certificate and the runtime config
// in sync with the mounted secret. Without it the process would serve the
// key pair it loaded at startup until the pod is restarted, so inbound mTLS
// would break as soon as the control plane rotates the certificate.
func grpcInboundReloadLoop(ctx context.Context, interval time.Duration, certs *grpcInboundCertStore, modes *grpcInboundModeStore) {
	if interval <= 0 {
		interval = grpcInboundReloadInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := certs.reload(); err != nil {
				log.Warnf("grpc-inbound: certificate reload failed, keeping previous material: %v", err)
			}
			if err := modes.reload(); err != nil {
				log.Warnf("grpc-inbound: runtime config reload failed, keeping previous mode: %v", err)
			}
		}
	}
}

// grpcInboundCertStore holds the currently loaded workload key pair and trust
// bundle. Handshakes read through it, so a reload takes effect on the next
// connection without dropping established ones.
type grpcInboundCertStore struct {
	certFile string
	keyFile  string
	caFile   string

	mu        sync.RWMutex
	cert      *tls.Certificate
	clientCAs *x509.CertPool
}

func newGRPCInboundCertStore(bootstrap *xdsresolver.BootstrapConfig) (*grpcInboundCertStore, error) {
	if bootstrap == nil {
		return nil, fmt.Errorf("bootstrap config is nil")
	}
	cfg, ok := bootstrap.CertProviders["default"]
	if !ok {
		return nil, fmt.Errorf("certificate_providers[default] not found")
	}
	if cfg.CertificateFile == "" || cfg.PrivateKeyFile == "" {
		return nil, fmt.Errorf("grpc-inbound mTLS requires certificate_file and private_key_file")
	}
	if cfg.CACertificateFile == "" {
		return nil, fmt.Errorf("grpc-inbound mTLS requires ca_certificate_file")
	}
	store := &grpcInboundCertStore{
		certFile: cfg.CertificateFile,
		keyFile:  cfg.PrivateKeyFile,
		caFile:   cfg.CACertificateFile,
	}
	if err := store.reload(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *grpcInboundCertStore) reload() error {
	cert, err := tls.LoadX509KeyPair(s.certFile, s.keyFile)
	if err != nil {
		return fmt.Errorf("load grpc-inbound certificate/key: %w", err)
	}
	// Leaf is needed to read the workload's own SPIFFE identity; older Go
	// releases leave it nil after LoadX509KeyPair.
	if cert.Leaf == nil && len(cert.Certificate) > 0 {
		leaf, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return fmt.Errorf("parse grpc-inbound certificate %s: %w", s.certFile, err)
		}
		cert.Leaf = leaf
	}
	rootPEM, err := os.ReadFile(s.caFile)
	if err != nil {
		return fmt.Errorf("read grpc-inbound CA certificate %s: %w", s.caFile, err)
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(rootPEM) {
		return fmt.Errorf("parse grpc-inbound CA certificate %s: no certificates found", s.caFile)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cert = &cert
	s.clientCAs = clientCAs
	return nil
}

func (s *grpcInboundCertStore) current() (*tls.Certificate, *x509.CertPool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cert, s.clientCAs
}

// tlsConfig returns a config whose material is resolved per handshake.
// GetConfigForClient is used rather than GetCertificate because the trust
// bundle rotates alongside the key pair and ClientCAs cannot be swapped from
// the certificate callback.
func (s *grpcInboundCertStore) tlsConfig(peers *grpcInboundPeerPolicy) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ClientAuth: tls.RequireAndVerifyClientCert,
		GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
			cert, clientCAs := s.current()
			if cert == nil || clientCAs == nil {
				return nil, fmt.Errorf("grpc-inbound certificate material is not loaded")
			}
			return &tls.Config{
				MinVersion:            tls.VersionTLS12,
				Certificates:          []tls.Certificate{*cert},
				ClientCAs:             clientCAs,
				ClientAuth:            tls.RequireAndVerifyClientCert,
				VerifyPeerCertificate: peers.verifyPeerCertificate,
			}, nil
		},
	}
}

// trustDomain reports the trust domain of the workload's own SPIFFE identity,
// used as the default peer trust domain when none is configured explicitly.
func (s *grpcInboundCertStore) trustDomain() string {
	cert, _ := s.current()
	if cert == nil || cert.Leaf == nil {
		return ""
	}
	for _, id := range spiffeIdentities(cert.Leaf) {
		return id.Host
	}
	return ""
}

// grpcInboundPeerPolicy authorizes an authenticated peer. Chain verification
// alone only proves the peer holds a certificate signed by the mesh CA, which
// makes every workload in the mesh a valid caller for every other workload.
// This narrows that to a trust domain and, when configured, to an explicit set
// of SPIFFE identities.
type grpcInboundPeerPolicy struct {
	trustDomain string
	allowed     map[string]struct{}
}

// verifyPeerCertificate runs after chain verification, so verifiedChains is
// non-empty and its leaf is already trusted. It only decides whether that
// proven identity may talk to this workload.
func (p *grpcInboundPeerPolicy) verifyPeerCertificate(_ [][]byte, verifiedChains [][]*x509.Certificate) error {
	if p == nil || (p.trustDomain == "" && len(p.allowed) == 0) {
		return nil
	}
	if len(verifiedChains) == 0 || len(verifiedChains[0]) == 0 {
		return fmt.Errorf("grpc-inbound: peer presented no verified certificate chain")
	}
	identities := spiffeIdentities(verifiedChains[0][0])
	if len(identities) == 0 {
		return fmt.Errorf("grpc-inbound: peer certificate carries no SPIFFE identity")
	}
	for _, id := range identities {
		if p.trustDomain != "" && id.Host != p.trustDomain {
			continue
		}
		if len(p.allowed) == 0 {
			return nil
		}
		if _, ok := p.allowed[id.String()]; ok {
			return nil
		}
	}
	return fmt.Errorf("grpc-inbound: peer identity %s is not authorized", identities[0])
}

// spiffeIdentities returns the SPIFFE URI SANs of a certificate. A workload
// certificate normally carries exactly one.
func spiffeIdentities(cert *x509.Certificate) []*neturl.URL {
	if cert == nil {
		return nil
	}
	out := make([]*neturl.URL, 0, len(cert.URIs))
	for _, uri := range cert.URIs {
		if uri != nil && uri.Scheme == "spiffe" {
			out = append(out, uri)
		}
	}
	return out
}

// parseGRPCInboundPrincipals accepts full spiffe:// URIs or the
// ns/<namespace>/sa/<serviceaccount> shorthand, which is expanded against the
// trust domain.
func parseGRPCInboundPrincipals(list, trustDomain string) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	for _, raw := range strings.Split(list, ",") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if !strings.HasPrefix(entry, "spiffe://") {
			if trustDomain == "" {
				return nil, fmt.Errorf("principal %q needs a trust domain: set --trust-domain or use a full spiffe:// URI", entry)
			}
			entry = "spiffe://" + trustDomain + "/" + strings.TrimPrefix(entry, "/")
		}
		parsed, err := neturl.Parse(entry)
		if err != nil {
			return nil, fmt.Errorf("parse principal %q: %w", raw, err)
		}
		out[parsed.String()] = struct{}{}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func serveGRPCInbound(ctx context.Context, lis net.Listener, tlsConfig *tls.Config, upstream string, mode func() grpcInboundMTLSMode, acceptTimeout, connectTimeout time.Duration) error {
	go func() {
		<-ctx.Done()
		_ = lis.Close()
	}()
	for {
		conn, err := lis.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}
		go proxyGRPCInboundConnection(conn, tlsConfig, upstream, mode(), acceptTimeout, connectTimeout)
	}
}

func proxyGRPCInboundConnection(inbound net.Conn, tlsConfig *tls.Config, upstream string, mode grpcInboundMTLSMode, acceptTimeout, connectTimeout time.Duration) {
	defer inbound.Close()
	if acceptTimeout > 0 {
		_ = inbound.SetDeadline(time.Now().Add(acceptTimeout))
	}

	reader := bufio.NewReader(inbound)
	first, err := reader.Peek(1)
	if err != nil {
		return
	}
	buffered := &bufferedConn{Conn: inbound, reader: reader}
	if isTLSClientHello(first[0]) {
		if mode == grpcInboundMTLSModeDisable {
			return
		}
		tlsConn := tls.Server(buffered, tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			return
		}
		inbound = tlsConn
	} else {
		if mode == grpcInboundMTLSModeStrict {
			return
		}
		inbound = buffered
	}

	_ = inbound.SetDeadline(time.Time{})
	dialer := net.Dialer{Timeout: connectTimeout}
	outbound, err := dialer.Dial("tcp", upstream)
	if err != nil {
		return
	}
	defer outbound.Close()
	copyBothDirections(inbound, outbound)
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func isTLSClientHello(first byte) bool {
	return first == 0x16
}

func (o *grpcInboundOptions) effectiveMTLSMode(modes *grpcInboundModeStore) func() grpcInboundMTLSMode {
	if mode, ok := parseGRPCInboundMTLSMode(o.mtlsMode); ok {
		return func() grpcInboundMTLSMode { return mode }
	}
	return modes.current
}

// grpcInboundModeStore caches the inbound mTLS mode read from the runtime
// config. Reading it per connection would put a file read and a full JSON
// parse on the accept path, and would let a transient read error silently
// downgrade a STRICT port to PERMISSIVE.
type grpcInboundModeStore struct {
	path string
	port int

	mu     sync.RWMutex
	mode   grpcInboundMTLSMode
	loaded bool
}

func newGRPCInboundModeStore(path string, port int) *grpcInboundModeStore {
	return &grpcInboundModeStore{path: path, port: port}
}

// current fails closed: until the runtime config has been read successfully
// at least once, inbound traffic must present a client certificate.
func (s *grpcInboundModeStore) current() grpcInboundMTLSMode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.loaded {
		return grpcInboundMTLSModeStrict
	}
	return s.mode
}

// reload replaces the cached mode only on a successful read and parse. Any
// failure leaves the last known good mode in place, so a remount race or a
// truncated write cannot relax the policy.
func (s *grpcInboundModeStore) reload() error {
	mode, err := loadGRPCInboundMTLSMode(s.path, s.port)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = mode
	s.loaded = true
	return nil
}

func parseGRPCInboundMTLSMode(mode string) (grpcInboundMTLSMode, bool) {
	switch strings.ToUpper(strings.TrimSpace(mode)) {
	case string(grpcInboundMTLSModeDisable):
		return grpcInboundMTLSModeDisable, true
	case string(grpcInboundMTLSModePermissive):
		return grpcInboundMTLSModePermissive, true
	case string(grpcInboundMTLSModeStrict):
		return grpcInboundMTLSModeStrict, true
	default:
		return "", false
	}
}

func upstreamPort(upstream string) int {
	_, port, err := net.SplitHostPort(upstream)
	if err != nil {
		return 0
	}
	out, err := strconv.Atoi(port)
	if err != nil {
		return 0
	}
	return out
}

// loadGRPCInboundMTLSMode reads the inbound mTLS mode for port from the runtime
// config. An absent config means the workload is unconfigured and yields
// PERMISSIVE, matching a standalone run with no mounted secret. A config that
// exists but cannot be read or parsed returns an error so the caller can keep
// the last known mode instead of relaxing the policy.
func loadGRPCInboundMTLSMode(path string, port int) (grpcInboundMTLSMode, error) {
	if path == "" {
		return grpcInboundMTLSModePermissive, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return grpcInboundMTLSModePermissive, nil
		}
		return "", fmt.Errorf("read runtime config %s: %w", path, err)
	}
	var cfg struct {
		Services []struct {
			Ports []struct {
				Port     int    `json:"port"`
				MTLSMode string `json:"mtlsMode"`
			} `json:"ports"`
		} `json:"services"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parse runtime config %s: %w", path, err)
	}

	foundDisable := false
	foundPermissive := false
	for _, svc := range cfg.Services {
		for _, svcPort := range svc.Ports {
			if port != 0 && svcPort.Port != port {
				continue
			}
			mode, ok := parseGRPCInboundMTLSMode(svcPort.MTLSMode)
			if !ok {
				continue
			}
			if mode == grpcInboundMTLSModeStrict {
				return grpcInboundMTLSModeStrict, nil
			}
			foundPermissive = foundPermissive || mode == grpcInboundMTLSModePermissive
			foundDisable = foundDisable || mode == grpcInboundMTLSModeDisable
		}
	}
	if foundPermissive {
		return grpcInboundMTLSModePermissive, nil
	}
	if foundDisable {
		return grpcInboundMTLSModeDisable, nil
	}
	return grpcInboundMTLSModePermissive, nil
}

func copyBothDirections(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(a, b)
		closeWrite(a)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(b, a)
		closeWrite(b)
	}()
	wg.Wait()
}

func closeWrite(conn net.Conn) {
	if c, ok := conn.(interface{ CloseWrite() error }); ok {
		_ = c.CloseWrite()
		return
	}
	_ = conn.Close()
}
