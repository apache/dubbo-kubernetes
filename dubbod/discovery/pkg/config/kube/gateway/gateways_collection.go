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
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model/kstatus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gateway "sigs.k8s.io/gateway-api/apis/v1"
)

type Gateway struct {
	*config.Config `json:"config"`
}

func (g Gateway) ResourceName() string {
	return config.NamespacedName(g.Config).String()
}

func GatewaysCollection(
	gateways krt.Collection[*gateway.Gateway],
	gatewayClasses krt.Collection[GatewayClass],
	opts krt.OptionsBuilder,
) (
	krt.StatusCollection[*gateway.Gateway, gateway.GatewayStatus],
	krt.Collection[Gateway],
) {
	statusCol, gw := krt.NewStatusManyCollection(gateways, func(ctx krt.HandlerContext, obj *gateway.Gateway) (*gateway.GatewayStatus, []Gateway) {
		result := []Gateway{}
		kgw := obj.Spec
		status := obj.Status.DeepCopy()
		class := fetchClass(ctx, gatewayClasses, kgw.GatewayClassName)
		if class == nil {
			return nil, nil
		}
		controllerName := class.Controller
		classInfo, f := classInfos[controllerName]
		if !f {
			return nil, nil
		}
		if classInfo.disableRouteGeneration {
			// For now we still mark the Gateway as accepted, but let higher layers control route generation.
			status = setGatewayConditions(status, obj.Generation, true, true)
			return status, result
		}

		// Default behavior: GatewayClass is known and managed by this controller.
		// Mark Accepted/Programmed to ensure status no longer stays at "Waiting for controller".
		status = setGatewayConditions(status, obj.Generation, true, true)
		return status, result
	}, opts.WithName("KubernetesGateway")...)

	return statusCol, gw
}

func setGatewayConditions(
	existing *gateway.GatewayStatus,
	gen int64,
	accepted, programmed bool,
) *gateway.GatewayStatus {
	boolToConditionStatus := func(b bool) metav1.ConditionStatus {
		if b {
			return metav1.ConditionTrue
		}
		return metav1.ConditionFalse
	}

	msg := "Handled by Dubbo controller"
	conds := kstatus.UpdateConditionIfChanged(existing.Conditions, metav1.Condition{
		Type:               string(gateway.GatewayConditionAccepted),
		Status:             boolToConditionStatus(accepted),
		ObservedGeneration: gen,
		LastTransitionTime: metav1.Now(),
		Reason:             "Accepted",
		Message:            msg,
	})
	conds = kstatus.UpdateConditionIfChanged(conds, metav1.Condition{
		Type:               string(gateway.GatewayConditionProgrammed),
		Status:             boolToConditionStatus(programmed),
		ObservedGeneration: gen,
		LastTransitionTime: metav1.Now(),
		Reason:             "Programmed",
		Message:            msg,
	})
	existing.Conditions = conds
	return existing
}

func FinalGatewayStatusCollection(
	gatewayStatuses krt.StatusCollection[*gateway.Gateway, gateway.GatewayStatus],
	opts krt.OptionsBuilder,
) krt.StatusCollection[*gateway.Gateway, gateway.GatewayStatus] {
	return krt.NewCollection(
		gatewayStatuses,
		func(ctx krt.HandlerContext, i krt.ObjectWithStatus[*gateway.Gateway, gateway.GatewayStatus]) *krt.ObjectWithStatus[*gateway.Gateway, gateway.GatewayStatus] {
			status := i.Status.DeepCopy()
			return &krt.ObjectWithStatus[*gateway.Gateway, gateway.GatewayStatus]{
				Obj:    i.Obj,
				Status: *status,
			}
		}, opts.WithName("GatewayFinalStatus")...)
}
