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

package aggregate

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/serviceregistry/provider"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
)

func TestServiceEntryDecoratesKubernetesService(t *testing.T) {
	hostname := host.Name("reviews.bookinfo.svc.cluster.local")
	kubeService := &model.Service{Hostname: hostname, ServiceAccounts: []string{"spiffe://cluster.local/ns/bookinfo/sa/default"}}
	externalService := &model.Service{Hostname: hostname, ServiceAccounts: []string{
		"spiffe://cluster.local/ns/bookinfo/sa/default",
		"spiffe://cluster.local/ns/bookinfo/sa/reviews",
	}}
	controller := NewController(Options{})
	controller.addRegistry(&staticRegistry{provider: provider.Kubernetes, services: []*model.Service{kubeService}}, nil)
	controller.addRegistry(&staticRegistry{provider: provider.External, services: []*model.Service{externalService}}, nil)

	services := controller.Services()
	if len(services) != 1 {
		t.Fatalf("got %d services, want one merged service", len(services))
	}
	if got := len(services[0].ServiceAccounts); got != 2 {
		t.Fatalf("got %d service accounts, want 2", got)
	}
	if got := controller.GetService(hostname); got == nil || len(got.ServiceAccounts) != 2 {
		t.Fatalf("GetService() = %#v, want decorated Kubernetes service", got)
	}
}

type staticRegistry struct {
	provider provider.ID
	services []*model.Service
}

func (*staticRegistry) Run(stop <-chan struct{})                                  { <-stop }
func (*staticRegistry) HasSynced() bool                                           { return true }
func (s *staticRegistry) Services() []*model.Service                              { return s.services }
func (s *staticRegistry) Provider() provider.ID                                   { return s.provider }
func (*staticRegistry) Cluster() cluster.ID                                       { return "cluster-1" }
func (*staticRegistry) GetProxyServiceTargets(*model.Proxy) []model.ServiceTarget { return nil }
func (s *staticRegistry) GetService(name host.Name) *model.Service {
	for _, service := range s.services {
		if service.Hostname == name {
			return service
		}
	}
	return nil
}
