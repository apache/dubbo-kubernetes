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

package xds

import (
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/util/protoconv"
	core "github.com/dubml/xds-api/core/v1"
	router "github.com/dubml/xds-api/extensions/filters/v1/http/router"
	hcm "github.com/dubml/xds-api/extensions/filters/v1/network/http_connection_manager"
	listener "github.com/dubml/xds-api/listener/v1"
	route "github.com/dubml/xds-api/route/v1"
	discovery "github.com/dubml/xds-api/service/discovery/v1"
	standardcore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	standardlistener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	standardroute "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	standardhcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	standard "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestStandardADSConvertsRouteSemanticsByFieldName(t *testing.T) {
	input := &route.RouteConfiguration{Name: "routes", VirtualHosts: []*route.VirtualHost{{Name: "host", Domains: []string{"*"}, Routes: []*route.Route{{Match: &route.RouteMatch{PathSpecifier: &route.RouteMatch_Prefix{Prefix: "/"}}, Action: &route.Route_Route{Route: &route.RouteAction{ClusterSpecifier: &route.RouteAction_Cluster{Cluster: "backend"}, Timeout: durationpb.New(3 * time.Second)}}}}}}}
	before := proto.Clone(input)
	output, err := standardAny(protoconv.MessageToAny(input))
	if err != nil {
		t.Fatal(err)
	}
	decoded := &standardroute.RouteConfiguration{}
	if err := output.UnmarshalTo(decoded); err != nil {
		t.Fatal(err)
	}
	action := decoded.VirtualHosts[0].Routes[0].GetRoute()
	if action.GetCluster() != "backend" || action.GetTimeout().AsDuration() != 3*time.Second {
		t.Fatalf("changed route semantics: %v", action)
	}
	if !proto.Equal(before, input) {
		t.Fatal("conversion mutated the shared custom resource")
	}
}

type recordingStandardStream struct {
	grpc.ServerStream
	request   *standard.DiscoveryRequest
	responses []*standard.DiscoveryResponse
}

func (s *recordingStandardStream) Recv() (*standard.DiscoveryRequest, error) { return s.request, nil }
func (s *recordingStandardStream) Send(response *standard.DiscoveryResponse) error {
	s.responses = append(s.responses, response)
	return nil
}

func TestStandardADSDefersDependenciesUntilSubscribedAndHonorsUnsubscribe(t *testing.T) {
	transport := &recordingStandardStream{}
	stream := newStandardStream(transport)
	resourceType := "type.googleapis.com/route.v1.RouteConfiguration"
	standardType := "type.googleapis.com/envoy.config.route.v3.RouteConfiguration"
	response := func(nonce string) *discovery.DiscoveryResponse {
		return &discovery.DiscoveryResponse{TypeUrl: resourceType, VersionInfo: nonce, Nonce: nonce, Resources: []*anypb.Any{protoconv.MessageToAny(&route.RouteConfiguration{Name: "route"})}}
	}
	for _, nonce := range []string{"old", "new"} {
		if err := stream.Send(response(nonce)); err != nil {
			t.Fatal(err)
		}
	}
	if len(transport.responses) != 0 {
		t.Fatal("SDK received an unsolicited dependency")
	}
	transport.request = &standard.DiscoveryRequest{TypeUrl: standardType, ResourceNames: []string{"route"}}
	converted, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if converted.TypeUrl != resourceType || len(transport.responses) != 1 || transport.responses[0].Nonce != "new" {
		t.Fatal("first watch did not receive the latest dependency")
	}
	transport.request = &standard.DiscoveryRequest{TypeUrl: standardType, ResourceNames: []string{"route"}, ResponseNonce: "new", VersionInfo: "new"}
	converted, err = stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if converted.ResponseNonce != "new" || converted.VersionInfo != "new" || len(transport.responses) != 1 {
		t.Fatal("ACK changed subscription state or nonce")
	}
	transport.request = &standard.DiscoveryRequest{TypeUrl: standardType}
	if _, err := stream.Recv(); err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(response("unsubscribed")); err != nil {
		t.Fatal(err)
	}
	if len(transport.responses) != 1 {
		t.Fatal("SDK received an update after unsubscribe")
	}
	transport.request = &standard.DiscoveryRequest{TypeUrl: standardType, ResourceNames: []string{"route"}}
	if _, err := stream.Recv(); err != nil {
		t.Fatal(err)
	}
	if len(transport.responses) != 2 || transport.responses[1].Nonce != "unsubscribed" {
		t.Fatal("new watch did not receive the current resource")
	}
}

func TestStandardADSConvertsNestedAnyAndSetsV3ConfigSource(t *testing.T) {
	manager := &hcm.HttpConnectionManager{RouteSpecifier: &hcm.HttpConnectionManager_Rds{Rds: &hcm.Rds{RouteConfigName: "routes", ConfigSource: &core.ConfigSource{ConfigSourceSpecifier: &core.ConfigSource_Ads{Ads: &core.AggregatedConfigSource{}}}}}, HttpFilters: []*hcm.HttpFilter{{Name: "envoy.filters.http.router", ConfigType: &hcm.HttpFilter_TypedConfig{TypedConfig: protoconv.MessageToAny(&router.Router{})}}}}
	input := &listener.Listener{Name: "service", ApiListener: &listener.ApiListener{ApiListener: protoconv.MessageToAny(manager)}}
	output, err := standardAny(protoconv.MessageToAny(input))
	if err != nil {
		t.Fatal(err)
	}
	decoded := &standardlistener.Listener{}
	if err := output.UnmarshalTo(decoded); err != nil {
		t.Fatal(err)
	}
	converted := &standardhcm.HttpConnectionManager{}
	if err := decoded.ApiListener.ApiListener.UnmarshalTo(converted); err != nil {
		t.Fatal(err)
	}
	if converted.GetRds().GetConfigSource().GetAds() == nil || converted.GetRds().GetConfigSource().GetResourceApiVersion() != standardcore.ApiVersion_V3 {
		t.Fatalf("invalid standard RDS source: %v", converted.GetRds())
	}
	if converted.HttpFilters[0].GetTypedConfig().TypeUrl != "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router" {
		t.Fatal("nested HTTP filter retained custom type URL")
	}
}

func TestStandardADSRejectsUnrepresentableAndMalformedResources(t *testing.T) {
	for _, resource := range []*anypb.Any{nil, {TypeUrl: "type.googleapis.com/unknown.Message"}, {TypeUrl: "type.googleapis.com/route.v1.RouteConfiguration", Value: []byte{0xff}}, protoconv.MessageToAny(&route.RouteConfiguration{Name: "route", AgentConfig: &route.AgentConfig{}})} {
		if _, err := standardAny(resource); err == nil {
			t.Fatalf("accepted unsupported resource: %v", resource)
		}
	}
}
