package endpoints

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/memory"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	networking "github.com/kdubbo/api/networking/v1alpha3"
	endpoint "github.com/kdubbo/xds-api/endpoint/v1"
)

func TestBuildClusterLoadAssignmentUsesXServerPortForDUBBOMutual(t *testing.T) {
	hostname := host.Name("nginx.app.svc.cluster.local")
	svc := newEndpointTestService("nginx", "app", string(hostname), 80)
	push := newEndpointTestPushContext(t, []config.Config{
		newEndpointTestMTLSMeshService("nginx-routing", "app", hostname),
	}, []*model.Service{svc})
	index := model.NewEndpointIndex(model.DisabledCache{})
	index.UpdateServiceEndpoints(model.ShardKey{}, string(hostname), "app", []*model.DubboEndpoint{{
		Addresses:       []string{"10.0.0.1"},
		EndpointPort:    80,
		ServicePortName: "http",
		Labels:          map[string]string{"version": "v1"},
		HealthStatus:    model.Healthy,
	}}, false)

	clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "v1", hostname, 80)
	builder := NewEndpointBuilder(clusterName, newEndpointTestProxy(), push)
	cla := builder.BuildClusterLoadAssignment(index)

	if got := firstEndpointPort(t, cla); got != inject.ProxylessXServerPort {
		t.Fatalf("endpoint port = %d, want xserver port %d", got, inject.ProxylessXServerPort)
	}
}

func TestBuildClusterLoadAssignmentKeepsAppPortWithoutDUBBOMutual(t *testing.T) {
	hostname := host.Name("nginx.app.svc.cluster.local")
	svc := newEndpointTestService("nginx", "app", string(hostname), 80)
	push := newEndpointTestPushContext(t, nil, []*model.Service{svc})
	index := model.NewEndpointIndex(model.DisabledCache{})
	index.UpdateServiceEndpoints(model.ShardKey{}, string(hostname), "app", []*model.DubboEndpoint{{
		Addresses:       []string{"10.0.0.1"},
		EndpointPort:    80,
		ServicePortName: "http",
		HealthStatus:    model.Healthy,
	}}, false)

	clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", hostname, 80)
	builder := NewEndpointBuilder(clusterName, newEndpointTestProxy(), push)
	cla := builder.BuildClusterLoadAssignment(index)

	if got := firstEndpointPort(t, cla); got != 80 {
		t.Fatalf("endpoint port = %d, want app port 80", got)
	}
}

func firstEndpointPort(t *testing.T, cla *endpoint.ClusterLoadAssignment) uint32 {
	t.Helper()
	localities := cla.GetEndpoints()
	if len(localities) == 0 || len(localities[0].GetLbEndpoints()) == 0 {
		t.Fatalf("CLA has no endpoints")
	}
	return localities[0].GetLbEndpoints()[0].GetEndpoint().GetAddress().GetSocketAddress().GetPortValue()
}

func newEndpointTestPushContext(t *testing.T, configs []config.Config, services []*model.Service) *model.PushContext {
	t.Helper()
	store := memory.Make(collections.DubboGatewayAPI())
	for _, cfg := range configs {
		if _, err := store.Create(cfg); err != nil {
			t.Fatalf("create config %s/%s: %v", cfg.Namespace, cfg.Name, err)
		}
	}
	env := model.NewEnvironment()
	env.ConfigStore = store
	env.ServiceDiscovery = endpointTestServiceDiscovery{services: services}
	env.Watcher = meshwatcher.ConfigAdapter(krt.NewStatic(&meshwatcher.MeshConfigResource{
		MeshConfig: mesh.DefaultMeshConfig(),
	}, true))
	env.Init()
	push := model.NewPushContext()
	push.InitContext(env, nil, nil)
	return push
}

func newEndpointTestProxy() *model.Proxy {
	return &model.Proxy{
		ID:              "proxyless~10.0.0.2~nginx-consumer.app~app.svc.cluster.local",
		Type:            model.Proxyless,
		ConfigNamespace: "app",
		Metadata: &model.NodeMetadata{
			Generator: "grpc",
			Namespace: "app",
		},
	}
}

func newEndpointTestService(name, namespace, hostname string, port int) *model.Service {
	return &model.Service{
		Hostname: host.Name(hostname),
		Ports: model.PortList{{
			Name:     "http",
			Port:     port,
			Protocol: protocol.HTTP2,
		}},
		Attributes: model.ServiceAttributes{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func newEndpointTestMTLSMeshService(name, namespace string, hostname host.Name) config.Config {
	return config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.MeshService,
			Name:             name,
			Namespace:        namespace,
			Domain:           "cluster.local",
		},
		Spec: &networking.MeshService{
			Hosts: []string{string(hostname)},
			TrafficPolicy: &networking.TrafficPolicy{
				Tls: &networking.ClientTLSSettings{Mode: networking.ClientTLSSettings_DUBBO_MUTUAL},
			},
			Routes: []*networking.MeshServiceRoute{{
				Service: []*networking.ServiceDestination{{
					Name:   "v1",
					Host:   string(hostname),
					Labels: map[string]string{"version": "v1"},
					Port:   &networking.ServicePort{Number: 80},
					Weight: 100,
				}},
			}},
		},
	}
}

type endpointTestServiceDiscovery struct {
	services []*model.Service
}

func (s endpointTestServiceDiscovery) Services() []*model.Service {
	return s.services
}

func (s endpointTestServiceDiscovery) GetService(hostname host.Name) *model.Service {
	for _, svc := range s.services {
		if svc.Hostname == hostname {
			return svc
		}
	}
	return nil
}

func (s endpointTestServiceDiscovery) GetProxyServiceTargets(*model.Proxy) []model.ServiceTarget {
	return nil
}
