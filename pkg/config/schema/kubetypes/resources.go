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

package kubetypes

import (
	dubboapimeshv1alpha1 "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	apiorgapachedubboapinetworkingv1alpha3 "github.com/apache/dubbo-kubernetes/client-go/pkg/apis/networking/v1alpha3"
	orgapachedubboapisecurityv1alpha3 "github.com/apache/dubbo-kubernetes/client-go/pkg/apis/security/v1alpha3"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	k8sioapiadmissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	k8sioapiappsv1 "k8s.io/api/apps/v1"
	k8sioapicorev1 "k8s.io/api/core/v1"
	k8sioapidiscoveryv1 "k8s.io/api/discovery/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	sigsk8siogatewayapiapisv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func getGvk(obj any) (config.GroupVersionKind, bool) {
	switch obj.(type) {
	case *k8sioapiadmissionregistrationv1.MutatingWebhookConfiguration:
		return gvk.MutatingWebhookConfiguration, true
	case *k8sioapiadmissionregistrationv1.ValidatingWebhookConfiguration:
		return gvk.ValidatingWebhookConfiguration, true
	case *dubboapimeshv1alpha1.MeshGlobalConfig:
		return gvk.MeshGlobalConfig, true
	case *orgapachedubboapisecurityv1alpha3.PeerAuthentication:
		return gvk.PeerAuthentication, true
	case *apiorgapachedubboapinetworkingv1alpha3.DestinationRule:
		return gvk.DestinationRule, true
	case *apiorgapachedubboapinetworkingv1alpha3.VirtualService:
		return gvk.VirtualService, true
	case *k8sioapicorev1.ConfigMap:
		return gvk.ConfigMap, true
	case *k8sioapicorev1.Endpoints:
		return gvk.Endpoints, true
	case *k8sioapidiscoveryv1.EndpointSlice:
		return gvk.EndpointSlice, true
	case *k8sioapicorev1.Service:
		return gvk.Service, true
	case *k8sioapicorev1.Pod:
		return gvk.Pod, true
	case *k8sioapicorev1.Namespace:
		return gvk.Namespace, true
	case *k8sioapicorev1.ServiceAccount:
		return gvk.ServiceAccount, true
	case *k8sioapiappsv1.Deployment:
		return gvk.Deployment, true
	case *gatewayv1.GatewayClass:
		return gvk.GatewayClass, true
	case *gatewayv1.Gateway:
		return gvk.Gateway, true
	case *sigsk8siogatewayapiapisv1.HTTPRoute:
		return gvk.HTTPRoute, true
	default:
		return config.GroupVersionKind{}, false
	}
}
