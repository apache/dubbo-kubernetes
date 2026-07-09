package grpcgen

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/grpcxds"
	security "github.com/kdubbo/api/security/v1alpha3"
	cluster "github.com/kdubbo/xds-api/cluster/v1"
	tlsv1 "github.com/kdubbo/xds-api/extensions/transport_sockets/tls/v1"
	listener "github.com/kdubbo/xds-api/listener/v1"
)

func TestPeerAuthenticationStrictBuildsDownstreamMTLSListener(t *testing.T) {
	svc := newRDSTestService("nginx", "app", "nginx.app.svc.cluster.local", 80)
	proxy := newMTLSTestProxy()
	proxy.ServiceTargets = []model.ServiceTarget{{
		Service: svc,
		Port: model.ServiceInstancePort{
			ServicePort: svc.Ports[0],
			TargetPort:  80,
		},
	}}
	push := newRDSTestPushContext(t, []config.Config{
		newStrictPeerAuthenticationConfig("app-strict-mtls", "app"),
	}, []*model.Service{svc})

	listenerName := grpcxds.ServerListenerNamePrefix + "0.0.0.0:80"
	resources := (&GrpcConfigGenerator{}).BuildListeners(proxy, push, []string{listenerName})
	l := findListener(t, resources, listenerName)
	if len(l.GetFilterChains()) != 1 {
		t.Fatalf("filter chains = %d, want 1", len(l.GetFilterChains()))
	}
	transportSocket := l.GetFilterChains()[0].GetTransportSocket()
	if transportSocket == nil {
		t.Fatalf("listener %s has no transport socket", listenerName)
	}

	tlsContext := &tlsv1.DownstreamTlsContext{}
	if err := transportSocket.GetTypedConfig().UnmarshalTo(tlsContext); err != nil {
		t.Fatalf("unmarshal downstream tls context: %v", err)
	}
	if tlsContext.GetRequireClientCertificate() == nil || !tlsContext.GetRequireClientCertificate().GetValue() {
		t.Fatalf("RequireClientCertificate = %v, want true", tlsContext.GetRequireClientCertificate())
	}
	assertCommonTLSContext(t, tlsContext.GetCommonTlsContext())
}

func TestPeerAuthenticationPermissiveBuildsPlaintextAndMTLSFilterChains(t *testing.T) {
	svc := newRDSTestService("nginx", "app", "nginx.app.svc.cluster.local", 80)
	proxy := newMTLSTestProxy()
	proxy.ServiceTargets = []model.ServiceTarget{{
		Service: svc,
		Port: model.ServiceInstancePort{
			ServicePort: svc.Ports[0],
			TargetPort:  80,
		},
	}}
	push := newRDSTestPushContext(t, []config.Config{
		newPeerAuthenticationConfig("app-permissive-mtls", "app", security.PeerAuthentication_MutualTLS_PERMISSIVE),
	}, []*model.Service{svc})

	listenerName := grpcxds.ServerListenerNamePrefix + "0.0.0.0:80"
	resources := (&GrpcConfigGenerator{}).BuildListeners(proxy, push, []string{listenerName})
	l := findListener(t, resources, listenerName)
	if len(l.GetFilterChains()) != 2 {
		t.Fatalf("filter chains = %d, want plaintext plus mTLS", len(l.GetFilterChains()))
	}

	mtlsChain := l.GetFilterChains()[0]
	if mtlsChain.GetTransportSocket() == nil {
		t.Fatalf("mTLS filter chain has no transport socket")
	}
	if got := mtlsChain.GetFilterChainMatch().GetTransportProtocol(); got != "tls" {
		t.Fatalf("mTLS filter chain transport protocol = %q, want tls", got)
	}
	tlsContext := &tlsv1.DownstreamTlsContext{}
	if err := mtlsChain.GetTransportSocket().GetTypedConfig().UnmarshalTo(tlsContext); err != nil {
		t.Fatalf("unmarshal downstream tls context: %v", err)
	}
	if tlsContext.GetRequireClientCertificate() == nil || !tlsContext.GetRequireClientCertificate().GetValue() {
		t.Fatalf("RequireClientCertificate = %v, want true", tlsContext.GetRequireClientCertificate())
	}

	plaintextChain := l.GetFilterChains()[1]
	if plaintextChain.GetTransportSocket() != nil {
		t.Fatalf("plaintext filter chain has transport socket")
	}
}

func TestRootNamespacePeerAuthenticationBuildsGlobalStrictListener(t *testing.T) {
	svc := newRDSTestService("nginx", "app", "nginx.app.svc.cluster.local", 80)
	proxy := newMTLSTestProxy()
	proxy.ServiceTargets = []model.ServiceTarget{{
		Service: svc,
		Port: model.ServiceInstancePort{
			ServicePort: svc.Ports[0],
			TargetPort:  80,
		},
	}}
	push := newRDSTestPushContext(t, []config.Config{
		newPeerAuthenticationConfig("default", "dubbo-system", security.PeerAuthentication_MutualTLS_STRICT),
	}, []*model.Service{svc})

	listenerName := grpcxds.ServerListenerNamePrefix + "0.0.0.0:80"
	resources := (&GrpcConfigGenerator{}).BuildListeners(proxy, push, []string{listenerName})
	l := findListener(t, resources, listenerName)
	if len(l.GetFilterChains()) != 1 {
		t.Fatalf("filter chains = %d, want 1", len(l.GetFilterChains()))
	}
	if l.GetFilterChains()[0].GetTransportSocket() == nil {
		t.Fatalf("global STRICT listener %s has no transport socket", listenerName)
	}
}

func newMTLSTestProxy() *model.Proxy {
	return &model.Proxy{
		ID:              "proxyless~10.0.0.1~nginx-consumer.app~app.svc.cluster.local",
		Type:            model.Proxyless,
		DNSDomain:       "app.svc.cluster.local",
		ConfigNamespace: "app",
		Metadata: &model.NodeMetadata{
			Namespace: "app",
		},
	}
}

func newStrictPeerAuthenticationConfig(name, namespace string) config.Config {
	return newPeerAuthenticationConfig(name, namespace, security.PeerAuthentication_MutualTLS_STRICT)
}

func newPeerAuthenticationConfig(name, namespace string, mode security.PeerAuthentication_MutualTLS_Mode) config.Config {
	return config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.PeerAuthentication,
			Name:             name,
			Namespace:        namespace,
			Domain:           "cluster.local",
		},
		Spec: &security.PeerAuthentication{
			Mtls: &security.PeerAuthentication_MutualTLS{
				Mode: mode,
			},
		},
	}
}

func findCluster(t *testing.T, resources model.Resources, name string) *cluster.Cluster {
	t.Helper()
	for _, resource := range resources {
		if resource.Name != name {
			continue
		}
		c := &cluster.Cluster{}
		if err := resource.Resource.UnmarshalTo(c); err != nil {
			t.Fatalf("unmarshal cluster %s: %v", name, err)
		}
		return c
	}
	t.Fatalf("cluster %s not found in %d resources", name, len(resources))
	return nil
}

func findListener(t *testing.T, resources model.Resources, name string) *listener.Listener {
	t.Helper()
	for _, resource := range resources {
		if resource.Name != name {
			continue
		}
		l := &listener.Listener{}
		if err := resource.Resource.UnmarshalTo(l); err != nil {
			t.Fatalf("unmarshal listener %s: %v", name, err)
		}
		return l
	}
	t.Fatalf("listener %s not found in %d resources", name, len(resources))
	return nil
}

func assertCommonTLSContext(t *testing.T, ctx *tlsv1.CommonTlsContext) {
	t.Helper()
	if ctx == nil {
		t.Fatal("common TLS context is nil")
	}
	if got := ctx.GetAlpnProtocols(); len(got) != 1 || got[0] != "h2" {
		t.Fatalf("ALPN protocols = %v, want [h2]", got)
	}
	cert := ctx.GetTlsCertificateCertificateProviderInstance()
	if cert == nil || cert.GetInstanceName() != "default" || cert.GetCertificateName() != "default" {
		t.Fatalf("workload cert provider = %v, want default/default", cert)
	}
	combined := ctx.GetCombinedValidationContext()
	if combined == nil {
		t.Fatal("combined validation context is nil")
	}
	root := combined.GetValidationContextCertificateProviderInstance()
	if root == nil || root.GetInstanceName() != "default" || root.GetCertificateName() != "ROOTCA" {
		t.Fatalf("root cert provider = %v, want default/ROOTCA", root)
	}
}
