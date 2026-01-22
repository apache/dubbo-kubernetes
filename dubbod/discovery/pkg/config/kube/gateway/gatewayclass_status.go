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
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model/kstatus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func GetClassStatus(existing *gatewayv1.GatewayClassStatus, gen int64) *gatewayv1.GatewayClassStatus {
	if existing == nil {
		existing = &gatewayv1.GatewayClassStatus{}
	}
	existing.Conditions = kstatus.UpdateConditionIfChanged(existing.Conditions, metav1.Condition{
		Type:               string(gatewayv1.GatewayClassConditionStatusAccepted),
		Status:             kstatus.StatusTrue,
		ObservedGeneration: gen,
		LastTransitionTime: metav1.Now(),
		Reason:             string(gatewayv1.GatewayClassConditionStatusAccepted),
		Message:            "Handled by Dubbo controller",
	})
	return existing
}
