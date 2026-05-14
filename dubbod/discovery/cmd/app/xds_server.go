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
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"os"
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
	acceptTimeout  time.Duration
	connectTimeout time.Duration
}

func newXServerCommand() *cobra.Command {
	opts := &xdsServerOptions{
		listen:         firstNonEmpty(os.Getenv("DUBBO_XSERVER_LISTEN"), fmt.Sprintf(":%d", inject.ProxylessXServerPort)),
		upstream:       firstNonEmpty(os.Getenv("DUBBO_XSERVER_UPSTREAM"), "127.0.0.1:80"),
		bootstrapPath:  os.Getenv("GRPC_XDS_BOOTSTRAP"),
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
	return serveXServer(ctx, lis, tlsConfig, o.upstream, o.acceptTimeout, o.connectTimeout)
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

func serveXServer(ctx context.Context, lis net.Listener, tlsConfig *tls.Config, upstream string, acceptTimeout, connectTimeout time.Duration) error {
	tlsListener := tls.NewListener(lis, tlsConfig)
	go func() {
		<-ctx.Done()
		_ = tlsListener.Close()
	}()
	for {
		conn, err := tlsListener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}
		go proxyXServerConnection(conn, upstream, acceptTimeout, connectTimeout)
	}
}

func proxyXServerConnection(inbound net.Conn, upstream string, acceptTimeout, connectTimeout time.Duration) {
	defer inbound.Close()
	if acceptTimeout > 0 {
		_ = inbound.SetDeadline(time.Now().Add(acceptTimeout))
	}
	if tlsConn, ok := inbound.(*tls.Conn); ok {
		if err := tlsConn.Handshake(); err != nil {
			return
		}
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
