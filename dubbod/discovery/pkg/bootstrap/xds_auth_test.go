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

package bootstrap

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAuthorizeXDSPod(t *testing.T) {
	for _, tc := range []struct {
		name     string
		change   func(*model.Proxy, *corev1.Pod)
		identity string
		wantOK   bool
	}{
		{name: "workload", wantOK: true},
		{name: "wrong service account", identity: "spiffe://cluster.local/ns/app/sa/other"},
		{name: "wrong namespace", identity: "spiffe://cluster.local/ns/other/sa/default"},
		{name: "wrong trust domain", identity: "spiffe://other/ns/app/sa/default"},
		{name: "wrong IP", change: func(p *model.Proxy, _ *corev1.Pod) { p.IPAddresses = []string{"10.0.0.2"} }},
		{name: "wrong pod", change: func(p *model.Proxy, _ *corev1.Pod) { p.ID = "other.app" }},
		{name: "unmanaged", change: func(_ *model.Proxy, pod *corev1.Pod) { pod.Annotations = nil }},
		{name: "deleted", change: func(_ *model.Proxy, pod *corev1.Pod) { now := metav1.Now(); pod.DeletionTimestamp = &now }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "app", Annotations: map[string]string{inject.InherentInjectTemplatesAnnoName: inject.InherentGRPCTemplateName}}, Status: corev1.PodStatus{PodIP: "10.0.0.1"}}
			proxy := &model.Proxy{ID: "pod.app", Type: model.Inherent, Metadata: &model.NodeMetadata{Namespace: "app"}, IPAddresses: []string{"10.0.0.1"}}
			if tc.change != nil {
				tc.change(proxy, pod)
			}
			identity := tc.identity
			if identity == "" {
				identity = "spiffe://cluster.local/ns/app/sa/default"
			}
			err := authorizeXDSPod(proxy, []string{identity}, pod, []string{"cluster.local"})
			if (err == nil) != tc.wantOK {
				t.Fatalf("authorizeXDSPod = %v", err)
			}
		})
	}
}

func TestXDSRejectsUnmanagedCluster(t *testing.T) {
	s := &Server{clusterID: "local"}
	for _, clusterID := range []string{"", "other", "local"} {
		proxy := &model.Proxy{Metadata: &model.NodeMetadata{Namespace: "app"}}
		proxy.Metadata.ClusterID = cluster.ID(clusterID)
		if err := s.authorizeXDSWorkload(proxy, nil); err == nil {
			t.Fatal("accepted workload without managed cluster")
		}
	}
}
