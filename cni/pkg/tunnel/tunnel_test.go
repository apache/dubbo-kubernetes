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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/cni/pkg/xdsclient"
)

func TestTunnelForwardsOutboundToInboundOverMTLSHTTP2Connect(t *testing.T) {
	// Start a language-agnostic echo backend simulating the destination app container.
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echoLn.Close()
	go func() {
		for {
			c, err := echoLn.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}(c)
		}
	}()

	caCert, caKey, rootPEM := newTestCA(t)
	srcCertPEM, srcKeyPEM := newTestWorkloadCert(t, caCert, caKey, "spiffe://cluster.local/ns/test-inject/sa/client")
	dstCertPEM, dstKeyPEM := newTestWorkloadCert(t, caCert, caKey, "spiffe://cluster.local/ns/test-inject/sa/server")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Destination node proxy receiving mTLS HTTP/2 CONNECT tunnels.
	dstProxy := NewProxy(Config{
		OutboundAddr:      "127.0.0.1:0",
		InboundTunnelAddr: "127.0.0.1:0",
	})
	dstProxy.RegisterWorkload(&Workload{
		PodName:   "echo-server",
		Namespace: "test-inject",
		PodIP:     "127.0.0.1",
		Certs: xdsclient.CertMaterial{
			CertChainPEM: dstCertPEM,
			KeyPEM:       dstKeyPEM,
			RootCertPEM:  rootPEM,
		},
	})
	if err := dstProxy.Start(ctx); err != nil {
		t.Fatalf("dstProxy.Start() error = %v", err)
	}
	defer dstProxy.Close()

	_, dstTunnelPortStr, err := net.SplitHostPort(dstProxy.InboundTunnelAddr())
	if err != nil {
		t.Fatal(err)
	}
	var dstTunnelPort int
	for _, ch := range dstTunnelPortStr {
		dstTunnelPort = dstTunnelPort*10 + int(ch-'0')
	}

	// Source node proxy forwarding over mTLS HTTP/2 CONNECT tunnel directly to destination tunnel port.
	srcProxy := NewProxy(Config{
		OutboundAddr:      "127.0.0.1:0",
		InboundTunnelAddr: "127.0.0.1:0",
		PeerTunnelPort:    dstTunnelPort,
		DestinationResolver: func(net.Conn) (string, error) {
			return echoLn.Addr().String(), nil
		},
	})
	srcWorkload := &Workload{
		PodName:   "test-app",
		Namespace: "test-inject",
		PodIP:     "127.0.0.1",
		Certs: xdsclient.CertMaterial{
			CertChainPEM: srcCertPEM,
			KeyPEM:       srcKeyPEM,
			RootCertPEM:  rootPEM,
		},
	}
	srcProxy.RegisterWorkload(srcWorkload)
	if err := srcProxy.Start(ctx); err != nil {
		t.Fatalf("srcProxy.Start() error = %v", err)
	}
	defer srcProxy.Close()

	// Exercise forwardViaTunnel directly via a local pipe to verify end-to-end mTLS HTTP/2 CONNECT streaming.
	clientSide, proxySide := net.Pipe()
	defer clientSide.Close()
	errCh := make(chan error, 1)
	go func() {
		defer proxySide.Close()
		errCh <- srcProxy.forwardViaTunnel(ctx, proxySide, srcWorkload, echoLn.Addr().String(), 3*time.Second)
	}()

	const payload = "hello-triple-ambient-tunnel"
	if _, err := clientSide.Write([]byte(payload)); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	buf := make([]byte, len(payload))
	if _, err := io.ReadFull(clientSide, buf); err != nil {
		t.Fatalf("read echoed payload: %v", err)
	}
	if string(buf) != payload {
		t.Fatalf("echoed = %q, want %q", string(buf), payload)
	}
	_ = clientSide.Close()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("forwardViaTunnel() error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for forwardViaTunnel completion")
	}

	if dstProxy.Metrics().InboundTotal < 1 {
		t.Fatalf("expected dstProxy InboundTotal >= 1, got %+v", dstProxy.Metrics())
	}
}

func newTestCA(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "dubbo-tunnel-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert, key, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func newTestWorkloadCert(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey, spiffeURI string) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(spiffeURI)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		URIs:         []*url.URL{u},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
}
