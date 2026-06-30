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

package crdclient

import (
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestRegisterEventHandlerPersistsRegistration(t *testing.T) {
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })

	routes := krt.NewStaticCollection[config.Config](nil, nil, krt.WithStop(stop))
	cl := &Client{
		kinds: map[config.GroupVersionKind]nsStore{
			gvk.HTTPRoute: {
				collection: routes,
				index:      krt.NewNamespaceIndex(routes),
			},
		},
	}
	events := make(chan model.Event, 1)
	cl.RegisterEventHandler(gvk.HTTPRoute, func(_, _ config.Config, event model.Event) {
		events <- event
	})

	if got := len(cl.kinds[gvk.HTTPRoute].handlers); got != 1 {
		t.Fatalf("registered handlers = %d, want 1", got)
	}

	routes.UpdateObject(config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.HTTPRoute,
			Name:             "orders",
			Namespace:        "app",
		},
		Spec: &gatewayv1.HTTPRouteSpec{},
	})
	expectEvent(t, events, model.EventAdd)
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
