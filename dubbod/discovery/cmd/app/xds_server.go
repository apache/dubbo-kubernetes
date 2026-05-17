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

type xdsServerOptions struct {
	listen         string
	upstream       string
	bootstrapPath  string
	runtimeConfig  string
	mtlsMode       string
	acceptTimeout  time.Duration
	connectTimeout time.Duration
}

type xserverMTLSMode string

const (
	xserverMTLSModeDisable    xserverMTLSMode = "DISABLE"
	xserverMTLSModePermissive xserverMTLSMode = "PERMISSIVE"
	xserverMTLSModeStrict     xserverMTLSMode = "STRICT"
)

func newXServerCommand() *cobra.Command {
	opts := &xdsServerOptions{
		listen:         firstNonEmpty(os.Getenv("DUBBO_XSERVER_LISTEN"), fmt.Sprintf(":%d", inject.ProxylessXServerPort)),
		upstream:       firstNonEmpty(os.Getenv("DUBBO_XSERVER_UPSTREAM"), "127.0.0.1:80"),
		bootstrapPath:  os.Getenv("GRPC_XDS_BOOTSTRAP"),
		runtimeConfig:  firstNonEmpty(os.Getenv(inject.ProxylessGRPCConfigEnvName), inject.ProxylessGRPCConfigPath),
		mtlsMode:       os.Getenv("DUBBO_XSERVER_MTLS_MODE"),
		acceptTimeout:  durationSecondsFromEnv("DUBBO_XSERVER_ACCEPT_TIMEOUT", 0),
		connectTimeout: durationSecondsFromEnv("DUBBO_XSERVER_CONNECT_TIMEOUT", 5*time.Second),
	}
	c := &cobra.Command{
		Use:   "xserver",
		Short: "run an inbound mTLS data-plane proxy for proxyless workloads",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			log.SetDefaultScope(xserverLogScope)
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
	c.Flags().DurationVar(&opts.acceptTimeout, "accept-timeout", opts.acceptTimeout, "optional TLS handshake timeout")
	c.Flags().DurationVar(&opts.connectTimeout, "connect-timeout", opts.connectTimeout, "timeout for connecting to the local upstream")
	return c
}

func (o *xdsServerOptions) run(ctx context.Context) error {
	if o.bootstrapPath == "" {
		return fmt.Errorf("xserver requires GRPC_XDS_BOOTSTRAP or --bootstrap")
	}
	if o.listen == "" {
		return fmt.Errorf("xserver listen address is required")
	}
	if o.upstream == "" {
		return fmt.Errorf("xserver upstream address is required")
	}
	bootstrap, err := xdsresolver.ParseBootstrap(o.bootstrapPath)
	if err != nil {
		return err
	}
	tlsConfig, err := xserverTLSConfigFromBootstrap(bootstrap)
	if err != nil {
		return err
	}
	lis, err := net.Listen("tcp", o.listen)
	if err != nil {
		return fmt.Errorf("listen xserver %s: %w", o.listen, err)
	}
	defer lis.Close()
	return serveXServer(ctx, lis, tlsConfig, o.upstream, o.effectiveMTLSMode, o.acceptTimeout, o.connectTimeout)
}

func xserverTLSConfigFromBootstrap(bootstrap *xdsresolver.BootstrapConfig) (*tls.Config, error) {
	if bootstrap == nil {
		return nil, fmt.Errorf("bootstrap config is nil")
	}
	cfg, ok := bootstrap.CertProviders["default"]
	if !ok {
		return nil, fmt.Errorf("certificate_providers[default] not found")
	}
	if cfg.CertificateFile == "" || cfg.PrivateKeyFile == "" {
		return nil, fmt.Errorf("xserver mTLS requires certificate_file and private_key_file")
	}
	if cfg.CACertificateFile == "" {
		return nil, fmt.Errorf("xserver mTLS requires ca_certificate_file")
	}
	cert, err := tls.LoadX509KeyPair(cfg.CertificateFile, cfg.PrivateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load xserver certificate/key: %w", err)
	}
	rootPEM, err := os.ReadFile(cfg.CACertificateFile)
	if err != nil {
		return nil, fmt.Errorf("read xserver CA certificate %s: %w", cfg.CACertificateFile, err)
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(rootPEM) {
		return nil, fmt.Errorf("parse xserver CA certificate %s: no certificates found", cfg.CACertificateFile)
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    clientCAs,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}, nil
}

func serveXServer(ctx context.Context, lis net.Listener, tlsConfig *tls.Config, upstream string, mode func() xserverMTLSMode, acceptTimeout, connectTimeout time.Duration) error {
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
		go proxyXServerConnection(conn, tlsConfig, upstream, mode(), acceptTimeout, connectTimeout)
	}
}

func proxyXServerConnection(inbound net.Conn, tlsConfig *tls.Config, upstream string, mode xserverMTLSMode, acceptTimeout, connectTimeout time.Duration) {
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
		if mode == xserverMTLSModeDisable {
			return
		}
		tlsConn := tls.Server(buffered, tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			return
		}
		inbound = tlsConn
	} else {
		if mode == xserverMTLSModeStrict {
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

func (o *xdsServerOptions) effectiveMTLSMode() xserverMTLSMode {
	if mode, ok := parseXServerMTLSMode(o.mtlsMode); ok {
		return mode
	}
	if mode, ok := xserverMTLSModeFromRuntimeConfig(o.runtimeConfig, upstreamPort(o.upstream)); ok {
		return mode
	}
	return xserverMTLSModePermissive
}

func parseXServerMTLSMode(mode string) (xserverMTLSMode, bool) {
	switch strings.ToUpper(strings.TrimSpace(mode)) {
	case string(xserverMTLSModeDisable):
		return xserverMTLSModeDisable, true
	case string(xserverMTLSModePermissive):
		return xserverMTLSModePermissive, true
	case string(xserverMTLSModeStrict):
		return xserverMTLSModeStrict, true
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

func xserverMTLSModeFromRuntimeConfig(path string, port int) (xserverMTLSMode, bool) {
	if path == "" {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
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
		return "", false
	}

	foundDisable := false
	foundPermissive := false
	for _, svc := range cfg.Services {
		for _, svcPort := range svc.Ports {
			if port != 0 && svcPort.Port != port {
				continue
			}
			mode, ok := parseXServerMTLSMode(svcPort.MTLSMode)
			if !ok {
				continue
			}
			if mode == xserverMTLSModeStrict {
				return xserverMTLSModeStrict, true
			}
			foundPermissive = foundPermissive || mode == xserverMTLSModePermissive
			foundDisable = foundDisable || mode == xserverMTLSModeDisable
		}
	}
	if foundPermissive {
		return xserverMTLSModePermissive, true
	}
	if foundDisable {
		return xserverMTLSModeDisable, true
	}
	return "", false
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
