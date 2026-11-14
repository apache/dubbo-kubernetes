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
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/wellknown"
	"github.com/apache/dubbo-kubernetes/pkg/xdsbootstrap"
	security "github.com/dubml/api/security/v1alpha3"
	typev1alpha3 "github.com/dubml/api/type/v1alpha3"
	core "github.com/dubml/xds-api/core/v1"
	routerv1 "github.com/dubml/xds-api/extensions/filters/v1/http/router"
	hcmv1 "github.com/dubml/xds-api/extensions/filters/v1/network/http_connection_manager"
	listener "github.com/dubml/xds-api/listener/v1"
)

func TestBuildListenersWildcardAndExplicitReturnAPIListenerOnly(t *testing.T) {
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
		newPeerAuthenticationConfig("app-strict-mtls", "app", security.PeerAuthentication_MutualTLS_STRICT),
		{
			Meta: config.Meta{
				GroupVersionKind: gvk.RequestAuthentication,
				Name:             "jwt-example",
				Namespace:        "app",
			},
			Spec: &security.RequestAuthentication{
				Selector: &typev1alpha3.WorkloadSelector{MatchLabels: map[string]string{"app": "nginx"}},
				JwtRules: []*security.JWTRule{{
					Issuer:  "testing@secure.dubbo.apache.org",
					JwksUri: "https://secure.dubbo.apache.org/jwt/samples/jwks.json",
				}},
			},
		},
		{
			Meta: config.Meta{
				GroupVersionKind: gvk.AuthorizationPolicy,
				Name:             "allow-example",
				Namespace:        "app",
			},
			Spec: &security.AuthorizationPolicy{
				Action: security.AuthorizationPolicy_ALLOW,
			},
		},
	}, []*model.Service{svc})

	assertTripleListener := func(t *testing.T, resources model.Resources, wantName string) {
		t.Helper()
		if len(resources) != 1 {
			t.Fatalf("resources = %d, want 1", len(resources))
		}
		l := findListener(t, resources, wantName)
		if len(l.GetFilterChains()) != 0 {
			t.Fatalf("filter_chains = %d, want 0", len(l.GetFilterChains()))
		}
		if l.GetApiListener() == nil || l.GetApiListener().GetApiListener() == nil {
			t.Fatalf("listener %s missing api_listener", wantName)
		}
		hcm := &hcmv1.HttpConnectionManager{}
		if err := l.GetApiListener().GetApiListener().UnmarshalTo(hcm); err != nil {
			t.Fatalf("unmarshal api_listener hcm: %v", err)
		}
		filters := hcm.GetHttpFilters()
		if len(filters) != 1 || filters[0].GetName() != wellknown.HTTPRouter {
			t.Fatalf("http_filters = %v, want only %q", filters, wellknown.HTTPRouter)
		}
	}

	const outboundName = "nginx.app.svc.cluster.local:80"
	wildcardResources := (&TripleConfigGenerator{}).BuildListeners(proxy, push, nil)
	assertTripleListener(t, wildcardResources, outboundName)

	explicitResources := (&TripleConfigGenerator{}).BuildListeners(proxy, push, []string{outboundName})
	assertTripleListener(t, explicitResources, outboundName)

	inboundResources := (&TripleConfigGenerator{}).BuildListeners(proxy, push, []string{xdsbootstrap.ServerListenerNamePrefix + "0.0.0.0:80"})
	if len(inboundResources) != 0 {
		t.Fatalf("inbound request returned %d listeners, want 0", len(inboundResources))
	}
}

func TestBuildListenersSnapshotValidationContract(t *testing.T) {
	svc := newRDSTestService("nginx", "app", "nginx.app.svc.cluster.local", 80)
	proxy := newMTLSTestProxy()
	push := newRDSTestPushContext(t, []config.Config{
		newPeerAuthenticationConfig("app-strict-mtls", "app", security.PeerAuthentication_MutualTLS_STRICT),
	}, []*model.Service{svc})

	const listenerName = "nginx.app.svc.cluster.local:80"
	const wantRouteName = "outbound|80||nginx.app.svc.cluster.local"

	for _, tc := range []struct {
		name      string
		requested []string
	}{
		{name: "wildcard", requested: nil},
		{name: "explicit", requested: []string{listenerName}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resources := (&TripleConfigGenerator{}).BuildListeners(proxy, push, tc.requested)
			if len(resources) != 1 {
				t.Fatalf("resources = %d, want 1", len(resources))
			}
			l := findListener(t, resources, listenerName)
			if len(l.GetFilterChains()) != 0 {
				t.Fatalf("filter_chains = %d, want 0", len(l.GetFilterChains()))
			}
			if len(l.GetListenerFilters()) != 0 {
				t.Fatalf("listener_filters = %d, want 0", len(l.GetListenerFilters()))
			}
			if l.GetUseOriginalDst().GetValue() {
				t.Fatalf("use_original_dst = true, want false/unset")
			}

			apiAny := l.GetApiListener().GetApiListener()
			if apiAny == nil {
				t.Fatal("api_listener is nil")
			}
			const wantHCMTypeURL = "type.googleapis.com/extensions.filters.network.http_connection_manager.v1.HttpConnectionManager"
			if apiAny.GetTypeUrl() != wantHCMTypeURL {
				t.Fatalf("api_listener type_url = %q, want %q", apiAny.GetTypeUrl(), wantHCMTypeURL)
			}

			hcm := &hcmv1.HttpConnectionManager{}
			if err := apiAny.UnmarshalTo(hcm); err != nil {
				t.Fatalf("unmarshal HttpConnectionManager: %v", err)
			}
			if hcm.GetCodecType() != hcmv1.HttpConnectionManager_AUTO {
				t.Fatalf("hcm.codec_type = %v, want AUTO (0)", hcm.GetCodecType())
			}

			const wantRouterTypeURL = "type.googleapis.com/extensions.filters.http.router.v1.Router"
			filters := hcm.GetHttpFilters()
			if len(filters) != 1 || filters[0].GetName() != wellknown.HTTPRouter {
				t.Fatalf("http_filters = %v, want single %q filter", filters, wellknown.HTTPRouter)
			}
			filterAny := filters[0].GetTypedConfig()
			if filterAny == nil || filterAny.GetTypeUrl() != wantRouterTypeURL {
				t.Fatalf("http_filter[0] typed_config = %v, want type_url %q", filterAny, wantRouterTypeURL)
			}
			if err := filterAny.UnmarshalTo(&routerv1.Router{}); err != nil {
				t.Fatalf("unmarshal Router filter: %v", err)
			}

			rdsSpec, ok := hcm.GetRouteSpecifier().(*hcmv1.HttpConnectionManager_Rds)
			if !ok || rdsSpec.Rds == nil {
				t.Fatalf("route_specifier = %T, want HttpConnectionManager_Rds", hcm.GetRouteSpecifier())
			}
			if rdsSpec.Rds.GetRouteConfigName() != wantRouteName {
				t.Fatalf("rds.route_config_name = %q, want %q", rdsSpec.Rds.GetRouteConfigName(), wantRouteName)
			}
			if _, ok := rdsSpec.Rds.GetConfigSource().GetConfigSourceSpecifier().(*core.ConfigSource_Ads); !ok {
				t.Fatalf("rds.config_source = %v, want ADS config source", rdsSpec.Rds.GetConfigSource())
			}
		})
	}
}

func newMTLSTestProxy() *model.Proxy {
	return &model.Proxy{
		ID:              "inherent~10.0.0.1~nginx-consumer.app~app.svc.cluster.local",
		Type:            model.Inherent,
		DNSDomain:       "app.svc.cluster.local",
		ConfigNamespace: "app",
		Metadata: &model.NodeMetadata{
			Generator: "triple",
			Namespace: "app",
		},
	}
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
