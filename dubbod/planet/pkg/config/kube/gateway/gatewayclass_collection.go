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
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	gateway "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type GatewayClass struct {
	Name       string
	Controller gateway.GatewayController
}

func (g GatewayClass) ResourceName() string {
	return g.Name
}

func getKnownControllerNames() []string {
	names := make([]string, 0, len(classInfos))
	for k := range classInfos {
		names = append(names, string(k))
	}
	return names
}

func GatewayClassesCollection(
	gatewayClasses krt.Collection[*gateway.GatewayClass],
	opts krt.OptionsBuilder,
) (
	krt.StatusCollection[*gateway.GatewayClass, gatewayv1.GatewayClassStatus],
	krt.Collection[GatewayClass],
) {
	return krt.NewStatusCollection(gatewayClasses, func(ctx krt.HandlerContext, obj *gateway.GatewayClass) (*gatewayv1.GatewayClassStatus, *GatewayClass) {
		log.Infof("Processing GatewayClass %s with controller name %q", obj.Name, obj.Spec.ControllerName)
		// Log all known controller names for debugging
		for k := range classInfos {
			log.Infof("Known controller in classInfos: %q", string(k))
		}
		_, known := classInfos[obj.Spec.ControllerName]
		if !known {
			log.Warnf("GatewayClass %s has unknown controller name %q, skipping status update. Known controllers: %v",
				obj.Name, obj.Spec.ControllerName, getKnownControllerNames())
			return nil, nil
		}
		log.Infof("GatewayClass %s has known controller name %q, updating status", obj.Name, obj.Spec.ControllerName)
		status := obj.Status.DeepCopy()
		status = GetClassStatus(status, obj.Generation)
		return status, &GatewayClass{
			Name:       obj.Name,
			Controller: obj.Spec.ControllerName,
		}
	}, opts.WithName("GatewayClasses")...)
}
