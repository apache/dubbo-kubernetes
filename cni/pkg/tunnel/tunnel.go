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

package tunnel

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	cniconstants "github.com/apache/dubbo-kubernetes/cni/pkg/constants"
	"github.com/apache/dubbo-kubernetes/cni/pkg/xdsclient"
	"github.com/apache/dubbo-kubernetes/pkg/spiffe"
	"golang.org/x/net/http2"
)

// Workload represents a local enrolled Pod managed by the node-level Triple tunnel proxy.
type Workload struct {
	PodName   string
	Namespace string
	PodIP     string
	Certs     xdsclient.CertMaterial
	XDS       *xdsclient.Client
}

// Config configures the node-level Triple mTLS tunnel and transparent outbound proxy.
type Config struct {
	OutboundAddr        string
	InboundTunnelAddr   string
	PeerTunnelPort      int
	DestinationResolver func(net.Conn) (string, error)
	DialTimeout         time.Duration
}

// Metrics tracks live node tunnel connection counts.
type Metrics struct {
	OutboundTotal uint64 `json:"outbound_total"`
	InboundTotal  uint64 `json:"inbound_total"`
	TunnelTotal   uint64 `json:"tunnel_total"`
	Active        int64  `json:"active"`
}

// Proxy runs the node-level outbound capture listener (:15001) and inbound mTLS
// HTTP/2 CONNECT tunnel server (:15008) for all enrolled Pods on the node.
type Proxy struct {
	cfg Config

	mu        sync.RWMutex
	workloads map[string]*Workload // keyed by PodIP

	outboundLn net.Listener
	inboundLn  net.Listener
	inboundSrv *http.Server

	ticket        atomic.Uint64
	outboundTotal atomic.Uint64
	inboundTotal  atomic.Uint64
	tunnelTotal   atomic.Uint64
	activeConns   atomic.Int64
}

// NewProxy creates a node-level Triple tunnel Proxy.
func NewProxy(cfg Config) *Proxy {
	if cfg.OutboundAddr == "" {
		cfg.OutboundAddr = fmt.Sprintf(":%d", cniconstants.OutboundCapturePort)
	}
	if cfg.InboundTunnelAddr == "" {
		cfg.InboundTunnelAddr = fmt.Sprintf(":%d", cniconstants.InboundTunnelPort)
	}
	if cfg.PeerTunnelPort <= 0 {
		cfg.PeerTunnelPort = cniconstants.InboundTunnelPort
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.DestinationResolver == nil {
		cfg.DestinationResolver = OriginalDestination
	}
	return &Proxy{
		cfg:       cfg,
		workloads: map[string]*Workload{},
	}
}

// RegisterWorkload registers or updates an enrolled local Pod by its PodIP.
func (p *Proxy) RegisterWorkload(w *Workload) {
	if w == nil || w.PodIP == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.workloads[w.PodIP] = w
}

// UnregisterWorkload removes a Pod from the node tunnel proxy.
func (p *Proxy) UnregisterWorkload(podIP string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.workloads, podIP)
}

// WorkloadByIP returns the registered local Workload for a PodIP.
func (p *Proxy) WorkloadByIP(podIP string) *Workload {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.workloads[podIP]
}

// Metrics returns a point-in-time snapshot of proxy connection counters.
func (p *Proxy) Metrics() Metrics {
	return Metrics{
		OutboundTotal: p.outboundTotal.Load(),
		InboundTotal:  p.inboundTotal.Load(),
		TunnelTotal:   p.tunnelTotal.Load(),
		Active:        p.activeConns.Load(),
	}
}

// OutboundAddr returns the bound outbound capture listener address.
func (p *Proxy) OutboundAddr() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.outboundLn == nil {
		return p.cfg.OutboundAddr
	}
	return p.outboundLn.Addr().String()
}

// InboundTunnelAddr returns the bound inbound mTLS tunnel listener address.
func (p *Proxy) InboundTunnelAddr() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.inboundLn == nil {
		return p.cfg.InboundTunnelAddr
	}
	return p.inboundLn.Addr().String()
}

// Start binds both the outbound capture listener and inbound mTLS HTTP/2 CONNECT tunnel server.
func (p *Proxy) Start(ctx context.Context) error {
	outLn, err := net.Listen("tcp", p.cfg.OutboundAddr)
	if err != nil {
		return fmt.Errorf("listen outbound capture %s: %w", p.cfg.OutboundAddr, err)
	}
	inRawLn, err := net.Listen("tcp", p.cfg.InboundTunnelAddr)
	if err != nil {
		_ = outLn.Close()
		return fmt.Errorf("listen inbound tunnel %s: %w", p.cfg.InboundTunnelAddr, err)
	}

	tlsCfg := &tls.Config{
		MinVersion:     tls.VersionTLS12,
		NextProtos:     []string{cniconstants.TunnelALPN},
		ClientAuth:     tls.RequireAndVerifyClientCert,
		GetConfigForClient: p.inboundTLSConfigForClient,
	}
	inLn := tls.NewListener(inRawLn, tlsCfg)

	srv := &http.Server{
		Handler:           http.HandlerFunc(p.handleInboundTunnel),
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := http2.ConfigureServer(srv, &http2.Server{}); err != nil {
		_ = outLn.Close()
		_ = inLn.Close()
		return fmt.Errorf("configure HTTP/2 tunnel server: %w", err)
	}

	p.mu.Lock()
	p.outboundLn = outLn
	p.inboundLn = inLn
	p.inboundSrv = srv
	p.mu.Unlock()

	go p.serveOutbound(ctx, outLn)
	go func() {
		_ = srv.Serve(inLn)
	}()
	go func() {
		<-ctx.Done()
		_ = p.Close()
	}()
	return nil
}

// Close shuts down both listeners gracefully.
func (p *Proxy) Close() error {
	p.mu.Lock()
	outLn := p.outboundLn
	inLn := p.inboundLn
	srv := p.inboundSrv
	p.mu.Unlock()

	var errs []error
	if outLn != nil {
		errs = append(errs, outLn.Close())
	}
	if srv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		errs = append(errs, srv.Shutdown(shutdownCtx))
		cancel()
	} else if inLn != nil {
		errs = append(errs, inLn.Close())
	}
	return errors.Join(errs...)
}

func (p *Proxy) serveOutbound(ctx context.Context, ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
		go p.handleOutbound(ctx, conn)
	}
}

func (p *Proxy) handleOutbound(ctx context.Context, clientConn net.Conn) {
	defer clientConn.Close()
	p.outboundTotal.Add(1)
	p.activeConns.Add(1)
	defer p.activeConns.Add(-1)

	origDst, err := p.cfg.DestinationResolver(clientConn)
	if err != nil || origDst == "" {
		return
	}

	srcIP, _, _ := net.SplitHostPort(clientConn.RemoteAddr().String())
	srcWorkload := p.lookupWorkloadForSource(srcIP)

	targetAddr := origDst
	useTunnel := false
	dialTimeout := p.cfg.DialTimeout

	if srcWorkload != nil && srcWorkload.XDS != nil {
		ticket := p.ticket.Add(1)
		if sel, err := srcWorkload.XDS.Snapshot().SelectByDestinationIPPort(origDst, ticket); err == nil {
			targetAddr = sel.Endpoint.HostPort()
			useTunnel = sel.UseMTLS
			if sel.ConnectTimeout > 0 {
				dialTimeout = sel.ConnectTimeout
			}
		}
	}

	if useTunnel && srcWorkload != nil {
		p.tunnelTotal.Add(1)
		if err := p.forwardViaTunnel(ctx, clientConn, srcWorkload, targetAddr, dialTimeout); err != nil {
			return
		}
		return
	}

	dialer := &net.Dialer{Timeout: dialTimeout}
	upstream, err := dialer.DialContext(ctx, "tcp", targetAddr)
	if err != nil {
		return
	}
	defer upstream.Close()
	pipeConns(clientConn, upstream)
}

func (p *Proxy) forwardViaTunnel(ctx context.Context, clientConn net.Conn, src *Workload, targetAddr string, timeout time.Duration) error {
	targetHost, _, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return err
	}
	tunnelAddr := net.JoinHostPort(targetHost, strconv.Itoa(p.cfg.PeerTunnelPort))

	rootPool, err := src.Certs.RootCertPool()
	if err != nil {
		return err
	}
	cert, err := src.Certs.LoadKeyPair()
	if err != nil {
		return err
	}
	tlsCfg := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            rootPool,
		NextProtos:         []string{cniconstants.TunnelALPN},
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, // Peer certificate chain + SPIFFE URI SAN verified below.
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			return verifyMeshPeerCert(rawCerts, rootPool)
		},
	}

	tr := &http2.Transport{
		DialTLSContext: func(dialCtx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			d := &tls.Dialer{
				NetDialer: &net.Dialer{Timeout: timeout},
				Config:    tlsCfg,
			}
			return d.DialContext(dialCtx, network, addr)
		},
	}
	defer tr.CloseIdleConnections()

	pr, pw := io.Pipe()
	// Standard HTTP/2 CONNECT uses target host:port as authority.
	parsedReq, err := http.NewRequestWithContext(ctx, http.MethodConnect, "https://"+tunnelAddr, pr)
	if err != nil {
		_ = pr.Close()
		_ = pw.Close()
		return err
	}
	parsedReq.Host = targetAddr
	parsedReq.Header.Set("X-Dubbo-Source-Namespace", src.Namespace)
	parsedReq.Header.Set("X-Dubbo-Source-Pod", src.PodName)

	resp, err := tr.RoundTrip(parsedReq)
	if err != nil {
		_ = pr.Close()
		_ = pw.Close()
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_ = pw.Close()
		return fmt.Errorf("tunnel CONNECT %s failed with status %d", targetAddr, resp.StatusCode)
	}

	errCh := make(chan error, 2)
	go func() {
		_, copyErr := io.Copy(pw, clientConn)
		_ = pw.Close()
		errCh <- copyErr
	}()
	go func() {
		_, copyErr := io.Copy(clientConn, resp.Body)
		errCh <- copyErr
	}()
	<-errCh
	return nil
}

func (p *Proxy) handleInboundTunnel(w http.ResponseWriter, r *http.Request) {
	p.inboundTotal.Add(1)
	p.activeConns.Add(1)
	defer p.activeConns.Add(-1)

	if r.Method != http.MethodConnect {
		http.Error(w, "only CONNECT is supported on Triple tunnel port", http.StatusMethodNotAllowed)
		return
	}
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "mTLS client certificate required", http.StatusUnauthorized)
		return
	}
	if !hasValidSPIFFEURI(r.TLS.PeerCertificates[0]) {
		http.Error(w, "invalid SPIFFE peer identity", http.StatusForbidden)
		return
	}

	targetAddr := r.Host
	if _, _, err := net.SplitHostPort(targetAddr); err != nil {
		http.Error(w, "invalid CONNECT target host:port", http.StatusBadRequest)
		return
	}

	dialer := &net.Dialer{Timeout: p.cfg.DialTimeout}
	localConn, err := dialer.DialContext(r.Context(), "tcp", targetAddr)
	if err != nil {
		http.Error(w, fmt.Sprintf("dial local target %s: %v", targetAddr, err), http.StatusBadGateway)
		return
	}
	defer localConn.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(localConn, r.Body)
		if tc, ok := localConn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
		done <- struct{}{}
	}()
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, readErr := localConn.Read(buf)
			if n > 0 {
				if _, writeErr := w.Write(buf[:n]); writeErr != nil {
					break
				}
				flusher.Flush()
			}
			if readErr != nil {
				break
			}
		}
		done <- struct{}{}
	}()
	<-done
	<-done
}

func (p *Proxy) inboundTLSConfigForClient(_ *tls.ClientHelloInfo) (*tls.Config, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, w := range p.workloads {
		cert, err := w.Certs.LoadKeyPair()
		if err != nil {
			continue
		}
		roots, err := w.Certs.RootCertPool()
		if err != nil {
			continue
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientCAs:    roots,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			NextProtos:   []string{cniconstants.TunnelALPN},
			MinVersion:   tls.VersionTLS12,
		}, nil
	}
	return nil, fmt.Errorf("no enrolled workload TLS credentials available on node")
}

func (p *Proxy) lookupWorkloadForSource(srcIP string) *Workload {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if w, ok := p.workloads[srcIP]; ok {
		return w
	}
	// When running local loopback tests, fall back to the single registered workload if unique.
	if (srcIP == "127.0.0.1" || srcIP == "::1") && len(p.workloads) == 1 {
		for _, w := range p.workloads {
			return w
		}
	}
	return nil
}

func verifyMeshPeerCert(rawCerts [][]byte, roots *x509.CertPool) error {
	if len(rawCerts) == 0 {
		return fmt.Errorf("missing peer certificate")
	}
	certs := make([]*x509.Certificate, 0, len(rawCerts))
	for _, raw := range rawCerts {
		c, err := x509.ParseCertificate(raw)
		if err != nil {
			return err
		}
		certs = append(certs, c)
	}
	intermediates := x509.NewCertPool()
	for _, c := range certs[1:] {
		intermediates.AddCert(c)
	}
	if _, err := certs[0].Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err != nil {
		return fmt.Errorf("verify peer cert chain: %w", err)
	}
	if !hasValidSPIFFEURI(certs[0]) {
		return fmt.Errorf("peer certificate missing valid SPIFFE URI SAN")
	}
	return nil
}

func hasValidSPIFFEURI(cert *x509.Certificate) bool {
	for _, u := range cert.URIs {
		if id, err := spiffe.ParseIdentity(u.String()); err == nil &&
			id.TrustDomain != "" && id.Namespace != "" && id.ServiceAccount != "" {
			return true
		}
	}
	return false
}

func pipeConns(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	copySide := func(dst, src net.Conn) {
		defer wg.Done()
		_, _ = io.Copy(dst, src)
		if tc, ok := dst.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}
	go copySide(a, b)
	go copySide(b, a)
	wg.Wait()
}
