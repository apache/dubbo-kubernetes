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
	"os"
	"path/filepath"
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds/endpoints"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	cluster "github.com/dubml/xds-api/cluster/v1"
	endpoint "github.com/dubml/xds-api/endpoint/v1"
	listener "github.com/dubml/xds-api/listener/v1"
	route "github.com/dubml/xds-api/route/v1"
	security "github.com/dubml/api/security/v1alpha3"
	"google.golang.org/protobuf/proto"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestExportTripleSnapshotFixtures(t *testing.T) {
	fixtureDir := os.Getenv("TRIPLE_FIXTURE_DIR")
	if fixtureDir == "" {
		t.Skip("set TRIPLE_FIXTURE_DIR to export snapshot fixtures")
	}
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("create fixture directory %s: %v", fixtureDir, err)
	}

	const reviewsHost = "reviews.moviereview.svc.cluster.local"
	const v1Host = "reviews-v1.moviereview.svc.cluster.local"
	const v2Host = "reviews-v2.mtls.svc.cluster.local"

	reviewsSvc := newRDSTestService("reviews", "moviereview", reviewsHost, 9080)
	v1Svc := newRDSTestService("reviews-v1", "moviereview", v1Host, 9080)
	v2Svc := newRDSTestService("reviews-v2", "mtls", v2Host, 9080)
	v2Svc.ServiceAccounts = []string{"spiffe://cluster.local/ns/mtls/sa/reviews-v2"}

	routeCfg := newServiceAttachedHTTPRouteConfig("reviews-weighted", "moviereview", "reviews", 9080)
	routeSpec := routeCfg.Spec.(*gatewayv1.HTTPRouteSpec)
	mtlsNs := gatewayv1.Namespace("mtls")
	port9080 := gatewayv1.PortNumber(9080)
	routeSpec.Rules[0].BackendRefs = []gatewayv1.HTTPBackendRef{
		{
			BackendRef: gatewayv1.BackendRef{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Name: "reviews-v1",
					Port: &port9080,
				},
				Weight: ptrTo[int32](20),
			},
		},
		{
			BackendRef: gatewayv1.BackendRef{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Namespace: &mtlsNs,
					Name:      "reviews-v2",
					Port:      &port9080,
				},
				Weight: ptrTo[int32](80),
			},
		},
	}

	strictMTLS := newPeerAuthenticationConfig("mtls-strict", "mtls", security.PeerAuthentication_MutualTLS_STRICT)
	push := newRDSTestPushContext(t, []config.Config{routeCfg, strictMTLS}, []*model.Service{
		reviewsSvc, v1Svc, v2Svc,
	})

	proxy := newMTLSTestProxy()
	proxy.ConfigNamespace = "moviereview"
	gen := &TripleConfigGenerator{}

	writeProtoFixture := func(filename string, raw []byte) {
		t.Helper()
		if len(raw) == 0 {
			t.Fatalf("empty serialized fixture for %s", filename)
		}
		target := filepath.Join(fixtureDir, filename)
		if err := os.WriteFile(target, raw, 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", target, err)
		}
	}

	ldsRes := gen.BuildListeners(proxy, push, []string{reviewsHost + ":9080"})
	if len(ldsRes) != 1 {
		t.Fatalf("LDS resources = %d, want 1", len(ldsRes))
	}
	l := &listener.Listener{}
	if err := ldsRes[0].GetResource().UnmarshalTo(l); err != nil {
		t.Fatalf("unmarshal LDS: %v", err)
	}
	writeProtoFixture("lds-reviews.pb", ldsRes[0].GetResource().GetValue())

	routeName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", host.Name(reviewsHost), 9080)
	rdsRes, _ := gen.BuildHTTPRoutes(proxy, &model.PushRequest{Push: push}, []string{routeName})
	if len(rdsRes) != 1 {
		t.Fatalf("RDS resources = %d, want 1", len(rdsRes))
	}
	rc := &route.RouteConfiguration{}
	if err := rdsRes[0].GetResource().UnmarshalTo(rc); err != nil {
		t.Fatalf("unmarshal RDS: %v", err)
	}
	assertRouteConfigurationSnapshotContract(t, rc)
	gotWeights := weightedClustersByName(t, rc.GetVirtualHosts()[0].GetRoutes()[0])
	if gotWeights["outbound|9080||"+v1Host] != 20 || gotWeights["outbound|9080||"+v2Host] != 80 {
		t.Fatalf("unexpected RDS weighted clusters: %v", gotWeights)
	}
	writeProtoFixture("rds-reviews.pb", rdsRes[0].GetResource().GetValue())

	v1ClusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", host.Name(v1Host), 9080)
	v2ClusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", host.Name(v2Host), 9080)
	cdsRes := gen.BuildClusters(proxy, push, []string{v1ClusterName, v2ClusterName})
	if len(cdsRes) != 2 {
		t.Fatalf("CDS resources = %d, want 2", len(cdsRes))
	}
	var mtlsSeen bool
	for _, res := range cdsRes {
		c := &cluster.Cluster{}
		if err := res.GetResource().UnmarshalTo(c); err != nil {
			t.Fatalf("unmarshal CDS %s: %v", res.Name, err)
		}
		if res.Name == v2ClusterName {
			if c.GetTransportSocket() == nil {
				t.Fatalf("expected mTLS transport socket for %s", v2ClusterName)
			}
			mtlsSeen = true
			writeProtoFixture("cds-reviews-v2-mtls.pb", res.GetResource().GetValue())
		} else {
			writeProtoFixture("cds-reviews-v1.pb", res.GetResource().GetValue())
		}
	}
	if !mtlsSeen {
		t.Fatalf("mTLS cluster %s not generated", v2ClusterName)
	}

	index := model.NewEndpointIndex(model.DisabledCache{})
	index.UpdateServiceEndpoints(model.ShardKey{}, v1Host, "moviereview", []*model.DubboEndpoint{{
		Addresses:       []string{"10.0.0.1"},
		EndpointPort:    9080,
		ServicePortName: "http",
		HealthStatus:    model.Healthy,
	}}, false)
	index.UpdateServiceEndpoints(model.ShardKey{}, v2Host, "mtls", []*model.DubboEndpoint{{
		Addresses:       []string{"10.0.0.2"},
		EndpointPort:    9080,
		ServicePortName: "http",
		ServiceAccount:  "spiffe://cluster.local/ns/mtls/sa/reviews-v2",
		HealthStatus:    model.Healthy,
	}}, false)

	for _, item := range []struct {
		clusterName string
		fileName    string
	}{
		{clusterName: v1ClusterName, fileName: "eds-reviews-v1.pb"},
		{clusterName: v2ClusterName, fileName: "eds-reviews-v2-mtls.pb"},
	} {
		cla := endpoints.NewEndpointBuilder(item.clusterName, proxy, push).BuildClusterLoadAssignment(index)
		if len(cla.GetEndpoints()) == 0 {
			t.Fatalf("EDS %s generated empty CLA", item.clusterName)
		}
		raw, err := proto.Marshal(cla)
		if err != nil {
			t.Fatalf("marshal EDS %s: %v", item.clusterName, err)
		}
		roundtrip := &endpoint.ClusterLoadAssignment{}
		if err := proto.Unmarshal(raw, roundtrip); err != nil {
			t.Fatalf("roundtrip EDS %s: %v", item.clusterName, err)
		}
		writeProtoFixture(item.fileName, raw)
	}
}
