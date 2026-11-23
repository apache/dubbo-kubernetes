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

package xdsclient

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/memory"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/serviceregistry/aggregate"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/util/protoconv"
	discoveryxds "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/server/ca/authenticate"
	meshconfig "github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	pkgmodel "github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	clusterv1 "github.com/dubml/xds-api/cluster/v1"
	corev1 "github.com/dubml/xds-api/core/v1"
	endpointv1 "github.com/dubml/xds-api/endpoint/v1"
	hcmv1 "github.com/dubml/xds-api/extensions/filters/v1/network/http_connection_manager"
	tlsv1 "github.com/dubml/xds-api/extensions/transport_sockets/tls/v1"
	listenerv1 "github.com/dubml/xds-api/listener/v1"
	routev1 "github.com/dubml/xds-api/route/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestResolveConfigPrecedenceArgsOverEnvOverManual(t *testing.T) {
	t.Setenv("XDS_ADDRESS", "env.dubbod.svc:26012")
	t.Setenv("POD_NAMESPACE", "env-ns")
	t.Setenv("DUBBO_META_GENERATOR", "triple")

	cfg, err := ResolveConfig(Config{
		ServerURI: "manual.dubbod.svc:26012",
		Namespace: "manual-ns",
		Insecure:  true,
	}, []string{"--xds-address=arg.dubbod.svc:26012", "--xds-namespace=arg-ns"})
	if err != nil {
		t.Fatalf("ResolveConfig() error = %v", err)
	}
	if cfg.ServerURI != "arg.dubbod.svc:26012" {
		t.Fatalf("ServerURI = %q, want args override", cfg.ServerURI)
	}
	if cfg.Namespace != "arg-ns" {
		t.Fatalf("Namespace = %q, want arg-ns", cfg.Namespace)
	}

	// Env fallback when manual is empty and no args are supplied.
	cfgEnv, err := ResolveConfig(Config{Insecure: true}, nil)
	if err != nil {
		t.Fatalf("ResolveConfig(env) error = %v", err)
	}
	if cfgEnv.ServerURI != "env.dubbod.svc:26012" || cfgEnv.Namespace != "env-ns" {
		t.Fatalf("unexpected env config: %+v", cfgEnv)
	}

	// Bootstrap file parsing via TRIPLE_XDS_BOOTSTRAP.
	tmp := t.TempDir()
	bootstrapPath := filepath.Join(tmp, "triple-bootstrap.json")
	bootstrapJSON := `{
  "xds_servers": [{
    "server_uri": "dubbod.dubbo-system.svc:26012",
    "channel_creds": [{"type": "tls", "config": {
      "certificate_file": "/etc/dubbo/proxy/cert-chain.pem",
      "private_key_file": "/etc/dubbo/proxy/key.pem",
      "ca_certificate_file": "/etc/dubbo/proxy/root-cert.pem"
    }}]
  }],
  "node": {
    "id": "inherent~10.244.0.9~test-app.test-inject~test-inject.svc.cluster.local",
    "metadata": {"GENERATOR": "triple", "NAMESPACE": "test-inject", "CLUSTER_ID": "Kubernetes"}
  }
}`
	if err := os.WriteFile(bootstrapPath, []byte(bootstrapJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDS_ADDRESS", "")
	t.Setenv("TRIPLE_XDS_BOOTSTRAP", bootstrapPath)
	cfgBoot, err := ResolveConfig(Config{}, nil)
	if err != nil {
		t.Fatalf("ResolveConfig(bootstrap) error = %v", err)
	}
	if cfgBoot.ServerURI != "dubbod.dubbo-system.svc:26012" || cfgBoot.Node.GetId() != "inherent~10.244.0.9~test-app.test-inject~test-inject.svc.cluster.local" {
		t.Fatalf("unexpected bootstrap config: %+v", cfgBoot)
	}
}

type staticGenerator struct {
	resources map[string]model.Resources
}

func (g staticGenerator) Generate(_ *model.Proxy, w *model.WatchedResource, _ *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	return g.resources[w.TypeUrl], model.DefaultXdsLogDetails, nil
}

func TestClientSyncsWithDubbodMTLSAndAppearsInSyncz(t *testing.T) {
	caCert, caKey, rootPEM := generateTestCA(t)
	serverCert := generateTestCert(t, caCert, caKey, []string{"dubbod.dubbo-system.svc", "localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, "")
	clientCertPEM, clientKeyPEM := generateTestCertPEM(t, caCert, caKey, nil, nil, "spiffe://cluster.local/ns/test-inject/sa/default")

	rootPool := x509.NewCertPool()
	rootPool.AppendCertsFromPEM(rootPEM)

	env := model.NewEnvironment()
	env.ServiceDiscovery = aggregate.NewController(aggregate.Options{})
	env.ConfigStore = memory.Make(collection.Schemas{})
	env.Watcher = meshwatcher.ConfigAdapter(krt.NewStatic(&meshwatcher.MeshConfigResource{
		MeshConfig: meshconfig.DefaultMeshConfig(),
	}, true))
	env.Init()
	push := model.NewPushContext()
	push.PushVersion = "test-v1"
	push.InitContext(env, nil, nil)
	env.SetPushContext(push)
	ds := discoveryxds.NewDiscoveryServer(env, nil)
	ds.Authenticators = []security.Authenticator{&authenticate.ClientCertAuthenticator{}}
	ds.Authorize = func(_ *model.Proxy, ids []string) error {
		if len(ids) == 0 {
			return context.DeadlineExceeded
		}
		return nil
	}

	listenerName := "echo.test-inject.svc.cluster.local:8080"
	clusterName := "outbound|8080||echo.test-inject.svc.cluster.local"
	hcm := &hcmv1.HttpConnectionManager{
		RouteSpecifier: &hcmv1.HttpConnectionManager_Rds{
			Rds: &hcmv1.Rds{RouteConfigName: clusterName},
		},
	}
	gen := staticGenerator{
		resources: map[string]model.Resources{
			pkgmodel.ListenerType: {{
				Name: listenerName,
				Resource: protoconv.MessageToAny(&listenerv1.Listener{
					Name:        listenerName,
					ApiListener: &listenerv1.ApiListener{ApiListener: protoconv.MessageToAny(hcm)},
				}),
			}},
			pkgmodel.RouteType: {{
				Name: clusterName,
				Resource: protoconv.MessageToAny(&routev1.RouteConfiguration{
					Name: clusterName,
					VirtualHosts: []*routev1.VirtualHost{{
						Name:    "vh",
						Domains: []string{"*"},
						Routes: []*routev1.Route{{
							Match: &routev1.RouteMatch{PathSpecifier: &routev1.RouteMatch_Prefix{Prefix: "/"}},
							Action: &routev1.Route_Route{
								Route: &routev1.RouteAction{
									ClusterSpecifier: &routev1.RouteAction_Cluster{Cluster: clusterName},
								},
							},
						}},
					}},
				}),
			}},
			pkgmodel.ClusterType: {{
				Name: clusterName,
				Resource: protoconv.MessageToAny(&clusterv1.Cluster{
					Name:                 clusterName,
					ClusterDiscoveryType: &clusterv1.Cluster_Type{Type: clusterv1.Cluster_EDS},
					EdsClusterConfig:     &clusterv1.Cluster_EdsClusterConfig{ServiceName: clusterName},
					TransportSocket: &corev1.TransportSocket{
						Name: "tls",
						ConfigType: &corev1.TransportSocket_TypedConfig{
							TypedConfig: protoconv.MessageToAny(&tlsv1.UpstreamTlsContext{Sni: "echo.test-inject.svc.cluster.local"}),
						},
					},
				}),
			}},
			pkgmodel.EndpointType: {{
				Name: clusterName,
				Resource: protoconv.MessageToAny(&endpointv1.ClusterLoadAssignment{
					ClusterName: clusterName,
					Endpoints: []*endpointv1.LocalityLbEndpoints{{
						LbEndpoints: []*endpointv1.LbEndpoint{{
							HealthStatus: corev1.HealthStatus_HEALTHY,
							HostIdentifier: &endpointv1.LbEndpoint_Endpoint{
								Endpoint: &endpointv1.Endpoint{
									Address: &corev1.Address{
										Address: &corev1.Address_SocketAddress{
											SocketAddress: &corev1.SocketAddress{
												Address:       "10.244.0.12",
												PortSpecifier: &corev1.SocketAddress_PortValue{PortValue: 8080},
											},
										},
									},
								},
							},
						}},
					}},
				}),
			}},
		},
	}
	ds.Generators = map[string]model.XdsResourceGenerator{
		"triple/" + pkgmodel.ListenerType: gen,
		"triple/" + pkgmodel.RouteType:    gen,
		"triple/" + pkgmodel.ClusterType:  gen,
		"triple/" + pkgmodel.EndpointType: gen,
	}
	ds.CachesSynced()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	grpcSrv := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    rootPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	})))
	ds.Register(grpcSrv)
	go func() { _ = grpcSrv.Serve(ln) }()
	defer grpcSrv.Stop()

	client, err := New(Config{
		ServerURI:  ln.Addr().String(),
		ServerName: "dubbod.dubbo-system.svc",
		NodeID:     "inherent~10.244.0.9~test-app.test-inject~test-inject.svc.cluster.local",
		Namespace:  "test-inject",
		ClusterID:  "Kubernetes",
		Generator:  "triple",
		Certs: CertMaterial{
			CertChainPEM: clientCertPEM,
			KeyPEM:       clientKeyPEM,
			RootCertPEM:  rootPEM,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = client.Run(ctx) }()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snap := client.Snapshot()
		if snap.Ready {
			sel, err := snap.Select(listenerName, listenerName, "/hello", 1)
			if err == nil && sel.Endpoint.HostPort() == "10.244.0.12:8080" && sel.UseMTLS {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !client.Snapshot().Ready {
		t.Fatalf("client snapshot never became ready; status=%+v", client.Status())
	}

	// Verify dubbod /debug/syncz reports test-app as connected and SYNCED.
	var synced bool
	for time.Now().Before(deadline) {
		syncz := ds.Syncz()
		if len(syncz) == 1 && syncz[0].ProxyID == "test-app.test-inject" {
			allSynced := len(syncz[0].WatchedTypes) >= 4
			for _, wt := range syncz[0].WatchedTypes {
				if !wt.Synced {
					allSynced = false
				}
			}
			if allSynced {
				synced = true
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !synced {
		t.Fatalf("dubbod Syncz did not report test-app.test-inject SYNCED: %+v", ds.Syncz())
	}
}

func generateTestCA(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "dubbo-test-ca"},
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

func generateTestCert(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey, dns []string, ips []net.IP, spiffeURI string) tls.Certificate {
	t.Helper()
	certPEM, keyPEM := generateTestCertPEM(t, caCert, caKey, dns, ips, spiffeURI)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func generateTestCertPEM(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey, dns []string, ips []net.IP, spiffeURI string) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     dns,
		IPAddresses:  ips,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	if spiffeURI != "" {
		u, err := url.Parse(spiffeURI)
		if err != nil {
			t.Fatal(err)
		}
		tmpl.URIs = []*url.URL{u}
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
