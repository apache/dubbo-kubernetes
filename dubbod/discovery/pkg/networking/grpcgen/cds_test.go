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
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	security "github.com/dubml/api/security/v1alpha3"
	cluster "github.com/dubml/xds-api/cluster/v1"
	tlsv1 "github.com/dubml/xds-api/extensions/transport_sockets/tls/v1"
)

func TestStrictPeerAuthenticationPinsBackendIdentity(t *testing.T) {
	hostName := host.Name("payment.app.svc.cluster.local")
	service := newRDSTestService("payment", "app", string(hostName), 8080)
	backendSAN := "spiffe://cluster.local/ns/app/sa/payment"
	service.ServiceAccounts = []string{backendSAN}
	push := newRDSTestPushContext(t, []config.Config{
		{
			Meta: config.Meta{
				GroupVersionKind: gvk.PeerAuthentication,
				Name:             "strict",
				Namespace:        "app",
			},
			Spec: &security.PeerAuthentication{
				Mtls: &security.PeerAuthentication_MutualTLS{
					Mode: security.PeerAuthentication_MutualTLS_STRICT,
				},
			},
		},
	}, []*model.Service{service})

	resources := (&GrpcConfigGenerator{}).BuildClusters(&model.Proxy{
		ID:              "inherent~10.0.0.2~caller.app~app.svc.cluster.local",
		Type:            model.Inherent,
		ConfigNamespace: "app",
	}, push, []string{"outbound|8080||" + string(hostName)})
	if len(resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(resources))
	}
	generated := &cluster.Cluster{}
	if err := resources[0].GetResource().UnmarshalTo(generated); err != nil {
		t.Fatalf("unmarshal cluster: %v", err)
	}
	if generated.GetTransportSocket() == nil {
		t.Fatal("STRICT PeerAuthentication did not produce an mTLS transport socket")
	}
	tlsContext := &tlsv1.UpstreamTlsContext{}
	if err := generated.GetTransportSocket().GetTypedConfig().UnmarshalTo(tlsContext); err != nil {
		t.Fatalf("unmarshal upstream TLS context: %v", err)
	}
	got := tlsContext.GetCommonTlsContext().
		GetCombinedValidationContext().
		GetDefaultValidationContext().
		GetMatchSubjectAltNames()
	if len(got) != 1 || got[0] != backendSAN {
		t.Fatalf("SAN pins = %v, want only the backend identity", got)
	}
}
