package model

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	networking "github.com/kdubbo/api/networking/v1alpha3"
)

func TestMeshServiceToVirtualServiceConfigBuildsServiceWeightedRoute(t *testing.T) {
	cfg := newModelTestMeshService("reviews-routing", "default", "reviews.default.svc.cluster.local", []*networking.MeshServiceRoute{
		{Service: []*networking.ServiceDestination{{
			Name:   "reviews-1",
			Host:   "reviews.default.svc.cluster.local",
			Weight: 20,
		}}},
		{Service: []*networking.ServiceDestination{{
			Name:   "reviews-2",
			Host:   "reviews.default.svc.cluster.local",
			Weight: 80,
		}}},
	})

	got := meshServiceToVirtualServiceConfig(cfg).Spec.(*networking.VirtualService)
	if len(got.Http) != 1 {
		t.Fatalf("http routes = %d, want 1", len(got.Http))
	}
	if len(got.Http[0].Route) != 2 {
		t.Fatalf("weighted destinations = %d, want 2", len(got.Http[0].Route))
	}

	assertHTTPDestination(t, got.Http[0].Route[0], "reviews-1.default.svc.cluster.local", "", 20)
	assertHTTPDestination(t, got.Http[0].Route[1], "reviews-2.default.svc.cluster.local", "", 80)
}

func TestMeshServiceToVirtualServiceConfigKeepsLabelSubsetRoute(t *testing.T) {
	cfg := newModelTestMeshService("reviews-routing", "default", "reviews.default.svc.cluster.local", []*networking.MeshServiceRoute{{
		Service: []*networking.ServiceDestination{
			{
				Name:   "v1",
				Host:   "reviews.default.svc.cluster.local",
				Labels: map[string]string{"version": "v1"},
				Weight: 20,
			},
			{
				Name:   "v2",
				Host:   "reviews.default.svc.cluster.local",
				Labels: map[string]string{"version": "v2"},
				Weight: 80,
			},
		},
	}})

	got := meshServiceToVirtualServiceConfig(cfg).Spec.(*networking.VirtualService)
	if len(got.Http) != 1 || len(got.Http[0].Route) != 2 {
		t.Fatalf("http routes = %v, want one route with two destinations", got.Http)
	}
	assertHTTPDestination(t, got.Http[0].Route[0], "reviews.default.svc.cluster.local", "v1", 20)
	assertHTTPDestination(t, got.Http[0].Route[1], "reviews.default.svc.cluster.local", "v2", 80)

	rules := meshServiceToDestinationRuleConfigs(cfg)
	if len(rules) != 1 {
		t.Fatalf("destination rules = %d, want 1", len(rules))
	}
	dr := rules[0].Spec.(*networking.DestinationRule)
	if dr.Host != "reviews.default.svc.cluster.local" {
		t.Fatalf("destination rule host = %q, want reviews.default.svc.cluster.local", dr.Host)
	}
	if len(dr.Subsets) != 2 {
		t.Fatalf("subsets = %d, want 2", len(dr.Subsets))
	}
	if got := dr.Subsets[1].Labels["version"]; got != "v2" {
		t.Fatalf("subset label = %q, want v2", got)
	}
}

func TestMeshServiceToDestinationRuleConfigsBuildsServiceTrafficPolicies(t *testing.T) {
	cfg := newModelTestMeshService("reviews-routing", "default", "reviews.default.svc.cluster.local", []*networking.MeshServiceRoute{{
		Service: []*networking.ServiceDestination{
			{Name: "reviews-1", Host: "reviews.default.svc.cluster.local", Weight: 20},
			{Name: "reviews-2", Host: "reviews.default.svc.cluster.local", Weight: 80},
		},
	}})
	cfg.Spec.(*networking.MeshService).TrafficPolicy = &networking.TrafficPolicy{
		Tls: &networking.ClientTLSSettings{Mode: networking.ClientTLSSettings_DUBBO_MUTUAL},
	}

	rules := meshServiceToDestinationRuleConfigs(cfg)
	if len(rules) != 2 {
		t.Fatalf("destination rules = %d, want 2", len(rules))
	}
	for i, wantHost := range []string{"reviews-1.default.svc.cluster.local", "reviews-2.default.svc.cluster.local"} {
		dr := rules[i].Spec.(*networking.DestinationRule)
		if dr.Host != wantHost {
			t.Fatalf("destination rule[%d] host = %q, want %q", i, dr.Host, wantHost)
		}
		if len(dr.Subsets) != 0 {
			t.Fatalf("destination rule[%d] subsets = %d, want 0", i, len(dr.Subsets))
		}
		if got := dr.GetTrafficPolicy().GetTls().GetMode(); got != networking.ClientTLSSettings_DUBBO_MUTUAL {
			t.Fatalf("destination rule[%d] tls mode = %v, want DUBBO_MUTUAL", i, got)
		}
	}
}

func newModelTestMeshService(name, namespace, hostname string, routes []*networking.MeshServiceRoute) config.Config {
	return config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.MeshService,
			Name:             name,
			Namespace:        namespace,
			Domain:           "cluster.local",
		},
		Spec: &networking.MeshService{
			Hosts:  []string{hostname},
			Routes: routes,
		},
	}
}

func assertHTTPDestination(t *testing.T, got *networking.HTTPRouteDestination, wantHost, wantSubset string, wantWeight int32) {
	t.Helper()
	if got.GetDestination().GetHost() != wantHost {
		t.Fatalf("destination host = %q, want %q", got.GetDestination().GetHost(), wantHost)
	}
	if got.GetDestination().GetSubset() != wantSubset {
		t.Fatalf("destination subset = %q, want %q", got.GetDestination().GetSubset(), wantSubset)
	}
	if got.GetWeight() != wantWeight {
		t.Fatalf("destination weight = %d, want %d", got.GetWeight(), wantWeight)
	}
}
