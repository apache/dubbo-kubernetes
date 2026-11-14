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

package triplegen

import (
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode"

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
	networking "github.com/dubml/api/networking/v1alpha3"
	route "github.com/dubml/xds-api/route/v1"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestBuildRouteMatchesPreservesHTTPRouteORSemantics(t *testing.T) {
	pathType := gatewayv1.PathMatchPathPrefix
	users := "/users"
	orders := "/orders"
	matches := buildRouteMatchesFromHTTPRouteMatches([]gatewayv1.HTTPRouteMatch{
		{Path: &gatewayv1.HTTPPathMatch{Type: &pathType, Value: &users}},
		{Path: &gatewayv1.HTTPPathMatch{Type: &pathType, Value: &orders}},
	})

	if len(matches) != 2 {
		t.Fatalf("matches = %d, want 2 independent xDS routes", len(matches))
	}
	if got := matches[0].GetPrefix(); got != users {
		t.Fatalf("first prefix = %q, want %q", got, users)
	}
	if got := matches[1].GetPrefix(); got != orders {
		t.Fatalf("second prefix = %q, want %q", got, orders)
	}
}

func TestBuildHTTPRouteInherentOutboundIgnoresGatewayAttachedHTTPRoute(t *testing.T) {
	push := newRDSTestPushContext(t, []config.Config{
		newWildcardHTTPRouteConfig("httpbin", "default", 8000),
	}, []*model.Service{
		newRDSTestService("nginx", "app", "nginx.app.svc.cluster.local", 80),
		newRDSTestService("httpbin", "default", "httpbin.default.svc.cluster.local", 8000),
	})

	rc := buildHTTPRoute(
		&model.Proxy{ID: "nginx-consumer.app", Type: model.Inherent},
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

func TestBuildHTTPRouteInherentOutboundUsesServiceAttachedHTTPRoute(t *testing.T) {
	push := newRDSTestPushContext(t, []config.Config{
		newServiceAttachedHTTPRouteConfig("reviews-routing", "moviereview", "reviews", 9080),
	}, []*model.Service{
		newRDSTestService("reviews", "moviereview", "reviews.moviereview.svc.cluster.local", 9080),
		newRDSTestService("reviews-v1", "moviereview", "reviews-v1.moviereview.svc.cluster.local", 9080),
		newRDSTestService("reviews-v2", "moviereview", "reviews-v2.moviereview.svc.cluster.local", 9080),
	})

	rc := buildHTTPRoute(
		&model.Proxy{ID: "moviepage.moviereview", Type: model.Inherent},
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
		&model.Proxy{ID: "moviepage.moviereview", Type: model.Inherent},
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

func TestBuildHTTPRouteOmitsGatewayAPIRetryPolicy(t *testing.T) {
	cfg := newServiceAttachedHTTPRouteConfig("reviews-retry", "moviereview", "reviews", 9080)
	spec := cfg.Spec.(*gatewayv1.HTTPRouteSpec)
	spec.Rules[0].Timeouts = &gatewayv1.HTTPRouteTimeouts{
		Request:        ptrTo(gatewayv1.Duration("2s")),
		BackendRequest: ptrTo(gatewayv1.Duration("250ms")),
	}
	spec.Rules[0].Retry = &gatewayv1.HTTPRouteRetry{
		Codes:    []gatewayv1.HTTPRouteRetryStatusCode{500, 503},
		Attempts: ptrTo(3),
		Backoff:  ptrTo(gatewayv1.Duration("100ms")),
	}
	push := newRDSTestPushContext(t, []config.Config{cfg}, []*model.Service{
		newRDSTestService("reviews", "moviereview", "reviews.moviereview.svc.cluster.local", 9080),
		newRDSTestService("reviews-v1", "moviereview", "reviews-v1.moviereview.svc.cluster.local", 9080),
		newRDSTestService("reviews-v2", "moviereview", "reviews-v2.moviereview.svc.cluster.local", 9080),
	})

	rc := buildHTTPRoute(
		&model.Proxy{ID: "moviepage.moviereview", Type: model.Inherent},
		push,
		"outbound|9080||reviews.moviereview.svc.cluster.local",
	)
	if rc == nil {
		t.Fatal("buildHTTPRoute() returned nil")
	}
	for i, r := range rc.VirtualHosts[0].Routes {
		if r.GetRoute().GetRetryPolicy() != nil {
			t.Fatalf("route[%d] retry policy = %v, want nil", i, r.GetRoute().GetRetryPolicy())
		}
		if r.GetRoute().GetFaultPolicy() != nil {
			t.Fatalf("route[%d] fault policy = %v, want nil", i, r.GetRoute().GetFaultPolicy())
		}
	}
}

func TestBuildHTTPRouteOmitsServiceFaultInjectionPolicy(t *testing.T) {
	routeConfig := newServiceAttachedHTTPRouteConfig("reviews-fault", "moviereview", "reviews", 9080)
	faultConfig := config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.FaultInjectionPolicy,
			Name:             "reviews-fault",
			Namespace:        "moviereview",
		},
		Spec: &networking.FaultInjectionPolicy{
			TargetRefs: []*networking.PolicyTargetReference{{
				Kind:        "Service",
				Name:        "reviews",
				SectionName: "http",
			}},
			Delay: &networking.FaultDelay{
				FixedDelay: durationpb.New(250 * time.Millisecond),
				Percentage: wrapperspb.UInt32(20),
			},
			Abort: &networking.FaultAbort{
				HttpStatus: 503,
				Percentage: wrapperspb.UInt32(10),
			},
		},
	}
	push := newRDSTestPushContext(t, []config.Config{routeConfig, faultConfig}, []*model.Service{
		newRDSTestService("reviews", "moviereview", "reviews.moviereview.svc.cluster.local", 9080),
		newRDSTestService("reviews-v1", "moviereview", "reviews-v1.moviereview.svc.cluster.local", 9080),
		newRDSTestService("reviews-v2", "moviereview", "reviews-v2.moviereview.svc.cluster.local", 9080),
	})

	rc := buildHTTPRoute(
		&model.Proxy{ID: "moviepage.moviereview", Type: model.Inherent},
		push,
		"outbound|9080||reviews.moviereview.svc.cluster.local",
	)
	if rc == nil {
		t.Fatal("buildHTTPRoute() returned nil")
	}
	for i, r := range rc.VirtualHosts[0].Routes {
		if r.GetRoute().GetFaultPolicy() != nil {
			t.Fatalf("route[%d] fault policy = %v, want nil", i, r.GetRoute().GetFaultPolicy())
		}
	}
}

func TestBuildHTTPRouteOmitsRegexMatchersAndLowercasesHeaders(t *testing.T) {
	regexPath := gatewayv1.PathMatchRegularExpression
	regexHeader := gatewayv1.HeaderMatchRegularExpression
	prefixPath := gatewayv1.PathMatchPathPrefix
	exactHeader := gatewayv1.HeaderMatchExact
	regexPathVal := "^/v[0-9]+/.*"
	prefixPathVal := "/ok"

	cfg := newServiceAttachedHTTPRouteConfig("reviews-match", "moviereview", "reviews", 9080)
	spec := cfg.Spec.(*gatewayv1.HTTPRouteSpec)
	spec.Rules[0].Matches = []gatewayv1.HTTPRouteMatch{
		{
			Path: &gatewayv1.HTTPPathMatch{Type: &regexPath, Value: &regexPathVal},
		},
		{
			Path: &gatewayv1.HTTPPathMatch{Type: &prefixPath, Value: &prefixPathVal},
			Headers: []gatewayv1.HTTPHeaderMatch{{
				Type:  &regexHeader,
				Name:  "X-Canary-Regex",
				Value: "^true$",
			}},
		},
		{
			Path: &gatewayv1.HTTPPathMatch{Type: &prefixPath, Value: &prefixPathVal},
			Headers: []gatewayv1.HTTPHeaderMatch{{
				Type:  &exactHeader,
				Name:  "X-End-User",
				Value: "Jason",
			}},
		},
	}

	push := newRDSTestPushContext(t, []config.Config{cfg}, []*model.Service{
		newRDSTestService("reviews", "moviereview", "reviews.moviereview.svc.cluster.local", 9080),
		newRDSTestService("reviews-v1", "moviereview", "reviews-v1.moviereview.svc.cluster.local", 9080),
		newRDSTestService("reviews-v2", "moviereview", "reviews-v2.moviereview.svc.cluster.local", 9080),
	})

	rc := buildHTTPRoute(
		&model.Proxy{ID: "moviepage.moviereview", Type: model.Inherent},
		push,
		"outbound|9080||reviews.moviereview.svc.cluster.local",
	)
	if rc == nil {
		t.Fatal("buildHTTPRoute() returned nil")
	}
	routes := rc.VirtualHosts[0].Routes
	if len(routes) != 2 {
		t.Fatalf("routes = %d, want 2 (1 valid match + 1 fallback)", len(routes))
	}
	for i, r := range routes {
		if r.GetMatch().GetSafeRegex() != nil {
			t.Fatalf("route[%d] emitted SafeRegex path matcher", i)
		}
		for _, h := range r.GetMatch().GetHeaders() {
			if h.GetSafeRegexMatch() != nil {
				t.Fatalf("route[%d] emitted SafeRegexMatch header matcher", i)
			}
		}
	}
	headers := routes[0].GetMatch().GetHeaders()
	if len(headers) != 1 || headers[0].GetName() != "x-end-user" || headers[0].GetExactMatch() != "Jason" {
		t.Fatalf("first route headers = %v, want lowercase x-end-user exact Jason", headers)
	}
}

func assertRouteConfigurationSnapshotContract(t *testing.T, rc *route.RouteConfiguration) {
	t.Helper()
	if rc == nil {
		t.Fatal("RouteConfiguration is nil")
	}
	if rc.GetAgentConfig() != nil {
		t.Fatalf("agent_config = %v, want nil", rc.GetAgentConfig())
	}

	seenDomains := map[string]struct{}{}
	for vhIdx, vh := range rc.GetVirtualHosts() {
		if len(vh.GetDomains()) == 0 {
			t.Fatalf("virtual_host[%d] domains is empty", vhIdx)
		}
		for _, d := range vh.GetDomains() {
			if d == "" {
				t.Fatalf("virtual_host[%d] contains empty domain", vhIdx)
			}
			for _, r := range d {
				if unicode.IsSpace(r) {
					t.Fatalf("virtual_host[%d] domain %q contains whitespace", vhIdx, d)
				}
			}
			validWildcard := !strings.Contains(d, "*") || d == "*" || (strings.Count(d, "*") == 1 && (strings.HasPrefix(d, "*.") || strings.HasSuffix(d, ".*")))
			if !validWildcard {
				t.Fatalf("virtual_host[%d] invalid wildcard domain %q", vhIdx, d)
			}
			lower := strings.ToLower(d)
			if _, exists := seenDomains[lower]; exists {
				t.Fatalf("duplicate virtual host domain %q", d)
			}
			seenDomains[lower] = struct{}{}
		}

		for rIdx, r := range vh.GetRoutes() {
			m := r.GetMatch()
			if m == nil {
				t.Fatalf("vh[%d].route[%d] missing match", vhIdx, rIdx)
			}
			switch spec := m.GetPathSpecifier().(type) {
			case *route.RouteMatch_Path:
				if !strings.HasPrefix(spec.Path, "/") {
					t.Fatalf("vh[%d].route[%d] exact path %q must start with '/'", vhIdx, rIdx, spec.Path)
				}
			case *route.RouteMatch_Prefix:
				if !strings.HasPrefix(spec.Prefix, "/") {
					t.Fatalf("vh[%d].route[%d] prefix %q must start with '/'", vhIdx, rIdx, spec.Prefix)
				}
			default:
				t.Fatalf("vh[%d].route[%d] unsupported path_specifier %T", vhIdx, rIdx, m.GetPathSpecifier())
			}

			for hIdx, h := range m.GetHeaders() {
				if h.GetName() == "" || h.GetName() != strings.ToLower(h.GetName()) {
					t.Fatalf("vh[%d].route[%d].header[%d] name = %q, want non-empty lowercase", vhIdx, rIdx, hIdx, h.GetName())
				}
				switch h.GetHeaderMatchSpecifier().(type) {
				case *route.HeaderMatcher_ExactMatch, *route.HeaderMatcher_PrefixMatch:
				default:
					t.Fatalf("vh[%d].route[%d].header[%d] unsupported specifier %T", vhIdx, rIdx, hIdx, h.GetHeaderMatchSpecifier())
				}
			}

			switch act := r.GetAction().(type) {
			case *route.Route_Route:
				if act.Route == nil {
					t.Fatalf("vh[%d].route[%d] Route action is nil", vhIdx, rIdx)
				}
				if act.Route.GetRetryPolicy() != nil {
					t.Fatalf("vh[%d].route[%d] retry_policy = %v, want nil", vhIdx, rIdx, act.Route.GetRetryPolicy())
				}
				if act.Route.GetFaultPolicy() != nil {
					t.Fatalf("vh[%d].route[%d] fault_policy = %v, want nil", vhIdx, rIdx, act.Route.GetFaultPolicy())
				}
				if timeout := act.Route.GetTimeout(); timeout != nil {
					if timeout.GetSeconds() < 0 || timeout.GetNanos() < 0 || timeout.GetNanos() >= 1_000_000_000 {
						t.Fatalf("vh[%d].route[%d] invalid timeout %v", vhIdx, rIdx, timeout)
					}
				}
				switch cs := act.Route.GetClusterSpecifier().(type) {
				case *route.RouteAction_Cluster:
					if cs.Cluster == "" {
						t.Fatalf("vh[%d].route[%d] empty cluster name", vhIdx, rIdx)
					}
				case *route.RouteAction_WeightedClusters:
					if cs.WeightedClusters == nil || len(cs.WeightedClusters.GetClusters()) == 0 {
						t.Fatalf("vh[%d].route[%d] empty weighted_clusters", vhIdx, rIdx)
					}
					seenClusters := map[string]struct{}{}
					var sum uint64
					for _, wc := range cs.WeightedClusters.GetClusters() {
						if wc.GetClusterHeader() != "" || wc.GetName() == "" {
							t.Fatalf("vh[%d].route[%d] invalid weighted cluster %v", vhIdx, rIdx, wc)
						}
						if _, dup := seenClusters[wc.GetName()]; dup {
							t.Fatalf("vh[%d].route[%d] duplicate weighted cluster %q", vhIdx, rIdx, wc.GetName())
						}
						seenClusters[wc.GetName()] = struct{}{}
						if wc.GetWeight() == nil {
							t.Fatalf("vh[%d].route[%d] weighted cluster %q missing weight", vhIdx, rIdx, wc.GetName())
						}
						sum += uint64(wc.GetWeight().GetValue())
					}
					if sum == 0 {
						t.Fatalf("vh[%d].route[%d] zero total weighted cluster weight", vhIdx, rIdx)
					}
					if tw := cs.WeightedClusters.GetTotalWeight(); tw != nil && uint64(tw.GetValue()) != sum {
						t.Fatalf("vh[%d].route[%d] total_weight = %d, sum = %d", vhIdx, rIdx, tw.GetValue(), sum)
					}
				default:
					t.Fatalf("vh[%d].route[%d] unsupported cluster_specifier %T", vhIdx, rIdx, act.Route.GetClusterSpecifier())
				}
			case *route.Route_NonForwardingAction:
			default:
				t.Fatalf("vh[%d].route[%d] unsupported action %T", vhIdx, rIdx, r.GetAction())
			}
		}
	}
}

func TestBuildHTTPRouteSnapshotValidationContract(t *testing.T) {
	exactPath := gatewayv1.PathMatchExact
	exactHeader := gatewayv1.HeaderMatchExact
	cfg := newServiceAttachedHTTPRouteConfig("reviews-contract", "moviereview", "reviews", 9080)
	spec := cfg.Spec.(*gatewayv1.HTTPRouteSpec)
	spec.Rules[0].Timeouts = &gatewayv1.HTTPRouteTimeouts{
		Request: ptrTo(gatewayv1.Duration("500ms")),
	}
	v1Port := gatewayv1.PortNumber(9080)
	spec.Rules[0].BackendRefs = append(spec.Rules[0].BackendRefs, gatewayv1.HTTPBackendRef{
		BackendRef: gatewayv1.BackendRef{
			BackendObjectReference: gatewayv1.BackendObjectReference{
				Name: "reviews-v1",
				Port: &v1Port,
			},
			Weight: ptrTo[int32](25),
		},
	})
	spec.Rules[0].Matches = []gatewayv1.HTTPRouteMatch{
		{
			Path: &gatewayv1.HTTPPathMatch{Type: &exactPath, Value: ptrTo("/reviews/detail")},
			Headers: []gatewayv1.HTTPHeaderMatch{{
				Type:  &exactHeader,
				Name:  "X-User-Tier",
				Value: "vip",
			}},
		},
	}

	push := newRDSTestPushContext(t, []config.Config{cfg}, []*model.Service{
		newRDSTestService("reviews", "moviereview", "reviews.moviereview.svc.cluster.local", 9080),
		newRDSTestService("reviews-v1", "moviereview", "reviews-v1.moviereview.svc.cluster.local", 9080),
		newRDSTestService("reviews-v2", "moviereview", "reviews-v2.moviereview.svc.cluster.local", 9080),
	})

	rc := buildHTTPRoute(
		&model.Proxy{ID: "moviepage.moviereview", Type: model.Inherent},
		push,
		"outbound|9080||reviews.moviereview.svc.cluster.local",
	)
	assertRouteConfigurationSnapshotContract(t, rc)

	defaultPush := newRDSTestPushContext(t, nil, []*model.Service{
		newRDSTestService("reviews", "moviereview", "reviews.moviereview.svc.cluster.local", 9080),
	})
	defaultRC := buildHTTPRoute(
		&model.Proxy{ID: "moviepage.moviereview", Type: model.Inherent},
		defaultPush,
		"outbound|9080||reviews.moviereview.svc.cluster.local",
	)
	assertRouteConfigurationSnapshotContract(t, defaultRC)
}

func TestBuildHTTPRouteSanitizesSnapshotEdgeCases(t *testing.T) {
	t.Run("wildcard host omits invalid short wildcard and duplicate star", func(t *testing.T) {
		push := newRDSTestPushContext(t, nil, []*model.Service{
			newRDSTestService("ext", "default", "*.example.com", 80),
		})
		rc := buildHTTPRoute(
			&model.Proxy{ID: "caller.default", Type: model.Inherent},
			push,
			"outbound|80||*.example.com",
		)
		assertRouteConfigurationSnapshotContract(t, rc)
		wantDomains := []string{"*.example.com:80", "*.example.com", "*"}
		if !reflect.DeepEqual(rc.VirtualHosts[0].GetDomains(), wantDomains) {
			t.Fatalf("domains = %v, want %v", rc.VirtualHosts[0].GetDomains(), wantDomains)
		}
	})

	t.Run("nil path defaults to slash, relative path dropped, duplicate backends merged, negative timeout omitted", func(t *testing.T) {
		prefixPath := gatewayv1.PathMatchPathPrefix
		exactPath := gatewayv1.PathMatchExact
		cfg := newServiceAttachedHTTPRouteConfig("reviews-edge", "moviereview", "reviews", 9080)
		spec := cfg.Spec.(*gatewayv1.HTTPRouteSpec)
		spec.Rules[0].Timeouts = &gatewayv1.HTTPRouteTimeouts{
			Request: ptrTo(gatewayv1.Duration("-5s")),
		}
		spec.Rules[0].Matches = []gatewayv1.HTTPRouteMatch{
			{Path: &gatewayv1.HTTPPathMatch{Type: &prefixPath, Value: nil}},
			{Path: &gatewayv1.HTTPPathMatch{Type: &exactPath, Value: ptrTo("no-leading-slash")}},
		}
		v1Port := gatewayv1.PortNumber(9080)
		spec.Rules[0].BackendRefs = append(spec.Rules[0].BackendRefs,
			gatewayv1.HTTPBackendRef{
				BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Name: "reviews-v2",
						Port: &v1Port,
					},
					Weight: ptrTo[int32](20),
				},
			},
			gatewayv1.HTTPBackendRef{
				BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Name: "reviews-v1",
						Port: &v1Port,
					},
					Weight: ptrTo[int32](15),
				},
			},
		)

		push := newRDSTestPushContext(t, []config.Config{cfg}, []*model.Service{
			newRDSTestService("reviews", "moviereview", "reviews.moviereview.svc.cluster.local", 9080),
			newRDSTestService("reviews-v1", "moviereview", "reviews-v1.moviereview.svc.cluster.local", 9080),
			newRDSTestService("reviews-v2", "moviereview", "reviews-v2.moviereview.svc.cluster.local", 9080),
		})

		rc := buildHTTPRoute(
			&model.Proxy{ID: "moviepage.moviereview", Type: model.Inherent},
			push,
			"outbound|9080||reviews.moviereview.svc.cluster.local",
		)
		assertRouteConfigurationSnapshotContract(t, rc)
		routes := rc.VirtualHosts[0].GetRoutes()
		if len(routes) != 2 {
			t.Fatalf("routes = %d, want 2 (1 sanitized rule[0] + 1 fallback rule[1])", len(routes))
		}
		if routes[0].GetMatch().GetPrefix() != "/" {
			t.Fatalf("prefix = %q, want /", routes[0].GetMatch().GetPrefix())
		}
		if routes[0].GetRoute().GetTimeout() != nil {
			t.Fatalf("negative timeout was not omitted: %v", routes[0].GetRoute().GetTimeout())
		}
		gotWeights := weightedClustersByName(t, routes[0])
		wantWeights := map[string]uint32{
			"outbound|9080||reviews-v1.moviereview.svc.cluster.local": 15,
			"outbound|9080||reviews-v2.moviereview.svc.cluster.local": 120,
		}
		if !reflect.DeepEqual(gotWeights, wantWeights) {
			t.Fatalf("coalesced weights = %v, want %v", gotWeights, wantWeights)
		}
	})

	t.Run("inbound non-forwarding route satisfies contract", func(t *testing.T) {
		push := newRDSTestPushContext(t, nil, nil)
		rc := buildHTTPRoute(
			&model.Proxy{ID: "moviepage.moviereview", Type: model.Inherent},
			push,
			"8080",
		)
		assertRouteConfigurationSnapshotContract(t, rc)
	})
}

func newRDSTestEnv(t *testing.T, configs []config.Config, services []*model.Service) *model.Environment {
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
	return env
}

func newRDSTestPushContext(t *testing.T, configs []config.Config, services []*model.Service) *model.PushContext {
	t.Helper()

	env := newRDSTestEnv(t, configs, services)
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
