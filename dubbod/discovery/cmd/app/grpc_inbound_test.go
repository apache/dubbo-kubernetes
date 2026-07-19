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
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	xdsresolver "github.com/kdubbo/xds-api/grpc/resolver"
)

func TestGRPCInboundRequiresClientCertificateAndProxiesHTTP(t *testing.T) {
	caCert, caKey := newTestCA(t)
	serverCert, serverKey := newSignedCert(t, caCert, caKey, "grpc-inbound")
	clientCert, clientKey := newSignedCert(t, caCert, caKey, "grpc-outbound")
	dir := t.TempDir()
	writePEM(t, filepath.Join(dir, "root-cert.pem"), "CERTIFICATE", caCert.Raw)
	writePEM(t, filepath.Join(dir, "cert-chain.pem"), "CERTIFICATE", serverCert.Raw)
	writePEM(t, filepath.Join(dir, "key.pem"), "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(serverKey))
	writePEM(t, filepath.Join(dir, "client-cert.pem"), "CERTIFICATE", clientCert.Raw)
	writePEM(t, filepath.Join(dir, "client-key.pem"), "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(clientKey))

	tlsConfig, err := grpcInboundTLSConfigFromBootstrap(&xdsresolver.BootstrapConfig{
		CertProviders: map[string]xdsresolver.FileWatcherCertConfig{
			"default": {
				CertificateFile:   filepath.Join(dir, "cert-chain.pem"),
				PrivateKeyFile:    filepath.Join(dir, "key.pem"),
				CACertificateFile: filepath.Join(dir, "root-cert.pem"),
			},
		},
	})
	if err != nil {
		t.Fatalf("grpcInboundTLSConfigFromBootstrap() failed: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintln(w, "nginx v1")
	}))
	defer upstream.Close()
	upstreamAddr := upstream.Listener.Addr().String()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() failed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- serveGRPCInbound(ctx, listener, tlsConfig, upstreamAddr, func() grpcInboundMTLSMode {
			return grpcInboundMTLSModeStrict
		}, time.Second, time.Second)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("serveGRPCInbound() error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("serveGRPCInbound() did not stop")
		}
	})

	url := "https://" + listener.Addr().String() + "/"
	_, err = (&http.Client{Timeout: time.Second}).Get("http://" + listener.Addr().String() + "/")
	if err == nil {
		t.Fatalf("plaintext request succeeded in STRICT mode, want failure")
	}

	_, err = (&http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true,
	}}}).Get(url)
	if err == nil {
		t.Fatalf("plain TLS request without client certificate succeeded, want failure")
	}

	cert, err := tls.LoadX509KeyPair(filepath.Join(dir, "client-cert.pem"), filepath.Join(dir, "client-key.pem"))
	if err != nil {
		t.Fatalf("tls.LoadX509KeyPair() failed: %v", err)
	}
	resp, err := (&http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}}}).Get(url)
	if err != nil {
		t.Fatalf("mTLS request failed: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() failed: %v", err)
	}
	if got, want := string(body), "nginx v1\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestGRPCInboundPermissiveAcceptsPlaintextAndMTLS(t *testing.T) {
	caCert, caKey := newTestCA(t)
	serverCert, serverKey := newSignedCert(t, caCert, caKey, "grpc-inbound")
	clientCert, clientKey := newSignedCert(t, caCert, caKey, "grpc-outbound")
	dir := t.TempDir()
	writePEM(t, filepath.Join(dir, "root-cert.pem"), "CERTIFICATE", caCert.Raw)
	writePEM(t, filepath.Join(dir, "cert-chain.pem"), "CERTIFICATE", serverCert.Raw)
	writePEM(t, filepath.Join(dir, "key.pem"), "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(serverKey))
	writePEM(t, filepath.Join(dir, "client-cert.pem"), "CERTIFICATE", clientCert.Raw)
	writePEM(t, filepath.Join(dir, "client-key.pem"), "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(clientKey))

	tlsConfig, err := grpcInboundTLSConfigFromBootstrap(&xdsresolver.BootstrapConfig{
		CertProviders: map[string]xdsresolver.FileWatcherCertConfig{
			"default": {
				CertificateFile:   filepath.Join(dir, "cert-chain.pem"),
				PrivateKeyFile:    filepath.Join(dir, "key.pem"),
				CACertificateFile: filepath.Join(dir, "root-cert.pem"),
			},
		},
	})
	if err != nil {
		t.Fatalf("grpcInboundTLSConfigFromBootstrap() failed: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintln(w, "nginx v1")
	}))
	defer upstream.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() failed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- serveGRPCInbound(ctx, listener, tlsConfig, upstream.Listener.Addr().String(), func() grpcInboundMTLSMode {
			return grpcInboundMTLSModePermissive
		}, time.Second, time.Second)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("serveGRPCInbound() error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("serveGRPCInbound() did not stop")
		}
	})

	plainResp, err := (&http.Client{Timeout: time.Second}).Get("http://" + listener.Addr().String() + "/")
	if err != nil {
		t.Fatalf("plaintext request failed: %v", err)
	}
	defer plainResp.Body.Close()
	plainBody, err := io.ReadAll(plainResp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll(plaintext) failed: %v", err)
	}
	if got, want := string(plainBody), "nginx v1\n"; got != want {
		t.Fatalf("plaintext body = %q, want %q", got, want)
	}

	cert, err := tls.LoadX509KeyPair(filepath.Join(dir, "client-cert.pem"), filepath.Join(dir, "client-key.pem"))
	if err != nil {
		t.Fatalf("tls.LoadX509KeyPair() failed: %v", err)
	}
	mtlsResp, err := (&http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}}}).Get("https://" + listener.Addr().String() + "/")
	if err != nil {
		t.Fatalf("mTLS request failed: %v", err)
	}
	defer mtlsResp.Body.Close()
	mtlsBody, err := io.ReadAll(mtlsResp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll(mTLS) failed: %v", err)
	}
	if got, want := string(mtlsBody), "nginx v1\n"; got != want {
		t.Fatalf("mTLS body = %q, want %q", got, want)
	}
}

func TestGRPCInboundMTLSModeFromRuntimeConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dubbo-grpc-xds.json")
	if err := os.WriteFile(path, []byte(`{
  "services": [
    {"ports": [{"port": 80, "mtlsMode": "PERMISSIVE"}]},
    {"ports": [{"port": 8080, "mtlsMode": "STRICT"}]}
  ]
}`), 0o600); err != nil {
		t.Fatalf("os.WriteFile() failed: %v", err)
	}

	if got, ok := grpcInboundMTLSModeFromRuntimeConfig(path, 80); !ok || got != grpcInboundMTLSModePermissive {
		t.Fatalf("mode for 80 = %q, %v; want PERMISSIVE, true", got, ok)
	}
	if got, ok := grpcInboundMTLSModeFromRuntimeConfig(path, 8080); !ok || got != grpcInboundMTLSModeStrict {
		t.Fatalf("mode for 8080 = %q, %v; want STRICT, true", got, ok)
	}
}

func newTestCA(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() failed: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("x509.CreateCertificate(CA) failed: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("x509.ParseCertificate(CA) failed: %v", err)
	}
	return cert, key
}

func newSignedCert(t *testing.T, caCert *x509.Certificate, caKey *rsa.PrivateKey, commonName string) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() failed: %v", err)
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatalf("rand.Int() failed: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:     []string{commonName},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("x509.CreateCertificate(%s) failed: %v", commonName, err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("x509.ParseCertificate(%s) failed: %v", commonName, err)
	}
	return cert, key
}

func writePEM(t *testing.T, path, typ string, der []byte) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create(%s) failed: %v", path, err)
	}
	defer file.Close()
	if err := pem.Encode(file, &pem.Block{Type: typ, Bytes: der}); err != nil {
		t.Fatalf("pem.Encode(%s) failed: %v", path, err)
	}
}
