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

package grpcgen

import (
	"reflect"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/memory"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	route "github.com/kdubbo/xds-api/route/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestBuildHTTPRouteProxylessOutboundIgnoresGatewayAttachedHTTPRoute(t *testing.T) {
	push := newRDSTestPushContext(t, []config.Config{
		newWildcardHTTPRouteConfig("httpbin", "default", 8000),
	}, []*model.Service{
		newRDSTestService("nginx", "app", "nginx.app.svc.cluster.local", 80),
		newRDSTestService("httpbin", "default", "httpbin.default.svc.cluster.local", 8000),
	})

	rc := buildHTTPRoute(
		&model.Proxy{ID: "nginx-consumer.app", Type: model.Proxyless},
		push,
		"outbound|80||nginx.app.svc.cluster.local",
	)
	if rc == nil {
		t.Fatal("buildHTTPRoute() returned nil")
	}
	if len(rc.VirtualHosts) != 1 {
		t.Fatalf("VirtualHosts = %d, want 1", len(rc.VirtualHosts))
	}
	if got := rc.VirtualHosts[0].Domains; !contains(got, "nginx.app.svc.cluster.local") {
		t.Fatalf("domains = %v, want nginx host", got)
	}
	if len(rc.VirtualHosts[0].Routes) != 1 {
		t.Fatalf("routes = %d, want 1", len(rc.VirtualHosts[0].Routes))
	}

	if got := rc.VirtualHosts[0].Routes[0].GetRoute().GetCluster(); got != "outbound|80||nginx.app.svc.cluster.local" {
		t.Fatalf("route cluster = %q, want default nginx cluster", got)
	}
}

func TestBuildHTTPRouteProxylessOutboundUsesServiceAttachedHTTPRoute(t *testing.T) {
	push := newRDSTestPushContext(t, []config.Config{
		newServiceAttachedHTTPRouteConfig("reviews-routing", "moviereview", "reviews", 9080),
	}, []*model.Service{
		newRDSTestService("reviews", "moviereview", "reviews.moviereview.svc.cluster.local", 9080),
		newRDSTestService("reviews-v1", "moviereview", "reviews-v1.moviereview.svc.cluster.local", 9080),
		newRDSTestService("reviews-v2", "moviereview", "reviews-v2.moviereview.svc.cluster.local", 9080),
	})

	rc := buildHTTPRoute(
		&model.Proxy{ID: "moviepage.moviereview", Type: model.Proxyless},
		push,
		"outbound|9080||reviews.moviereview.svc.cluster.local",
	)
	if rc == nil {
		t.Fatal("buildHTTPRoute() returned nil")
	}
	routes := rc.VirtualHosts[0].Routes
	if len(routes) != 2 {
		t.Fatalf("routes = %d, want matched route plus fallback", len(routes))
	}

	headers := routes[0].GetMatch().GetHeaders()
	if len(headers) != 1 || headers[0].GetName() != "end-user" || headers[0].GetExactMatch() != "jason" {
		t.Fatalf("first route headers = %v, want end-user exact jason", headers)
	}

	first := weightedClustersByName(t, routes[0])
	wantFirst := map[string]uint32{
		"outbound|9080||reviews-v2.moviereview.svc.cluster.local": 100,
	}
	if !reflect.DeepEqual(first, wantFirst) {
		t.Fatalf("first route weighted clusters = %v, want %v", first, wantFirst)
	}

	fallback := weightedClustersByName(t, routes[1])
	wantFallback := map[string]uint32{
		"outbound|9080||reviews-v1.moviereview.svc.cluster.local": 100,
	}
	if !reflect.DeepEqual(fallback, wantFallback) {
		t.Fatalf("fallback weighted clusters = %v, want %v", fallback, wantFallback)
	}
}

func TestBuildHTTPRouteSetsGatewayAPIRequestTimeout(t *testing.T) {
	cfg := newServiceAttachedHTTPRouteConfig("reviews-timeout", "moviereview", "reviews", 9080)
	spec := cfg.Spec.(*gatewayv1.HTTPRouteSpec)
	spec.Rules[0].Timeouts = &gatewayv1.HTTPRouteTimeouts{
		Request: ptrTo(gatewayv1.Duration("500ms")),
	}
	push := newRDSTestPushContext(t, []config.Config{cfg}, []*model.Service{
		newRDSTestService("reviews", "moviereview", "reviews.moviereview.svc.cluster.local", 9080),
		newRDSTestService("reviews-v1", "moviereview", "reviews-v1.moviereview.svc.cluster.local", 9080),
		newRDSTestService("reviews-v2", "moviereview", "reviews-v2.moviereview.svc.cluster.local", 9080),
	})

	rc := buildHTTPRoute(
		&model.Proxy{ID: "moviepage.moviereview", Type: model.Proxyless},
		push,
		"outbound|9080||reviews.moviereview.svc.cluster.local",
	)
	if rc == nil {
		t.Fatal("buildHTTPRoute() returned nil")
	}
	timeout := rc.VirtualHosts[0].Routes[0].GetRoute().GetTimeout()
	if timeout == nil || timeout.AsDuration() != 500*time.Millisecond {
		t.Fatalf("timeout = %v, want 500ms", timeout)
	}
}

func newRDSTestPushContext(t *testing.T, configs []config.Config, services []*model.Service) *model.PushContext {
	t.Helper()

	store := memory.Make(collections.DubboGatewayAPI())
	for _, cfg := range configs {
		if _, err := store.Create(cfg); err != nil {
			t.Fatalf("create config %s/%s: %v", cfg.Namespace, cfg.Name, err)
		}
	}

	env := model.NewEnvironment()
	env.ConfigStore = store
	env.ServiceDiscovery = staticServiceDiscovery{services: services}
	env.Watcher = meshwatcher.ConfigAdapter(krt.NewStatic(&meshwatcher.MeshConfigResource{
		MeshConfig: mesh.DefaultMeshConfig(),
	}, true))
	env.Init()

	push := model.NewPushContext()
	push.InitContext(env, nil, nil)
	return push
}

func newWildcardHTTPRouteConfig(backendName, backendNamespace string, backendPort int32) config.Config {
	port := backendPort
	weight := int32(1)
	return config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.HTTPRoute,
			Name:             "httpbin",
			Namespace:        backendNamespace,
			Domain:           "cluster.local",
		},
		Spec: &gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{Name: gatewayv1.ObjectName("httpbin-gateway")},
				},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name:      gatewayv1.ObjectName(backendName),
									Namespace: ptrTo(gatewayv1.Namespace(backendNamespace)),
									Port:      &port,
								},
								Weight: &weight,
							},
						},
					},
				},
			},
		},
	}
}

func newServiceAttachedHTTPRouteConfig(name, namespace, parentName string, parentPort int32) config.Config {
	group := gatewayv1.Group("")
	kind := gatewayv1.Kind("Service")
	port := parentPort
	weight := int32(100)
	pathType := gatewayv1.PathMatchPathPrefix
	pathValue := "/"
	headerType := gatewayv1.HeaderMatchExact
	return config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.HTTPRoute,
			Name:             name,
			Namespace:        namespace,
			Domain:           "cluster.local",
		},
		Spec: &gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{
					Group: &group,
					Kind:  &kind,
					Name:  gatewayv1.ObjectName(parentName),
					Port:  &port,
				}},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					Matches: []gatewayv1.HTTPRouteMatch{{
						Path: &gatewayv1.HTTPPathMatch{
							Type:  &pathType,
							Value: &pathValue,
						},
						Headers: []gatewayv1.HTTPHeaderMatch{{
							Type:  &headerType,
							Name:  gatewayv1.HTTPHeaderName("end-user"),
							Value: "jason",
						}},
					}},
					BackendRefs: []gatewayv1.HTTPBackendRef{{
						BackendRef: gatewayv1.BackendRef{
							BackendObjectReference: gatewayv1.BackendObjectReference{
								Name: gatewayv1.ObjectName("reviews-v2"),
								Port: &port,
							},
							Weight: &weight,
						},
					}},
				},
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{{
						BackendRef: gatewayv1.BackendRef{
							BackendObjectReference: gatewayv1.BackendObjectReference{
								Name: gatewayv1.ObjectName("reviews-v1"),
								Port: &port,
							},
							Weight: &weight,
						},
					}},
				},
			},
		},
	}
}

func newRDSTestService(name, namespace, hostname string, port int) *model.Service {
	return &model.Service{
		Hostname: host.Name(hostname),
		Ports: model.PortList{
			{
				Name:     "http",
				Port:     port,
				Protocol: protocol.HTTP2,
			},
		},
		Attributes: model.ServiceAttributes{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func weightedClustersByName(t *testing.T, r *route.Route) map[string]uint32 {
	t.Helper()

	action := r.GetRoute()
	if action == nil {
		t.Fatalf("route action = %T, want RouteAction", r.GetAction())
	}
	weighted := action.GetWeightedClusters()
	if weighted == nil {
		t.Fatalf("cluster specifier = %T, want weighted clusters", action.GetClusterSpecifier())
	}

	out := make(map[string]uint32, len(weighted.GetClusters()))
	for _, cluster := range weighted.GetClusters() {
		out[cluster.GetName()] = cluster.GetWeight().GetValue()
	}
	return out
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func ptrTo[T any](v T) *T {
	return &v
}

type staticServiceDiscovery struct {
	services []*model.Service
}

func (s staticServiceDiscovery) Services() []*model.Service {
	return s.services
}

func (s staticServiceDiscovery) GetService(hostname host.Name) *model.Service {
	for _, svc := range s.services {
		if svc.Hostname == hostname {
			return svc
		}
	}
	return nil
}

func (s staticServiceDiscovery) GetProxyServiceTargets(*model.Proxy) []model.ServiceTarget {
	return nil
}
