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
	"encoding/json"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/memory"
	discoverymodel "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/dubboagency/grpcxds"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	networking "github.com/kdubbo/api/networking/v1alpha3"
	security "github.com/kdubbo/api/security/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestShouldManageProxylessGRPCPod(t *testing.T) {
	tests := []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{
			name: "explicit grpc-engine template",
			pod: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					inject.ProxylessInjectTemplatesAnnoName: inject.ProxylessGRPCTemplateName,
				},
			}},
			want: true,
		},
		{
			name: "default injected pod marked by bootstrap env",
			pod: &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{
				Env: []corev1.EnvVar{{
					Name:  "GRPC_XDS_BOOTSTRAP",
					Value: inject.ProxylessGRPCBootstrapPath,
				}},
			}}}},
			want: true,
		},
		{
			name: "default injected pod marked by runtime config env",
			pod: &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{
				Env: []corev1.EnvVar{{
					Name:  inject.ProxylessGRPCConfigEnvName,
					Value: inject.ProxylessGRPCConfigPath,
				}},
			}}}},
			want: true,
		},
		{
			name: "default injected pod marked by xds secret volume",
			pod: &corev1.Pod{Spec: corev1.PodSpec{Volumes: []corev1.Volume{{
				Name: inject.ProxylessXDSVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{SecretName: "dubbo-xds-pod"},
				},
			}}}},
			want: true,
		},
		{
			name: "not proxyless grpc",
			pod:  &corev1.Pod{},
			want: false,
		},
		{
			name: "nil pod",
			pod:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldManageProxylessGRPCPod(tt.pod); got != tt.want {
				t.Fatalf("shouldManageProxylessGRPCPod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildProxylessGRPCSecretIncludesRuntimeConfig(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "nginx",
		Namespace: "app",
		UID:       "pod-uid",
	}}

	secret := buildProxylessGRPCSecret(pod, []byte("bootstrap"), []byte("runtime"), []byte("cert"), []byte("key"), []byte("root"))
	if secret.Name != inject.ProxylessGRPCSecretNameForMeta(pod.ObjectMeta) {
		t.Fatalf("secret name = %q, want proxyless secret name", secret.Name)
	}
	if got := string(secret.Data[inject.ProxylessGRPCBootstrapFileName]); got != "bootstrap" {
		t.Fatalf("bootstrap data = %q, want bootstrap", got)
	}
	if got := string(secret.Data[inject.ProxylessGRPCConfigFileName]); got != "runtime" {
		t.Fatalf("runtime config data = %q, want runtime", got)
	}
	if got := string(secret.Data[constants.CACertNamespaceConfigMapDataName]); got != "root" {
		t.Fatalf("root cert data = %q, want root", got)
	}
	if len(secret.OwnerReferences) != 1 || secret.OwnerReferences[0].UID != pod.UID {
		t.Fatalf("owner references = %+v, want pod owner", secret.OwnerReferences)
	}
}

func TestBuildRuntimeConfigJSON(t *testing.T) {
	workload := &proxylessGRPCWorkloadContext{
		nodeID:           "proxyless~10.0.0.1~nginx.app~app.svc.cluster.local",
		podName:          "nginx",
		podNamespace:     "app",
		podIP:            "10.0.0.1",
		serviceAccount:   "nginx",
		trustDomain:      "cluster.local",
		clusterID:        "remote",
		discoveryAddress: "192.168.15.164:32049",
		caAddress:        "192.168.15.164:32049",
	}

	data, err := buildRuntimeConfigJSON(workload, nil, nil)
	if err != nil {
		t.Fatalf("buildRuntimeConfigJSON() failed: %v", err)
	}

	var got proxylessGRPCRuntimeConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("runtime config is not valid JSON: %v", err)
	}

	if got.Version != proxylessGRPCRuntimeConfigVersion {
		t.Fatalf("version = %q, want %q", got.Version, proxylessGRPCRuntimeConfigVersion)
	}
	if got.Mode != "proxyless-grpc" {
		t.Fatalf("mode = %q, want proxyless-grpc", got.Mode)
	}
	if got.Env["GRPC_XDS_BOOTSTRAP"] != inject.ProxylessGRPCBootstrapPath {
		t.Fatalf("GRPC_XDS_BOOTSTRAP = %q, want %q", got.Env["GRPC_XDS_BOOTSTRAP"], inject.ProxylessGRPCBootstrapPath)
	}
	if got.Env[inject.ProxylessGRPCConfigEnvName] != inject.ProxylessGRPCConfigPath {
		t.Fatalf("%s = %q, want %q", inject.ProxylessGRPCConfigEnvName, got.Env[inject.ProxylessGRPCConfigEnvName], inject.ProxylessGRPCConfigPath)
	}
	if got.Bootstrap.DiscoveryAddress != workload.discoveryAddress {
		t.Fatalf("discoveryAddress = %q, want %q", got.Bootstrap.DiscoveryAddress, workload.discoveryAddress)
	}
	if got.Env["DUBBO_META_CLUSTER_ID"] != workload.clusterID {
		t.Fatalf("DUBBO_META_CLUSTER_ID = %q, want %q", got.Env["DUBBO_META_CLUSTER_ID"], workload.clusterID)
	}
	if got.Env[inject.ProxylessGRPCKeepaliveEnvName] != inject.ProxylessGRPCKeepaliveValue {
		t.Fatalf("%s = %q, want %q", inject.ProxylessGRPCKeepaliveEnvName, got.Env[inject.ProxylessGRPCKeepaliveEnvName], inject.ProxylessGRPCKeepaliveValue)
	}
	if got.Env[inject.ProxylessGRPCKeepaliveTimeEnv] != inject.ProxylessGRPCKeepaliveTime {
		t.Fatalf("%s = %q, want %q", inject.ProxylessGRPCKeepaliveTimeEnv, got.Env[inject.ProxylessGRPCKeepaliveTimeEnv], inject.ProxylessGRPCKeepaliveTime)
	}
	if got.Env[inject.ProxylessGRPCKeepaliveTimeoutEnv] != inject.ProxylessGRPCKeepaliveTimeout {
		t.Fatalf("%s = %q, want %q", inject.ProxylessGRPCKeepaliveTimeoutEnv, got.Env[inject.ProxylessGRPCKeepaliveTimeoutEnv], inject.ProxylessGRPCKeepaliveTimeout)
	}
	if got.Env[inject.ProxylessGRPCKeepalivePermitWithoutStreamEnv] != inject.ProxylessGRPCKeepaliveValue {
		t.Fatalf("%s = %q, want %q", inject.ProxylessGRPCKeepalivePermitWithoutStreamEnv, got.Env[inject.ProxylessGRPCKeepalivePermitWithoutStreamEnv], inject.ProxylessGRPCKeepaliveValue)
	}
	if !got.Keepalive.Enabled {
		t.Fatalf("keepalive enabled = false, want true")
	}
	if got.Keepalive.Time != inject.ProxylessGRPCKeepaliveTime {
		t.Fatalf("keepalive time = %q, want %q", got.Keepalive.Time, inject.ProxylessGRPCKeepaliveTime)
	}
	if got.Keepalive.Timeout != inject.ProxylessGRPCKeepaliveTimeout {
		t.Fatalf("keepalive timeout = %q, want %q", got.Keepalive.Timeout, inject.ProxylessGRPCKeepaliveTimeout)
	}
	if !got.Keepalive.PermitWithoutStream {
		t.Fatalf("keepalive permitWithoutStream = false, want true")
	}
	if got.Env[inject.ProxylessXDSAddressEnvName] != workload.discoveryAddress {
		t.Fatalf("%s = %q, want %q", inject.ProxylessXDSAddressEnvName, got.Env[inject.ProxylessXDSAddressEnvName], workload.discoveryAddress)
	}
	if got.Env["CA_ADDRESS"] != workload.caAddress {
		t.Fatalf("CA_ADDRESS = %q, want %q", got.Env["CA_ADDRESS"], workload.caAddress)
	}
	if got.Certificates.Provider != grpcxds.FileWatcherCertProviderName {
		t.Fatalf("cert provider = %q, want %q", got.Certificates.Provider, grpcxds.FileWatcherCertProviderName)
	}
	if got.Certificates.CertChain != inject.ProxylessXDSMountPath+"/"+constants.CertChainFilename {
		t.Fatalf("certChain = %q, want mounted cert-chain path", got.Certificates.CertChain)
	}
	if got.Workload.NodeID != workload.nodeID {
		t.Fatalf("nodeId = %q, want %q", got.Workload.NodeID, workload.nodeID)
	}
	if got.Workload.PodIP != workload.podIP {
		t.Fatalf("podIP = %q, want %q", got.Workload.PodIP, workload.podIP)
	}
	if got.Workload.ClusterID != workload.clusterID {
		t.Fatalf("clusterId = %q, want %q", got.Workload.ClusterID, workload.clusterID)
	}
}

func TestBuildWorkloadContextUsesInjectedRemoteValues(t *testing.T) {
	controller := &proxylessGRPCWorkloadController{
		server: &Server{
			environment: &discoverymodel.Environment{
				DomainSuffix: constants.DefaultClusterLocalDomain,
			},
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx",
			Namespace: "app",
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "nginx",
			Containers: []corev1.Container{{
				Name: "app",
				Env: []corev1.EnvVar{
					{Name: "DUBBO_META_CLUSTER_ID", Value: "remote"},
					{Name: inject.ProxylessXDSAddressEnvName, Value: "192.168.15.164:32049"},
					{Name: "CA_ADDRESS", Value: "192.168.15.164:32049"},
				},
			}},
		},
		Status: corev1.PodStatus{
			PodIP: "10.0.0.1",
		},
	}

	workload, err := controller.buildWorkloadContext(pod)
	if err != nil {
		t.Fatalf("buildWorkloadContext() failed: %v", err)
	}
	if workload.clusterID != "remote" {
		t.Fatalf("clusterID = %q, want remote", workload.clusterID)
	}
	if workload.discoveryAddress != "192.168.15.164:32049" {
		t.Fatalf("discoveryAddress = %q, want 192.168.15.164:32049", workload.discoveryAddress)
	}
	if workload.caAddress != "192.168.15.164:32049" {
		t.Fatalf("caAddress = %q, want 192.168.15.164:32049", workload.caAddress)
	}
}

func TestBuildRuntimeTrafficConfigCapturesProxylessSecurity(t *testing.T) {
	hostname := host.Name("provider.grpc-app.svc.cluster.local")
	svc := newProxylessRuntimeTestService("provider", "grpc-app", string(hostname), 17070)
	push := newProxylessRuntimeTestPushContext(t, []config.Config{
		newProxylessMTLSMeshServiceConfig("provider-mtls", "grpc-app", hostname),
		newProxylessStrictPeerAuthenticationConfig("grpc-app-strict-mtls", "grpc-app"),
	}, []*discoverymodel.Service{svc})

	serviceConfig := buildRuntimeServiceConfig(push, nil, svc)
	if len(serviceConfig.Ports) != 1 {
		t.Fatalf("ports = %d, want 1", len(serviceConfig.Ports))
	}
	if got := serviceConfig.Ports[0].MTLSMode; got != "STRICT" {
		t.Fatalf("mtlsMode = %q, want STRICT", got)
	}

	routeConfig := buildRuntimeRouteConfig(push, nil, svc, 17070)
	if len(routeConfig.Destinations) != 2 {
		t.Fatalf("destinations = %d, want 2", len(routeConfig.Destinations))
	}
	wantWeights := map[string]int{"v1": 50, "v2": 50}
	for _, destination := range routeConfig.Destinations {
		if destination.TLSMode != "DUBBO_MUTUAL" {
			t.Fatalf("destination %s tlsMode = %q, want DUBBO_MUTUAL", destination.Subset, destination.TLSMode)
		}
		if wantWeights[destination.Subset] != destination.Weight {
			t.Fatalf("destination %s weight = %d, want %d", destination.Subset, destination.Weight, wantWeights[destination.Subset])
		}
	}
}

func TestBuildRuntimeTrafficConfigCapturesPermissivePeerAuthentication(t *testing.T) {
	hostname := host.Name("provider.grpc-app.svc.cluster.local")
	svc := newProxylessRuntimeTestService("provider", "grpc-app", string(hostname), 17070)
	push := newProxylessRuntimeTestPushContext(t, []config.Config{
		newProxylessPeerAuthenticationConfig("grpc-app-permissive-mtls", "grpc-app", security.PeerAuthentication_MutualTLS_PERMISSIVE),
	}, []*discoverymodel.Service{svc})

	serviceConfig := buildRuntimeServiceConfig(push, nil, svc)
	if len(serviceConfig.Ports) != 1 {
		t.Fatalf("ports = %d, want 1", len(serviceConfig.Ports))
	}
	if got := serviceConfig.Ports[0].MTLSMode; got != "PERMISSIVE" {
		t.Fatalf("mtlsMode = %q, want PERMISSIVE", got)
	}
}

func TestProxylessGRPCRuntimeConfigNeedsUpdate(t *testing.T) {
	tests := []struct {
		name string
		req  *discoverymodel.PushRequest
		want bool
	}{
		{
			name: "full push",
			req:  &discoverymodel.PushRequest{Full: true},
			want: true,
		},
		{
			name: "meshservice",
			req: &discoverymodel.PushRequest{
				ConfigsUpdated: sets.New(discoverymodel.ConfigKey{Kind: kind.MeshService, Name: "provider-mtls", Namespace: "grpc-app"}),
			},
			want: true,
		},
		{
			name: "peerauthentication",
			req: &discoverymodel.PushRequest{
				ConfigsUpdated: sets.New(discoverymodel.ConfigKey{Kind: kind.PeerAuthentication, Name: "strict", Namespace: "grpc-app"}),
			},
			want: true,
		},
		{
			name: "requestauthentication",
			req: &discoverymodel.PushRequest{
				ConfigsUpdated: sets.New(discoverymodel.ConfigKey{Kind: kind.RequestAuthentication, Name: "jwt", Namespace: "grpc-app"}),
			},
			want: true,
		},
		{
			name: "authorizationpolicy",
			req: &discoverymodel.PushRequest{
				ConfigsUpdated: sets.New(discoverymodel.ConfigKey{Kind: kind.AuthorizationPolicy, Name: "require-jwt", Namespace: "grpc-app"}),
			},
			want: true,
		},
		{
			name: "unrelated configmap",
			req: &discoverymodel.PushRequest{
				ConfigsUpdated: sets.New(discoverymodel.ConfigKey{Kind: kind.ConfigMap, Name: "ui", Namespace: "dubbo-system"}),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := proxylessGRPCRuntimeConfigNeedsUpdate(tt.req); got != tt.want {
				t.Fatalf("proxylessGRPCRuntimeConfigNeedsUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNextRotationTime(t *testing.T) {
	now := time.Now()
	rotations := map[types.NamespacedName]time.Time{
		{Name: "late", Namespace: "app"}:  now.Add(time.Hour),
		{Name: "early", Namespace: "app"}: now.Add(time.Minute),
	}

	got, found := nextRotationTime(rotations)
	if !found {
		t.Fatalf("nextRotationTime() found = false, want true")
	}
	if !got.Equal(now.Add(time.Minute)) {
		t.Fatalf("nextRotationTime() = %v, want %v", got, now.Add(time.Minute))
	}

	if _, found := nextRotationTime(nil); found {
		t.Fatalf("nextRotationTime(nil) found = true, want false")
	}
}

func newProxylessRuntimeTestPushContext(t *testing.T, configs []config.Config, services []*discoverymodel.Service) *discoverymodel.PushContext {
	t.Helper()

	store := memory.Make(collections.DubboGatewayAPI())
	for _, cfg := range configs {
		if _, err := store.Create(cfg); err != nil {
			t.Fatalf("create config %s/%s: %v", cfg.Namespace, cfg.Name, err)
		}
	}

	env := discoverymodel.NewEnvironment()
	env.ConfigStore = store
	env.ServiceDiscovery = proxylessRuntimeStaticServiceDiscovery{services: services}
	env.Watcher = meshwatcher.ConfigAdapter(krt.NewStatic(&meshwatcher.MeshConfigResource{
		MeshConfig: mesh.DefaultMeshConfig(),
	}, true))
	env.Init()

	push := discoverymodel.NewPushContext()
	push.InitContext(env, nil, nil)
	return push
}

func newProxylessRuntimeTestService(name, namespace, hostname string, port int) *discoverymodel.Service {
	return &discoverymodel.Service{
		Hostname: host.Name(hostname),
		Ports: discoverymodel.PortList{
			{
				Name:     "grpc",
				Port:     port,
				Protocol: protocol.HTTP2,
			},
		},
		Attributes: discoverymodel.ServiceAttributes{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func newProxylessMTLSMeshServiceConfig(name, namespace string, hostname host.Name) config.Config {
	return config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.MeshService,
			Name:             name,
			Namespace:        namespace,
			Domain:           "cluster.local",
		},
		Spec: &networking.MeshService{
			Hosts: []string{string(hostname)},
			TrafficPolicy: &networking.TrafficPolicy{
				Tls: &networking.ClientTLSSettings{
					Mode: networking.ClientTLSSettings_DUBBO_MUTUAL,
				},
			},
			Routes: []*networking.MeshServiceRoute{{
				Service: []*networking.ServiceDestination{{
					Name:   "v1",
					Host:   string(hostname),
					Labels: map[string]string{"version": "v1"},
					Port:   &networking.ServicePort{Number: 17070},
					Weight: 50,
				}, {
					Name:   "v2",
					Host:   string(hostname),
					Labels: map[string]string{"version": "v2"},
					Port:   &networking.ServicePort{Number: 17070},
					Weight: 50,
				}},
			}},
		},
	}
}

func newProxylessStrictPeerAuthenticationConfig(name, namespace string) config.Config {
	return newProxylessPeerAuthenticationConfig(name, namespace, security.PeerAuthentication_MutualTLS_STRICT)
}

func newProxylessPeerAuthenticationConfig(name, namespace string, mode security.PeerAuthentication_MutualTLS_Mode) config.Config {
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

type proxylessRuntimeStaticServiceDiscovery struct {
	services []*discoverymodel.Service
}

func (s proxylessRuntimeStaticServiceDiscovery) Services() []*discoverymodel.Service {
	return s.services
}

func (s proxylessRuntimeStaticServiceDiscovery) GetService(hostname host.Name) *discoverymodel.Service {
	for _, svc := range s.services {
		if svc.Hostname == hostname {
			return svc
		}
	}
	return nil
}

func (s proxylessRuntimeStaticServiceDiscovery) GetProxyServiceTargets(*discoverymodel.Proxy) []discoverymodel.ServiceTarget {
	return nil
}

func TestScheduleRotationTracksEarliestTimer(t *testing.T) {
	now := time.Now()
	controller := &proxylessGRPCWorkloadController{
		rotations: make(map[types.NamespacedName]time.Time),
	}
	defer controller.stopAllTimers()

	late := types.NamespacedName{Name: "late", Namespace: "app"}
	early := types.NamespacedName{Name: "early", Namespace: "app"}
	controller.scheduleRotation(late, now.Add(100*time.Hour))
	controller.scheduleRotation(early, now.Add(10*time.Hour))

	controller.rotationMu.Lock()
	next := controller.nextRotation
	controller.rotationMu.Unlock()
	if next.Before(now.Add(7*time.Hour)) || next.After(now.Add(9*time.Hour)) {
		t.Fatalf("nextRotation = %v, want around 8h from now", next.Sub(now))
	}

	controller.clearRotation(early)
	controller.rotationMu.Lock()
	next = controller.nextRotation
	controller.rotationMu.Unlock()
	if next.Before(now.Add(79*time.Hour)) || next.After(now.Add(81*time.Hour)) {
		t.Fatalf("nextRotation after clearing earliest = %v, want around 80h from now", next.Sub(now))
	}
}
