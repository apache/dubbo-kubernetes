//
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

package validation

import (
	"testing"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	networking "github.com/kdubbo/api/networking/v1alpha3"
	security "github.com/kdubbo/api/security/v1alpha3"
	telemetry "github.com/kdubbo/api/telemetry/v1alpha1"
	typev1alpha3 "github.com/kdubbo/api/type/v1alpha3"
)

func makeConfig(spec config.Spec) config.Config {
	return config.Config{
		Meta: config.Meta{Name: "test", Namespace: "default"},
		Spec: spec,
	}
}

func TestValidateAuthorizationPolicy(t *testing.T) {
	cases := []struct {
		name    string
		spec    *security.AuthorizationPolicy
		wantErr bool
	}{
		{
			name:    "empty allow policy",
			spec:    &security.AuthorizationPolicy{},
			wantErr: false,
		},
		{
			name: "valid rule",
			spec: &security.AuthorizationPolicy{
				Action: security.AuthorizationPolicy_ALLOW,
				Rules: []*security.Rule{{
					From: []*security.From{{
						Source: &security.Source{RequestPrincipals: []string{"issuer/subject"}},
					}},
				}},
			},
			wantErr: false,
		},
		{
			name: "empty deny policy",
			spec: &security.AuthorizationPolicy{
				Action: security.AuthorizationPolicy_DENY,
			},
			wantErr: true,
		},
		{
			name: "from without source",
			spec: &security.AuthorizationPolicy{
				Rules: []*security.Rule{{
					From: []*security.From{{}},
				}},
			},
			wantErr: true,
		},
		{
			name: "empty request principal",
			spec: &security.AuthorizationPolicy{
				Rules: []*security.Rule{{
					From: []*security.From{{
						Source: &security.Source{RequestPrincipals: []string{""}},
					}},
				}},
			},
			wantErr: true,
		},
		{
			name: "when without values",
			spec: &security.AuthorizationPolicy{
				Rules: []*security.Rule{{
					When: []*security.Condition{{Key: "request.auth.claims[iss]"}},
				}},
			},
			wantErr: true,
		},
		{
			name: "empty selector matchLabels",
			spec: &security.AuthorizationPolicy{
				Selector: &typev1alpha3.WorkloadSelector{},
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateAuthorizationPolicy(makeConfig(tc.spec))
			if (err != nil) != tc.wantErr {
				t.Fatalf("got err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestValidatePeerAuthentication(t *testing.T) {
	cases := []struct {
		name    string
		spec    *security.PeerAuthentication
		wantErr bool
	}{
		{
			name:    "empty",
			spec:    &security.PeerAuthentication{},
			wantErr: false,
		},
		{
			name: "strict",
			spec: &security.PeerAuthentication{
				Mtls: &security.PeerAuthentication_MutualTLS{Mode: security.PeerAuthentication_MutualTLS_STRICT},
			},
			wantErr: false,
		},
		{
			name: "port level without selector",
			spec: &security.PeerAuthentication{
				PortLevelMtls: map[uint32]*security.PeerAuthentication_MutualTLS{
					8080: {Mode: security.PeerAuthentication_MutualTLS_DISABLE},
				},
			},
			wantErr: true,
		},
		{
			name: "port level zero port",
			spec: &security.PeerAuthentication{
				Selector: &typev1alpha3.WorkloadSelector{MatchLabels: map[string]string{"app": "foo"}},
				PortLevelMtls: map[uint32]*security.PeerAuthentication_MutualTLS{
					0: {Mode: security.PeerAuthentication_MutualTLS_DISABLE},
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidatePeerAuthentication(makeConfig(tc.spec))
			if (err != nil) != tc.wantErr {
				t.Fatalf("got err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateRequestAuthentication(t *testing.T) {
	cases := []struct {
		name    string
		spec    *security.RequestAuthentication
		wantErr bool
	}{
		{
			name:    "empty",
			spec:    &security.RequestAuthentication{},
			wantErr: false,
		},
		{
			name: "valid",
			spec: &security.RequestAuthentication{
				JwtRules: []*security.JWTRule{{
					Issuer:  "https://issuer.example.com",
					JwksUri: "https://issuer.example.com/.well-known/jwks.json",
				}},
			},
			wantErr: false,
		},
		{
			name: "missing issuer",
			spec: &security.RequestAuthentication{
				JwtRules: []*security.JWTRule{{JwksUri: "https://issuer.example.com/jwks"}},
			},
			wantErr: true,
		},
		{
			name: "both jwks and jwksUri",
			spec: &security.RequestAuthentication{
				JwtRules: []*security.JWTRule{{
					Issuer:  "issuer",
					JwksUri: "https://issuer.example.com/jwks",
					Jwks:    "{}",
				}},
			},
			wantErr: true,
		},
		{
			name: "bad jwksUri scheme",
			spec: &security.RequestAuthentication{
				JwtRules: []*security.JWTRule{{
					Issuer:  "issuer",
					JwksUri: "ftp://issuer.example.com/jwks",
				}},
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateRequestAuthentication(makeConfig(tc.spec))
			if (err != nil) != tc.wantErr {
				t.Fatalf("got err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateCircuitBreakerPolicy(t *testing.T) {
	validRef := []*networking.PolicyTargetReference{{Kind: "Service", Name: "backend"}}
	cases := []struct {
		name    string
		spec    *networking.CircuitBreakerPolicy
		wantErr bool
	}{
		{
			name: "valid",
			spec: &networking.CircuitBreakerPolicy{
				TargetRefs:     validRef,
				ConnectionPool: &networking.ConnectionPoolSettings{MaxConnections: 100},
			},
			wantErr: false,
		},
		{
			name: "no target refs",
			spec: &networking.CircuitBreakerPolicy{
				ConnectionPool: &networking.ConnectionPoolSettings{MaxConnections: 100},
			},
			wantErr: true,
		},
		{
			name: "unsupported kind",
			spec: &networking.CircuitBreakerPolicy{
				TargetRefs:     []*networking.PolicyTargetReference{{Kind: "Gateway", Name: "gw"}},
				ConnectionPool: &networking.ConnectionPoolSettings{MaxConnections: 100},
			},
			wantErr: true,
		},
		{
			name:    "no settings",
			spec:    &networking.CircuitBreakerPolicy{TargetRefs: validRef},
			wantErr: true,
		},
		{
			name: "negative connections",
			spec: &networking.CircuitBreakerPolicy{
				TargetRefs:     validRef,
				ConnectionPool: &networking.ConnectionPoolSettings{MaxConnections: -1},
			},
			wantErr: true,
		},
		{
			name: "bad ejection percent",
			spec: &networking.CircuitBreakerPolicy{
				TargetRefs: validRef,
				OutlierDetection: &networking.OutlierDetection{
					MaxEjectionPercent: 150,
				},
			},
			wantErr: true,
		},
		{
			name: "negative interval",
			spec: &networking.CircuitBreakerPolicy{
				TargetRefs: validRef,
				OutlierDetection: &networking.OutlierDetection{
					Interval: durationpb.New(-1),
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateCircuitBreakerPolicy(makeConfig(tc.spec))
			if (err != nil) != tc.wantErr {
				t.Fatalf("got err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateServiceEntry(t *testing.T) {
	valid := func() *networking.ServiceEntry {
		return &networking.ServiceEntry{
			Hosts:      []string{"api.example.com"},
			Addresses:  []string{"240.0.0.1"},
			Resolution: networking.ServiceEntry_STATIC,
			Ports:      []*networking.ServicePort{{Name: "tcp", Number: 443, Protocol: "TCP"}},
			Endpoints:  []*networking.WorkloadEntry{{Address: "10.0.0.1", Ports: map[string]uint32{"tcp": 8443}}},
		}
	}
	cases := []struct {
		name    string
		mutate  func(*networking.ServiceEntry)
		wantErr bool
	}{
		{name: "valid", mutate: func(*networking.ServiceEntry) {}, wantErr: false},
		{name: "missing host", mutate: func(s *networking.ServiceEntry) { s.Hosts = nil }, wantErr: true},
		{name: "bad address", mutate: func(s *networking.ServiceEntry) { s.Addresses = []string{"not-an-ip"} }, wantErr: true},
		{name: "duplicate port", mutate: func(s *networking.ServiceEntry) {
			s.Ports = append(s.Ports, &networking.ServicePort{Name: "tcp", Number: 443, Protocol: "TCP"})
		}, wantErr: true},
		{name: "unsupported protocol", mutate: func(s *networking.ServiceEntry) { s.Ports[0].Protocol = "SMTP" }, wantErr: true},
		{name: "selector and endpoints", mutate: func(s *networking.ServiceEntry) {
			s.WorkloadSelector = &typev1alpha3.WorkloadSelector{MatchLabels: map[string]string{"app": "api"}}
		}, wantErr: true},
		{name: "invalid endpoint", mutate: func(s *networking.ServiceEntry) { s.Endpoints[0].Address = "" }, wantErr: true},
		{name: "invalid export", mutate: func(s *networking.ServiceEntry) { s.ExportTo = []string{"bad_namespace"} }, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := valid()
			tc.mutate(spec)
			_, err := ValidateServiceEntry(makeConfig(spec))
			if got := err != nil; got != tc.wantErr {
				t.Fatalf("ValidateServiceEntry() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateWorkloadEntry(t *testing.T) {
	cases := []struct {
		name    string
		spec    *networking.WorkloadEntry
		wantErr bool
	}{
		{name: "valid IP", spec: &networking.WorkloadEntry{Address: "10.0.0.1", Ports: map[string]uint32{"grpc": 50051}}, wantErr: false},
		{name: "valid DNS", spec: &networking.WorkloadEntry{Address: "vm.example.com"}, wantErr: false},
		{name: "missing address", spec: &networking.WorkloadEntry{}, wantErr: true},
		{name: "invalid port", spec: &networking.WorkloadEntry{Address: "10.0.0.1", Ports: map[string]uint32{"grpc": 0}}, wantErr: true},
		{name: "invalid labels", spec: &networking.WorkloadEntry{Address: "10.0.0.1", Labels: map[string]string{"bad key": "value"}}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateWorkloadEntry(makeConfig(tc.spec))
			if got := err != nil; got != tc.wantErr {
				t.Fatalf("ValidateWorkloadEntry() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateTelemetry(t *testing.T) {
	cases := []struct {
		name    string
		spec    *telemetry.Telemetry
		wantErr bool
	}{
		{
			name:    "empty",
			spec:    &telemetry.Telemetry{},
			wantErr: false,
		},
		{
			name: "valid tracing",
			spec: &telemetry.Telemetry{
				Tracing: []*telemetry.Tracing{{
					Providers:                []*telemetry.Tracing_TracingProvider{{Name: "localtrace"}},
					Tags:                     []*telemetry.Tracing_Tag{{Name: "foo", Value: "bar"}},
					RandomSamplingPercentage: wrapperspb.Double(50),
				}},
			},
			wantErr: false,
		},
		{
			name: "provider without name",
			spec: &telemetry.Telemetry{
				Tracing: []*telemetry.Tracing{{
					Providers: []*telemetry.Tracing_TracingProvider{{}},
				}},
			},
			wantErr: true,
		},
		{
			name: "sampling out of range",
			spec: &telemetry.Telemetry{
				Tracing: []*telemetry.Tracing{{
					Providers:                []*telemetry.Tracing_TracingProvider{{Name: "jaeger"}},
					RandomSamplingPercentage: wrapperspb.Double(101),
				}},
			},
			wantErr: true,
		},
		{
			name: "tag without name",
			spec: &telemetry.Telemetry{Tracing: []*telemetry.Tracing{{
				Tags: []*telemetry.Tracing_Tag{{Value: "bar"}},
			}}},
			wantErr: true,
		},
		{
			name: "duplicate tag name",
			spec: &telemetry.Telemetry{Tracing: []*telemetry.Tracing{{
				Tags: []*telemetry.Tracing_Tag{{Name: "foo"}, {Name: "foo"}},
			}}},
			wantErr: true,
		},
		{
			name: "selector with empty matchLabels value is allowed",
			spec: &telemetry.Telemetry{
				Selector: &typev1alpha3.WorkloadSelector{MatchLabels: map[string]string{"app": "demo"}},
			},
			wantErr: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateTelemetry(makeConfig(tc.spec))
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("ValidateTelemetry() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}

	meshlevel := makeConfig(&telemetry.Telemetry{
		Selector: &typev1alpha3.WorkloadSelector{MatchLabels: map[string]string{"app": "demo"}},
	})
	meshlevel.Namespace = "dubbo-system"
	if _, err := ValidateTelemetry(meshlevel); err == nil {
		t.Fatal("ValidateTelemetry() accepted selector on meshlevel Telemetry")
	}
}
