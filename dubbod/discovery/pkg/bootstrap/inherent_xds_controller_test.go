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
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/url"
	"strings"
	"testing"
	"time"

	discoverymodel "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/ra"
	pkiutil "github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/util"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/xdsbootstrap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func testWorkloadSecret(t *testing.T, identity string) (*corev1.Pod, *corev1.Secret) {
	t.Helper()
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	root := &x509.Certificate{SerialNumber: big.NewInt(1), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(24 * time.Hour)}
	rootDER, err := x509.CreateCertificate(rand.Reader, root, root, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	uri, err := url.Parse(identity)
	if err != nil {
		t.Fatal(err)
	}
	leaf := &x509.Certificate{SerialNumber: big.NewInt(2), URIs: []*url.URL{uri}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(24 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	der, err := x509.CreateCertificate(rand.Reader, leaf, root, &key.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: "app", UID: "pod-uid"}}
	secret := buildInherentXDSSecret(pod, nil, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER}))
	return pod, secret
}

func TestWorkloadControllerCannotOverwriteOrDeleteForeignCredentials(t *testing.T) {
	pod, secret := testWorkloadSecret(t, "spiffe://cluster.local/ns/app/sa/default")
	pod.Annotations = map[string]string{inject.InherentInjectTemplatesAnnoName: inject.InherentTripleTemplateName}
	secret.OwnerReferences[0].UID = "another-pod-uid"
	client := credentialsTestClient{core: fake.NewClientset(pod, secret)}
	c := &inherentXDSWorkloadController{client: client, pods: credentialsTestPods{pod: pod}}
	if err := c.reconcile(types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}); err == nil || !strings.Contains(err.Error(), "owned by another workload") {
		t.Fatalf("expected ownership conflict, got %v", err)
	}
	if err := c.deleteSecretForPod(pod); err != nil {
		t.Fatal(err)
	}
	got, err := client.Kube().CoreV1().Secrets(secret.Namespace).Get(context.Background(), secret.Name, metav1.GetOptions{})
	if err != nil || got.OwnerReferences[0].UID != "another-pod-uid" {
		t.Fatalf("foreign credentials changed or deleted: error=%v", err)
	}
}

type credentialsTestClient struct {
	kube.Client
	core kubernetes.Interface
}

func (c credentialsTestClient) Kube() kubernetes.Interface { return c.core }

type credentialsTestPods struct {
	kclient.Client[*corev1.Pod]
	pod *corev1.Pod
}

func (p credentialsTestPods) Get(name, namespace string) *corev1.Pod {
	if p.pod.Name == name && p.pod.Namespace == namespace {
		return p.pod
	}
	return nil
}

func TestShouldManageInherentXDSPod(t *testing.T) {
	tests := []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{name: "triple template", pod: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{inject.InherentInjectTemplatesAnnoName: inject.InherentTripleTemplateName}}}, want: true},
		{name: "triple bootstrap", pod: &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{Env: []corev1.EnvVar{{Name: inject.InherentTripleBootstrapEnvName, Value: inject.InherentTripleBootstrapPath}}}}}}, want: true},
		{name: "legacy template", pod: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{inject.InherentInjectTemplatesAnnoName: "grpc-engine"}}}},
		{name: "legacy bootstrap", pod: &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{Env: []corev1.EnvVar{{Name: "GRPC_XDS_BOOTSTRAP", Value: "/etc/dubbo/proxy/grpc-bootstrap.json"}}}}}}},
		{name: "unmanaged secret volume", pod: &corev1.Pod{Spec: corev1.PodSpec{Volumes: []corev1.Volume{{Name: inject.InherentXDSVolumeName, VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "dubbo-xds-pod"}}}}}}},
		{name: "plain pod", pod: &corev1.Pod{}},
		{name: "nil pod"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldManageInherentXDSPod(tt.pod); got != tt.want {
				t.Fatalf("managed = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildInherentXDSSecretIncludesTripleBootstrap(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "nginx",
		Namespace: "app",
		UID:       "pod-uid",
	}}

	secret := buildInherentXDSSecret(pod, []byte("bootstrap"), []byte("cert"), []byte("key"), []byte("root"))
	if secret.Name != inject.InherentXDSSecretNameForMeta(pod.ObjectMeta) {
		t.Fatalf("secret name = %q, want inherent secret name", secret.Name)
	}
	if got := string(secret.Data[inject.InherentTripleBootstrapFileName]); got != "bootstrap" {
		t.Fatalf("bootstrap data = %q, want bootstrap", got)
	}
	if len(secret.Data) != 4 {
		t.Fatalf("secret keys = %v, want only Triple bootstrap and workload credentials", secret.Data)
	}
	if got := string(secret.Data[constants.CACertNamespaceConfigMapDataName]); got != "root" {
		t.Fatalf("root cert data = %q, want root", got)
	}
	if len(secret.OwnerReferences) != 1 || secret.OwnerReferences[0].UID != pod.UID {
		t.Fatalf("owner references = %+v, want pod owner", secret.OwnerReferences)
	}
}

func TestBuildWorkloadContextUsesLocalConfiguration(t *testing.T) {
	env := discoverymodel.NewEnvironment()
	meshConfig := mesh.DefaultMeshConfig()
	meshConfig.DefaultConfig.DiscoveryAddress = "dubbod.dubbo-system.svc:26012"
	env.Watcher = meshwatcher.ConfigAdapter(krt.NewStatic(&meshwatcher.MeshConfigResource{MeshConfig: meshConfig}, true))
	controller := &inherentXDSWorkloadController{
		server: &Server{environment: env, clusterID: "Kubernetes"},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: "app"},
		Spec:       corev1.PodSpec{ServiceAccountName: "nginx"},
		Status:     corev1.PodStatus{PodIP: "10.0.0.1"},
	}
	workload, err := controller.buildWorkloadContext(pod)
	if err != nil {
		t.Fatalf("buildWorkloadContext() failed: %v", err)
	}
	if workload.clusterID != "Kubernetes" {
		t.Fatalf("clusterID = %q, want Kubernetes", workload.clusterID)
	}
	if workload.discoveryAddress != meshConfig.DefaultConfig.DiscoveryAddress || workload.caAddress != workload.discoveryAddress {
		t.Fatalf("workload addresses = %q/%q, want local discovery address", workload.discoveryAddress, workload.caAddress)
	}
}

func TestInherentXDSWorkloadConfigNeedsUpdate(t *testing.T) {
	tests := []struct {
		name string
		req  *discoverymodel.ConfigChange
		want bool
	}{
		{
			name: "full push",
			req:  &discoverymodel.ConfigChange{Full: true},
			want: true,
		},
		{
			name: "httproute",
			req: &discoverymodel.ConfigChange{
				ConfigsUpdated: sets.New(discoverymodel.ConfigKey{Kind: kind.HTTPRoute, Name: "provider-routing", Namespace: "grpc-app"}),
			},
			want: true,
		},
		{
			name: "fault injection policy",
			req: &discoverymodel.ConfigChange{
				ConfigsUpdated: sets.New(discoverymodel.ConfigKey{Kind: kind.FaultInjectionPolicy, Name: "provider-fault", Namespace: "grpc-app"}),
			},
			want: true,
		},
		{
			name: "peerauthentication",
			req: &discoverymodel.ConfigChange{
				ConfigsUpdated: sets.New(discoverymodel.ConfigKey{Kind: kind.PeerAuthentication, Name: "strict", Namespace: "grpc-app"}),
			},
			want: true,
		},
		{
			name: "requestauthentication",
			req: &discoverymodel.ConfigChange{
				ConfigsUpdated: sets.New(discoverymodel.ConfigKey{Kind: kind.RequestAuthentication, Name: "jwt", Namespace: "grpc-app"}),
			},
			want: true,
		},
		{
			name: "authorizationpolicy",
			req: &discoverymodel.ConfigChange{
				ConfigsUpdated: sets.New(discoverymodel.ConfigKey{Kind: kind.AuthorizationPolicy, Name: "require-jwt", Namespace: "grpc-app"}),
			},
			want: true,
		},
		{
			name: "unrelated configmap",
			req: &discoverymodel.ConfigChange{
				ConfigsUpdated: sets.New(discoverymodel.ConfigKey{Kind: kind.ConfigMap, Name: "ui", Namespace: "dubbo-system"}),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inherentXDSWorkloadConfigNeedsUpdate(tt.req); got != tt.want {
				t.Fatalf("inherentXDSWorkloadConfigNeedsUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleWorkloadConfigUpdateEnqueuesManagedPods(t *testing.T) {
	managed := func(name, namespace string) *corev1.Pod {
		return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				inject.InherentInjectTemplatesAnnoName: inject.InherentTripleTemplateName,
			},
		}}
	}

	reconciled := make(chan types.NamespacedName, 3)
	controller := &inherentXDSWorkloadController{
		managedPods: staticInherentPodIndexer{items: []any{
			managed("zeta", "team-b"),
			"not-a-pod",
			nil,
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "unmanaged", Namespace: "team-a"}},
			managed("alpha", "team-a"),
		}},
	}
	controller.queue = controllers.NewQueue("Inherent runtime config test",
		controllers.WithReconciler(func(key types.NamespacedName) error {
			reconciled <- key
			return nil
		}),
		controllers.WithMaxAttempts(1),
	)

	controller.handleWorkloadConfigUpdate(&discoverymodel.ConfigChange{
		ConfigsUpdated: sets.New(discoverymodel.ConfigKey{Kind: kind.ConfigMap, Name: "unrelated"}),
	})
	controller.handleWorkloadConfigUpdate(&discoverymodel.ConfigChange{
		ConfigsUpdated: sets.New(discoverymodel.ConfigKey{Kind: kind.Service, Name: "provider", Namespace: "team-a"}),
	})

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		controller.queue.Run(stop)
		close(done)
	}()
	t.Cleanup(func() {
		close(stop)
		<-done
	})

	want := []types.NamespacedName{
		{Name: "alpha", Namespace: "team-a"},
		{Name: "zeta", Namespace: "team-b"},
	}
	for i, expected := range want {
		select {
		case got := <-reconciled:
			if got != expected {
				t.Fatalf("reconciled key %d = %v, want %v", i, got, expected)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for reconciled key %d", i)
		}
	}
	select {
	case got := <-reconciled:
		t.Fatalf("unexpected reconciled key %v from unrelated update", got)
	case <-time.After(50 * time.Millisecond):
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

type staticInherentPodIndexer struct {
	items []any
}

func (s staticInherentPodIndexer) Lookup(key string) []any {
	if key != inherentXDSManagedPodIndexKey {
		return nil
	}
	return s.items
}

func TestScheduleRotationTracksEarliestTimer(t *testing.T) {
	now := time.Now()
	controller := &inherentXDSWorkloadController{
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

type credentialsTestAuthority struct {
	ra.RegistrationAuthority
	root []byte
}

func (a credentialsTestAuthority) GetCAKeyCertBundle() *pkiutil.KeyCertBundle {
	return pkiutil.NewKeyCertBundleFromPem(nil, nil, nil, a.root, nil)
}

func TestTripleReconcilePreservesWorkloadCredentials(t *testing.T) {
	pod, original := testWorkloadSecret(t, "spiffe://cluster.local/ns/app/sa/default")
	pod.Annotations = map[string]string{inject.InherentInjectTemplatesAnnoName: inject.InherentTripleTemplateName}
	pod.Status.PodIP = "10.0.0.1"
	client := credentialsTestClient{core: fake.NewClientset(pod, original)}
	env := discoverymodel.NewEnvironment()
	meshConfig := mesh.DefaultMeshConfig()
	meshConfig.DefaultConfig.DiscoveryAddress = "dubbod.dubbo-system.svc:26012"
	env.Watcher = meshwatcher.ConfigAdapter(krt.NewStatic(&meshwatcher.MeshConfigResource{MeshConfig: meshConfig}, true))
	controller := &inherentXDSWorkloadController{
		server: &Server{environment: env, clusterID: "Kubernetes", RA: credentialsTestAuthority{root: original.Data[constants.CACertNamespaceConfigMapDataName]}},
		client: client, pods: credentialsTestPods{pod: pod},
		rotations: make(map[types.NamespacedName]time.Time),
	}
	defer controller.stopAllTimers()
	key := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
	if err := controller.reconcile(key); err != nil {
		t.Fatal(err)
	}
	secret, err := client.Kube().CoreV1().Secrets(pod.Namespace).Get(context.Background(), original.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, filename := range []string{constants.CertChainFilename, constants.CACertNamespaceConfigMapDataName} {
		if !bytes.Equal(secret.Data[filename], original.Data[filename]) {
			t.Fatalf("workload credential changed: %s", filename)
		}
	}
	oldBlock, _ := pem.Decode(original.Data[constants.KeyFilename])
	oldKey, err := x509.ParseECPrivateKey(oldBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	newBlock, _ := pem.Decode(secret.Data[constants.KeyFilename])
	if newBlock == nil || newBlock.Type != "PRIVATE KEY" {
		t.Fatal("expected PKCS#8 workload key")
	}
	newKey, err := x509.ParsePKCS8PrivateKey(newBlock.Bytes)
	if err != nil || !oldKey.Equal(newKey) {
		t.Fatalf("workload private key changed: %v", err)
	}
	var bootstrap xdsbootstrap.Bootstrap
	if err := json.Unmarshal(secret.Data[inject.InherentTripleBootstrapFileName], &bootstrap); err != nil {
		t.Fatal(err)
	}
	if bootstrap.Node.GetId() != "inherent~10.0.0.1~edge.app~app.svc.cluster.local" || len(bootstrap.XDSServers) != 1 || bootstrap.XDSServers[0].ServerURI != meshConfig.DefaultConfig.DiscoveryAddress {
		t.Fatalf("unexpected Triple bootstrap: %+v", bootstrap)
	}
	if len(secret.Data) != 4 || secret.OwnerReferences[0].UID != pod.UID {
		t.Fatal("workload Secret contract changed")
	}
	if controller.rotations[key].IsZero() {
		t.Fatal("certificate rotation was not scheduled")
	}
	actions := len(client.Kube().(*fake.Clientset).Actions())
	if err := controller.reconcile(key); err != nil {
		t.Fatal(err)
	}
	for _, action := range client.Kube().(*fake.Clientset).Actions()[actions:] {
		if action.GetVerb() != "get" {
			t.Fatalf("unchanged credentials were rewritten: %s", action.GetVerb())
		}
	}
}
