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

package nodeagent

import (
	"encoding/json"
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
)

func TestParseNetConfDefaults(t *testing.T) {
	conf, err := ParseNetConf([]byte(`{"name":"dubbo"}`))
	if err != nil {
		t.Fatalf("ParseNetConf() failed: %v", err)
	}
	if conf.CNIVersion != defaultCNIVersion {
		t.Fatalf("cniVersion = %q, want %q", conf.CNIVersion, defaultCNIVersion)
	}
	if conf.ManagedLabel != inject.ProxylessManagedLabel || conf.ManagedLabelValue != inject.ProxylessManagedLabelValue {
		t.Fatalf("managed label = %s/%s, want %s/%s", conf.ManagedLabel, conf.ManagedLabelValue,
			inject.ProxylessManagedLabel, inject.ProxylessManagedLabelValue)
	}
	if conf.GRPCInboundPort != inject.ProxylessGRPCInboundPort {
		t.Fatalf("grpcInboundPort = %d, want %d", conf.GRPCInboundPort, inject.ProxylessGRPCInboundPort)
	}
	if conf.IPTablesPath != defaultIPTablesPath || conf.IPSetPath != defaultIPSetPath {
		t.Fatalf("iptables/ipset path = %s/%s, want %s/%s", conf.IPTablesPath, conf.IPSetPath, defaultIPTablesPath, defaultIPSetPath)
	}
}

func TestPodRefFromCNIArgs(t *testing.T) {
	ref, ok := PodRefFromCNIArgs("IgnoreUnknown=1;K8S_POD_NAMESPACE=app;K8S_POD_NAME=nginx-v1")
	if !ok {
		t.Fatal("PodRefFromCNIArgs() ok = false, want true")
	}
	if ref.Namespace != "app" || ref.Name != "nginx-v1" {
		t.Fatalf("pod ref = %#v, want app/nginx-v1", ref)
	}
}

func TestPodIPsFromPrevResult(t *testing.T) {
	prev, _ := json.Marshal(map[string]any{
		"cniVersion": "1.0.0",
		"ips": []map[string]string{
			{"address": "10.244.0.12/24"},
		},
	})
	ips := PodIPsFromPrevResult(prev)
	if len(ips) != 1 || ips[0] != "10.244.0.12" {
		t.Fatalf("ips = %v, want [10.244.0.12]", ips)
	}
}
