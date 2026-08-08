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

package activation

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestScaledObjectReady(t *testing.T) {
	object := &unstructured.Unstructured{Object: map[string]any{
		"status": map[string]any{
			"conditions": []any{
				map[string]any{"type": "Ready", "status": "True"},
			},
		},
	}}
	if !scaledObjectReady(object) {
		t.Fatal("ScaledObject Ready=True was not recognized")
	}

	object.Object["status"] = map[string]any{
		"conditions": []any{map[string]any{"type": "Ready", "status": "False"}},
	}
	if scaledObjectReady(object) {
		t.Fatal("ScaledObject Ready=False was reported ready")
	}
}

func TestGatewayProgrammedRequiresCurrentGeneration(t *testing.T) {
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Generation: 4},
		Status: gatewayv1.GatewayStatus{Conditions: []metav1.Condition{{
			Type:               string(gatewayv1.GatewayConditionProgrammed),
			Status:             metav1.ConditionTrue,
			ObservedGeneration: 3,
		}}},
	}
	if gatewayProgrammed(gateway) {
		t.Fatal("stale Programmed=True condition was reported ready")
	}
	gateway.Status.Conditions[0].ObservedGeneration = 4
	if !gatewayProgrammed(gateway) {
		t.Fatal("current Programmed=True condition was not recognized")
	}
}
