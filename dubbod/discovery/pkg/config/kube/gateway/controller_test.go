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

package gateway

import (
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/features"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func TestGatewayControllerConfigStore(t *testing.T) {
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })

	gatewayConfigs := krt.NewStaticCollection[config.Config](nil, []config.Config{
		{
			Meta: config.Meta{
				GroupVersionKind: gvk.KubernetesGateway,
				Name:             "public",
				Namespace:        "app",
			},
			Spec: &gatewayv1.GatewaySpec{},
		},
	}, krt.WithStop(stop))
	httpRouteConfigs := krt.NewStaticCollection[config.Config](nil, []config.Config{
		{
			Meta: config.Meta{
				GroupVersionKind: gvk.HTTPRoute,
				Name:             "orders",
				Namespace:        "app",
			},
			Spec: &gatewayv1.HTTPRouteSpec{},
		},
		{
			Meta: config.Meta{
				GroupVersionKind: gvk.HTTPRoute,
				Name:             "admin",
				Namespace:        "ops",
			},
			Spec: &gatewayv1.HTTPRouteSpec{},
		},
	}, krt.WithStop(stop))
	c := &Controller{
		data: map[config.GroupVersionKind]kindStore{
			gvk.KubernetesGateway: newKindStore(gatewayConfigs),
			gvk.HTTPRoute:         newKindStore(httpRouteConfigs),
		},
	}

	if _, ok := c.Schemas().FindByGroupVersionKind(gvk.KubernetesGateway); !ok {
		t.Fatal("schemas missing KubernetesGateway")
	}
	if _, ok := c.Schemas().FindByGroupVersionKind(gvk.HTTPRoute); !ok {
		t.Fatal("schemas missing HTTPRoute")
	}

	if got := c.Get(gvk.KubernetesGateway, "public", "app"); got == nil || got.Name != "public" {
		t.Fatalf("gateway Get() = %#v, want public", got)
	}
	if got := c.List(gvk.HTTPRoute, "app"); len(got) != 1 || got[0].Name != "orders" {
		t.Fatalf("HTTPRoute namespace List() = %#v, want only app/orders", got)
	}
	if got := c.List(gvk.HTTPRoute, metav1.NamespaceAll); len(got) != 2 {
		t.Fatalf("HTTPRoute all-namespaces List() = %d, want 2", len(got))
	}
}

func TestGatewayControllerRegisterEventHandler(t *testing.T) {
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })

	routes := krt.NewStaticCollection[config.Config](nil, nil, krt.WithStop(stop))
	c := &Controller{
		data: map[config.GroupVersionKind]kindStore{
			gvk.HTTPRoute: newKindStore(routes),
		},
	}
	events := make(chan model.Event, 1)
	c.RegisterEventHandler(gvk.HTTPRoute, func(_, _ config.Config, event model.Event) {
		events <- event
	})

	routes.UpdateObject(config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.HTTPRoute,
			Name:             "orders",
			Namespace:        "app",
		},
		Spec: &gatewayv1.HTTPRouteSpec{},
	})
	expectEvent(t, events, model.EventAdd)

	routes.UpdateObject(config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.HTTPRoute,
			Name:             "orders",
			Namespace:        "app",
			Generation:       2,
		},
		Spec: &gatewayv1.HTTPRouteSpec{},
	})
	expectEvent(t, events, model.EventUpdate)

	routes.DeleteObject("app/orders")
	expectEvent(t, events, model.EventDelete)
}

func TestGatewayControllerSecretAllowed(t *testing.T) {
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })
	opts := krt.NewOptionsBuilder(stop, "test", nil)

	secretName := gatewayv1.ObjectName("tls-cert")
	grants := krt.NewStaticCollection[*gatewayv1beta1.ReferenceGrant](nil, []*gatewayv1beta1.ReferenceGrant{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "allow-specific",
				Namespace: "certs",
			},
			Spec: gatewayv1beta1.ReferenceGrantSpec{
				From: []gatewayv1beta1.ReferenceGrantFrom{{
					Group:     gatewayv1.Group(gvk.KubernetesGateway.Group),
					Kind:      gatewayv1.Kind(gvk.KubernetesGateway.Kind),
					Namespace: "ingress",
				}},
				To: []gatewayv1beta1.ReferenceGrantTo{{
					Group: gatewayv1.Group(gvk.Secret.Group),
					Kind:  gatewayv1.Kind(gvk.Secret.Kind),
					Name:  &secretName,
				}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "allow-all",
				Namespace: "shared-certs",
			},
			Spec: gatewayv1beta1.ReferenceGrantSpec{
				From: []gatewayv1beta1.ReferenceGrantFrom{{
					Group:     gatewayv1.Group(gvk.KubernetesGateway.Group),
					Kind:      gatewayv1.Kind(gvk.KubernetesGateway.Kind),
					Namespace: "ingress",
				}},
				To: []gatewayv1beta1.ReferenceGrantTo{{
					Group: gatewayv1.Group(gvk.Secret.Group),
					Kind:  gatewayv1.Kind(gvk.Secret.Kind),
				}},
			},
		},
	}, krt.WithStop(stop))
	referenceGrants := newReferenceGrantStore(ReferenceGrantsCollection(grants, opts))
	waitSynced(t, referenceGrants.Collection)
	c := &Controller{
		outputs: Outputs{
			ReferenceGrants: referenceGrants,
		},
	}

	if !c.SecretAllowed(gvk.KubernetesGateway, "kubernetes-gateway://certs/tls-cert", "ingress") {
		t.Fatal("SecretAllowed() = false, want true for specifically granted Secret")
	}
	if !c.SecretAllowed(gvk.KubernetesGateway, "kubernetes-gateway://shared-certs/any-cert", "ingress") {
		t.Fatal("SecretAllowed() = false, want true for wildcard Secret grant")
	}
	if c.SecretAllowed(gvk.KubernetesGateway, "kubernetes-gateway://certs/other-cert", "ingress") {
		t.Fatal("SecretAllowed() = true, want false for ungranted Secret name")
	}
	if c.SecretAllowed(gvk.KubernetesGateway, "kubernetes-gateway://certs/tls-cert", "other") {
		t.Fatal("SecretAllowed() = true, want false for ungranted source namespace")
	}
	if c.SecretAllowed(gvk.HTTPRoute, "kubernetes-gateway://certs/tls-cert", "ingress") {
		t.Fatal("SecretAllowed() = true, want false for ungranted source kind")
	}
}

func TestGatewayControllerHasSyncedChecksInputCollections(t *testing.T) {
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })

	services := krt.NewStaticCollection[*corev1.Service](unsyncedSyncer{}, nil, krt.WithStop(stop))
	routes := krt.NewStaticCollection[config.Config](nil, nil, krt.WithStop(stop))
	c := &Controller{
		inputs: Inputs{
			Services: services,
		},
		data: map[config.GroupVersionKind]kindStore{
			gvk.HTTPRoute: newKindStore(routes),
		},
	}

	if c.HasSynced() {
		t.Fatal("HasSynced() = true, want false while input collection is still syncing")
	}
}

func TestGatewaysCollectionEmitsManagedGatewayConfig(t *testing.T) {
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })
	opts := krt.NewOptionsBuilder(stop, "test", nil)

	gateways := krt.NewStaticCollection[*gatewayv1.Gateway](nil, []*gatewayv1.Gateway{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "public",
				Namespace: "app",
			},
			Spec: gatewayv1.GatewaySpec{
				GatewayClassName: gatewayv1.ObjectName(features.GatewayAPIDefaultGatewayClass),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foreign",
				Namespace: "app",
			},
			Spec: gatewayv1.GatewaySpec{
				GatewayClassName: "external",
			},
		},
	}, krt.WithStop(stop))
	gatewayClasses := krt.NewStaticCollection[GatewayClass](nil, nil, krt.WithStop(stop))

	statuses, configs := GatewaysCollection(gateways, gatewayClasses, opts)
	waitSynced(t, configs)
	waitSynced(t, statuses)

	got := configs.List()
	if len(got) != 1 {
		t.Fatalf("gateway configs = %#v, want only managed gateway", got)
	}
	cfg := got[0]
	if cfg.GroupVersionKind != gvk.KubernetesGateway || cfg.Name != "public" || cfg.Namespace != "app" {
		t.Fatalf("gateway config meta = %#v, want app/public KubernetesGateway", cfg.Meta)
	}
	if _, ok := cfg.Spec.(*gatewayv1.GatewaySpec); !ok {
		t.Fatalf("gateway config spec type = %T, want *gatewayv1.GatewaySpec", cfg.Spec)
	}

	statusList := statuses.List()
	if len(statusList) != 1 {
		t.Fatalf("gateway statuses = %#v, want only managed gateway status", statusList)
	}
	status := statusList[0].Status
	if !hasGatewayCondition(status, string(gatewayv1.GatewayConditionAccepted), metav1.ConditionTrue) {
		t.Fatalf("gateway status missing Accepted=True: %#v", status.Conditions)
	}
	if !hasGatewayCondition(status, string(gatewayv1.GatewayConditionProgrammed), metav1.ConditionTrue) {
		t.Fatalf("gateway status missing Programmed=True: %#v", status.Conditions)
	}
}

func TestConvertHTTPRouteToConfigCopiesSpec(t *testing.T) {
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "orders",
			Namespace: "app",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Hostnames: []gatewayv1.Hostname{"orders.example.com"},
		},
	}
	cfg := convertHTTPRouteToConfig(route)
	route.Spec.Hostnames[0] = "mutated.example.com"

	spec, ok := cfg.Spec.(*gatewayv1.HTTPRouteSpec)
	if !ok {
		t.Fatalf("HTTPRoute config spec type = %T, want *gatewayv1.HTTPRouteSpec", cfg.Spec)
	}
	if got := spec.Hostnames[0]; got != "orders.example.com" {
		t.Fatalf("HTTPRoute config hostname = %q, want deep-copied value", got)
	}
}

func expectEvent(t *testing.T, events <-chan model.Event, want model.Event) {
	t.Helper()
	select {
	case got := <-events:
		if got != want {
			t.Fatalf("event = %v, want %v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("expected event handler to receive %v event", want)
	}
}

func waitSynced(t *testing.T, syncer krt.Syncer) {
	t.Helper()
	timeout := make(chan struct{})
	timer := time.AfterFunc(2*time.Second, func() { close(timeout) })
	defer timer.Stop()
	if !syncer.WaitUntilSynced(timeout) {
		t.Fatal("collection did not sync before timeout")
	}
}

func hasGatewayCondition(status gatewayv1.GatewayStatus, conditionType string, conditionStatus metav1.ConditionStatus) bool {
	for _, condition := range status.Conditions {
		if condition.Type == conditionType && condition.Status == conditionStatus {
			return true
		}
	}
	return false
}

type unsyncedSyncer struct{}

func (unsyncedSyncer) HasSynced() bool {
	return false
}

func (unsyncedSyncer) WaitUntilSynced(<-chan struct{}) bool {
	return false
}
