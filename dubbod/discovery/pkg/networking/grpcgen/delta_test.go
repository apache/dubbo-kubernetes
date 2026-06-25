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

package grpcgen

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	v1 "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds/v1"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
)

func TestGenerateDeltasBuildsOnlyChangedServiceClusters(t *testing.T) {
	nginxHost := "nginx.app.svc.cluster.local"
	apiHost := "api.app.svc.cluster.local"
	nginxCluster := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", host.Name(nginxHost), 80)
	apiCluster := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", host.Name(apiHost), 80)
	push := newRDSTestPushContext(t, nil, []*model.Service{
		newRDSTestService("nginx", "app", nginxHost, 80),
		newRDSTestService("api", "app", apiHost, 80),
	})

	resources, removed, details, usedDelta, err := (&GrpcConfigGenerator{}).GenerateDeltas(newMTLSTestProxy(), &model.PushRequest{
		Push:           push,
		Full:           true,
		ConfigsUpdated: sets.New(model.ConfigKey{Kind: kind.Service, Name: nginxHost, Namespace: "app"}),
	}, &model.WatchedResource{
		TypeUrl:       v1.ClusterType,
		ResourceNames: sets.New(nginxCluster, apiCluster),
	})
	if err != nil {
		t.Fatalf("GenerateDeltas() failed: %v", err)
	}
	if !usedDelta || !details.Incremental {
		t.Fatalf("usedDelta/details.Incremental = %v/%v, want true/true", usedDelta, details.Incremental)
	}
	if got := resourceNamesForTest(resources); len(got) != 1 || got[0] != nginxCluster {
		t.Fatalf("resources = %v, want [%s]", got, nginxCluster)
	}
	if len(removed) != 0 {
		t.Fatalf("removed = %v, want empty", removed)
	}
}

func TestGenerateDeltasRemovesClustersForDeletedService(t *testing.T) {
	nginxHost := "nginx.app.svc.cluster.local"
	nginxCluster := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", host.Name(nginxHost), 80)
	push := newRDSTestPushContext(t, nil, nil)

	resources, removed, _, usedDelta, err := (&GrpcConfigGenerator{}).GenerateDeltas(newMTLSTestProxy(), &model.PushRequest{
		Push:           push,
		Full:           true,
		ConfigsUpdated: sets.New(model.ConfigKey{Kind: kind.Service, Name: nginxHost, Namespace: "app"}),
	}, &model.WatchedResource{
		TypeUrl:       v1.ClusterType,
		ResourceNames: sets.New(nginxCluster),
	})
	if err != nil {
		t.Fatalf("GenerateDeltas() failed: %v", err)
	}
	if !usedDelta {
		t.Fatal("usedDelta = false, want true")
	}
	if len(resources) != 0 {
		t.Fatalf("resources = %v, want empty", resourceNamesForTest(resources))
	}
	if len(removed) != 1 || removed[0] != nginxCluster {
		t.Fatalf("removed = %v, want [%s]", removed, nginxCluster)
	}
}

func resourceNamesForTest(resources model.Resources) []string {
	out := make([]string, 0, len(resources))
	for _, resource := range resources {
		out = append(out, resource.Name)
	}
	return out
}
