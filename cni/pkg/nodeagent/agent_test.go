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

package nodeagent

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	cniconstants "github.com/apache/dubbo-kubernetes/cni/pkg/constants"
	"github.com/apache/dubbo-kubernetes/cni/pkg/iptables"
	"github.com/apache/dubbo-kubernetes/cni/pkg/plugin"
	"github.com/apache/dubbo-kubernetes/cni/pkg/tunnel"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/memory"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/serviceregistry/aggregate"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/util/protoconv"
	discoveryxds "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/server/ca/authenticate"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	meshconfig "github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	pkgmodel "github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/dubml/api/annotation"
	"github.com/dubml/api/label"
	hcmv1 "github.com/dubml/xds-api/extensions/filters/v1/network/http_connection_manager"
	listenerv1 "github.com/dubml/xds-api/listener/v1"
	routev1 "github.com/dubml/xds-api/route/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type fakeRedirector struct {
	mu       sync.Mutex
	applied  []string
	cleaned  []string
}

func (f *fakeRedirector) Apply(_ context.Context, netns string, _ iptables.Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.applied = append(f.applied, netns)
	return nil
}

func (f *fakeRedirector) Cleanup(_ context.Context, netns string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cleaned = append(f.cleaned, netns)
	return nil
}

func TestIsEnrolledPod(t *testing.T) {
	enrolledNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test-inject",
			Labels: map[string]string{label.OrgApacheDubboInherentInject.Name: "true"},
		},
	}
	plainNS := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}}

	cases := []struct {
		name string
		pod  *corev1.Pod
		ns   *corev1.Namespace
		want bool
	}{
		{
			name: "triple-engine template pod",
			pod: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Name: "test-app", Namespace: "test-inject",
				Annotations: map[string]string{annotation.OrgApacheDubboInjectTemplates.Name: inject.InherentTripleTemplateName},
			}},
			ns:   plainNS,
			want: true,
		},
		{
			name: "namespace-enrolled arbitrary language pod without pod annotations",
			pod:  &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "python-app", Namespace: "test-inject"}},
			ns:   enrolledNS,
			want: true,
		},
		{
			name: "explicit pod opt-out overrides namespace",
			pod: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Name: "opt-out", Namespace: "test-inject",
				Annotations: map[string]string{cniconstants.RedirectionAnnotation: cniconstants.RedirectionDisabledValue},
			}},
			ns:   enrolledNS,
			want: false,
		},
		{
			name: "hostNetwork excluded",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "host-net", Namespace: "test-inject"},
				Spec:       corev1.PodSpec{HostNetwork: true},
			},
			ns:   enrolledNS,
			want: false,
		},
		{
			name: "dubbo-system excluded",
			pod: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Name: "dubbod", Namespace: "dubbo-system",
				Annotations: map[string]string{annotation.OrgApacheDubboInjectTemplates.Name: inject.InherentTripleTemplateName},
			}},
			ns:   enrolledNS,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsEnrolledPod(tc.pod, tc.ns, "dubbo-system"); got != tc.want {
				t.Fatalf("IsEnrolledPod() = %v, want %v", got, tc.want)
			}
		})
	}
}

type simpleLDSGen struct{}

func (simpleLDSGen) Generate(_ *model.Proxy, w *model.WatchedResource, _ *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	if w.TypeUrl != pkgmodel.ListenerType {
		return nil, model.DefaultXdsLogDetails, nil
	}
	hcm := &hcmv1.HttpConnectionManager{
		RouteSpecifier: &hcmv1.HttpConnectionManager_RouteConfig{
			RouteConfig: &routev1.RouteConfiguration{Name: "inline"},
		},
	}
	return model.Resources{{
		Name: "app.test-inject.svc.cluster.local:80",
		Resource: protoconv.MessageToAny(&listenerv1.Listener{
			Name:        "app.test-inject.svc.cluster.local:80",
			ApiListener: &listenerv1.ApiListener{ApiListener: protoconv.MessageToAny(hcm)},
		}),
	}}, model.DefaultXdsLogDetails, nil
}

func TestAgentReconcilesEnrolledPodConnectsToDubbodAndHandlesCNIPluginEvents(t *testing.T) {
	caCert, caKey, rootPEM := newAgentTestCA(t)
	srvCertPEM, srvKeyPEM := newAgentTestCert(t, caCert, caKey, []string{"dubbod.dubbo-system.svc"}, "")
	srvTLSCert, err := tls.X509KeyPair(srvCertPEM, srvKeyPEM)
	if err != nil {
		t.Fatal(err)
	}
	podCertPEM, podKeyPEM := newAgentTestCert(t, caCert, caKey, nil, "spiffe://cluster.local/ns/test-inject/sa/default")

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
	ds.Authorize = func(_ *model.Proxy, _ []string) error { return nil }
	ds.Generators = map[string]model.XdsResourceGenerator{
		"triple/" + pkgmodel.ListenerType: simpleLDSGen{},
	}
	ds.CachesSynced()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	grpcSrv := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{srvTLSCert},
		ClientCAs:    rootPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	})))
	ds.Register(grpcSrv)
	go func() { _ = grpcSrv.Serve(ln) }()
	defer grpcSrv.Stop()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test-inject",
			Annotations: map[string]string{
				annotation.OrgApacheDubboInjectTemplates.Name: inject.InherentTripleTemplateName,
			},
		},
		Spec:   corev1.PodSpec{NodeName: "node-1"},
		Status: corev1.PodStatus{PodIP: "10.244.0.9"},
	}
	secretName := inject.InherentXDSSecretNameForMeta(pod.ObjectMeta)
	bootstrapJSON := fmt.Sprintf(`{
  "xds_servers": [{
    "server_uri": %q,
    "channel_creds": [{"type": "tls"}]
  }],
  "node": {
    "id": "inherent~10.244.0.9~test-app.test-inject~test-inject.svc.cluster.local",
    "metadata": {"GENERATOR": "triple", "NAMESPACE": "test-inject", "CLUSTER_ID": "Kubernetes"}
  }
}`, ln.Addr().String())
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: "test-inject"},
		Data: map[string][]byte{
			inject.InherentTripleBootstrapFileName:      []byte(bootstrapJSON),
			constants.CertChainFilename:                 podCertPEM,
			constants.KeyFilename:                       podKeyPEM,
			constants.CACertNamespaceConfigMapDataName:  rootPEM,
		},
	}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-inject"}}

	kubeClient := fake.NewSimpleClientset(ns, pod, secret)
	redirector := &fakeRedirector{}
	sockDir, err := os.MkdirTemp("/tmp", "dcni-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(sockDir) })
	sockPath := filepath.Join(sockDir, "cni.sock")

	agent := New(kubeClient, Config{
		NodeName:        "node-1",
		SystemNamespace: "dubbo-system",
		SocketPath:      sockPath,
		StatusAddr:      "127.0.0.1:0",
		TunnelConfig: tunnel.Config{
			OutboundAddr:      "127.0.0.1:0",
			InboundTunnelAddr: "127.0.0.1:0",
		},
		Redirector: redirector,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := agent.Start(ctx); err != nil {
		t.Fatalf("agent.Start() error = %v", err)
	}
	defer agent.Close()

	// Simulate dubbo-cni plugin ADD callback over Unix domain socket.
	sender := &plugin.UDSEventSender{Timeout: 2 * time.Second}
	if err := sender.Send(ctx, sockPath, plugin.PodEvent{
		Command:     plugin.CmdAdd,
		ContainerID: "ctr-1",
		NetNS:       "/var/run/netns/test-app",
		IfName:      "eth0",
		PodName:     "test-app",
		Namespace:   "test-inject",
		PodIP:       "10.244.0.9",
	}); err != nil {
		t.Fatalf("UDS ADD callback failed: %v", err)
	}

	// Wait until dubbod /debug/syncz reports test-app.test-inject connected & SYNCED.
	deadline := time.Now().Add(5 * time.Second)
	synced := false
	for time.Now().Before(deadline) {
		syncz := ds.Syncz()
		if len(syncz) == 1 && syncz[0].ProxyID == "test-app.test-inject" {
			for _, wt := range syncz[0].WatchedTypes {
				if wt.TypeURL == pkgmodel.ListenerType && wt.Synced {
					synced = true
					break
				}
			}
			if synced {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !synced {
		t.Fatalf("dubbod Syncz never showed test-app.test-inject SYNCED; syncz=%+v pods=%+v", ds.Syncz(), agent.ManagedPods())
	}

	redirector.mu.Lock()
	appliedCount := len(redirector.applied)
	redirector.mu.Unlock()
	if appliedCount != 1 {
		t.Fatalf("expected iptables Apply called once, got %d", appliedCount)
	}

	// Verify node /debug/pods HTTP endpoint reports the managed pod.
	resp, err := http.Get("http://" + agent.StatusAddr() + "/debug/pods")
	if err != nil {
		t.Fatalf("GET /debug/pods error = %v", err)
	}
	defer resp.Body.Close()
	var infos []ManagedPodInfo
	if err := json.NewDecoder(resp.Body).Decode(&infos); err != nil {
		t.Fatalf("decode /debug/pods: %v", err)
	}
	if len(infos) != 1 || infos[0].PodName != "test-app" || !infos[0].Redirected || !infos[0].XDS.Connected {
		t.Fatalf("unexpected /debug/pods output: %+v", infos)
	}

	// Simulate CNI DEL and verify teardown cleans iptables and disconnects ADS client.
	if err := sender.Send(ctx, sockPath, plugin.PodEvent{
		Command:   plugin.CmdDel,
		PodName:   "test-app",
		Namespace: "test-inject",
	}); err != nil {
		t.Fatalf("UDS DEL callback failed: %v", err)
	}
	redirector.mu.Lock()
	cleanedCount := len(redirector.cleaned)
	redirector.mu.Unlock()
	if cleanedCount != 1 {
		t.Fatalf("expected iptables Cleanup called once, got %d", cleanedCount)
	}
}

func newAgentTestCA(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "dubbo-agent-ca"},
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

func newAgentTestCert(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey, dns []string, spiffeURI string) ([]byte, []byte) {
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
