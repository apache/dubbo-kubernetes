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

package server

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	telemetry "github.com/kdubbo/api/telemetry/v1alpha1"
)

type fakeConfigLister struct {
	items []config.Config
}

func (f fakeConfigLister) List(typ config.GroupVersionKind, namespace string) []config.Config {
	return f.items
}

func TestValidateMeshlevelUniqueness(t *testing.T) {
	existing := config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.Telemetry,
			Name:             "mesh-default",
			Namespace:        "dubbo-system",
		},
		Spec: &telemetry.Telemetry{},
	}
	wh := &Webhook{store: fakeConfigLister{items: []config.Config{existing}}}

	duplicate := config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.Telemetry,
			Name:             "another-default",
			Namespace:        "dubbo-system",
		},
		Spec: &telemetry.Telemetry{},
	}
	if err := wh.validateMeshlevelUniqueness(duplicate); err == nil {
		t.Fatal("validateMeshlevelUniqueness() accepted a second meshlevel Telemetry")
	}
	if err := wh.validateMeshlevelUniqueness(existing); err != nil {
		t.Fatalf("validateMeshlevelUniqueness() rejected update of the same resource: %v", err)
	}
}
