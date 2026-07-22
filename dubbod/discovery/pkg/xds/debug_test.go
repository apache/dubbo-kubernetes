// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0.

package xds

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
)

func TestEndpointzReportsVMEndpointState(t *testing.T) {
	hostname := host.Name("reviews-vm.mesh.local")
	service := &model.Service{
		Hostname: hostname,
		Ports: model.PortList{{
			Name: "grpc", Port: 50051, Protocol: protocol.GRPC,
		}},
		Attributes: model.ServiceAttributes{Name: "reviews-vm", Namespace: "bookinfo"},
	}
	env := model.NewEnvironment()
	env.ServiceDiscovery = debugServiceDiscovery{service: service}
	env.ConfigStore = testConfigStore{}
	env.Watcher = meshwatcher.ConfigAdapter(krt.NewStatic(&meshwatcher.MeshConfigResource{
		MeshConfig: mesh.DefaultMeshConfig(),
	}, true))
	env.Init()
	push := model.NewPushContext()
	push.InitContext(env, nil, nil)
	env.SetPushContext(push)
	env.EndpointIndex.UpdateServiceEndpoints(model.ShardKey{}, string(hostname), "bookinfo", []*model.DubboEndpoint{{
		Addresses:       []string{"192.0.2.10"},
		EndpointPort:    50051,
		ServicePortName: "grpc",
		HealthStatus:    model.UnHealthy,
		Network:         "vm-network",
		Locality:        "us-east-1/zone-a/rack-1",
		LbWeight:        7,
		ServiceAccount:  "reviews-vm",
		WorkloadName:    "reviews-vm",
	}}, false)

	recorder := httptest.NewRecorder()
	NewDiscoveryServer(env, nil, nil).endpointz(recorder, httptest.NewRequest("GET", "/debug/endpointz", nil))
	if recorder.Code != 200 {
		t.Fatalf("status code = %d, want 200", recorder.Code)
	}
	var result []map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode endpointz: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("endpoint count = %d, want 1", len(result))
	}
	endpoint := result[0]
	if endpoint["hostname"] != string(hostname) || endpoint["address"] != "192.0.2.10" || endpoint["health"] != "UNHEALTHY" {
		t.Fatalf("unexpected endpoint: %v", endpoint)
	}
	if endpoint["network"] != "vm-network" || endpoint["locality"] != "us-east-1/zone-a/rack-1" || endpoint["weight"] != float64(7) {
		t.Fatalf("unexpected endpoint topology: %v", endpoint)
	}
}

type debugServiceDiscovery struct {
	service *model.Service
}

func (d debugServiceDiscovery) Services() []*model.Service { return []*model.Service{d.service} }
func (d debugServiceDiscovery) GetService(name host.Name) *model.Service {
	if d.service != nil && d.service.Hostname == name {
		return d.service
	}
	return nil
}
func (debugServiceDiscovery) GetProxyServiceTargets(*model.Proxy) []model.ServiceTarget { return nil }
