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

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
)

func TestEdsUpdatedServicesForRequestTargetsServiceOnlyPush(t *testing.T) {
	req := &model.PushRequest{
		ConfigsUpdated: sets.New(model.ConfigKey{
			Kind:      kind.Service,
			Name:      "nginx.app.svc.cluster.local",
			Namespace: "app",
		}),
	}

	services, targeted := edsUpdatedServicesForRequest(req)
	if !targeted {
		t.Fatalf("targeted = false, want true")
	}
	if !services.Contains("nginx.app.svc.cluster.local") {
		t.Fatalf("services = %v, want nginx service", services.UnsortedList())
	}
	if !canSendPartialFullPushes(req) {
		t.Fatalf("canSendPartialFullPushes() = false, want true")
	}
}

func TestEdsUpdatedServicesForRequestFallsBackForUntargetedConfig(t *testing.T) {
	req := &model.PushRequest{
		ConfigsUpdated: sets.New(model.ConfigKey{
			Kind:      kind.MeshService,
			Name:      "nginx",
			Namespace: "app",
		}),
	}

	if _, targeted := edsUpdatedServicesForRequest(req); targeted {
		t.Fatalf("targeted = true, want false")
	}
	if canSendPartialFullPushes(req) {
		t.Fatalf("canSendPartialFullPushes() = true, want false")
	}
}
