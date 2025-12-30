//go:build !agent
// +build !agent

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

package collections

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/validation"
	"reflect"
	sigsk8siogatewayapiapisv1 "sigs.k8s.io/gateway-api/apis/v1"
	sigsk8siogatewayapiapisv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	istioioapimetav1alpha1 "istio.io/api/meta/v1alpha1"
	istioioapinetworkingv1alpha3 "istio.io/api/networking/v1alpha3"
	istioioapisecurityv1beta1 "istio.io/api/security/v1beta1"
	k8sioapiadmissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

var (
	PeerAuthentication = collection.Builder{
		Identifier:     "PeerAuthentication",
		Group:          "security.dubbo.apache.org",
		Kind:           "PeerAuthentication",
		Plural:         "peerauthentications",
		Version:        "v1",
		VersionAliases: []string{},
		Proto:          "istio.security.v1beta1.PeerAuthentication", StatusProto: "istio.meta.v1alpha1.IstioStatus",
		ReflectType: reflect.TypeOf(&istioioapisecurityv1beta1.PeerAuthentication{}).Elem(), StatusType: reflect.TypeOf(&istioioapimetav1alpha1.IstioStatus{}).Elem(),
		ProtoPackage: "istio.io/api/security/v1beta1", StatusPackage: "istio.io/api/meta/v1alpha1",
		ClusterScoped: false,
		Synthetic:     false,
		Builtin:       false,
	}.MustBuild()
	SubsetRule = collection.Builder{
		Identifier:     "SubsetRule",
		Group:          "networking.dubbo.apache.org",
		Kind:           "SubsetRule",
		Plural:         "subsetrules",
		Version:        "v1",
		VersionAliases: []string{},
		Proto:          "istio.networking.v1alpha3.DestinationRule", StatusProto: "istio.meta.v1alpha1.IstioStatus",
		ReflectType: reflect.TypeOf(&istioioapinetworkingv1alpha3.DestinationRule{}).Elem(), StatusType: reflect.TypeOf(&istioioapimetav1alpha1.IstioStatus{}).Elem(),
		ProtoPackage: "istio.io/api/networking/v1alpha3", StatusPackage: "istio.io/api/meta/v1alpha1",
		ClusterScoped: false,
		Synthetic:     false,
		Builtin:       false,
	}.MustBuild()
	ServiceRoute = collection.Builder{
		Identifier:     "ServiceRoute",
		Group:          "networking.dubbo.apache.org",
		Kind:           "ServiceRoute",
		Plural:         "serviceroutes",
		Version:        "v1",
		VersionAliases: []string{},
		Proto:          "istio.networking.v1alpha3.VirtualService", StatusProto: "istio.meta.v1alpha1.IstioStatus",
		ReflectType: reflect.TypeOf(&istioioapinetworkingv1alpha3.VirtualService{}).Elem(), StatusType: reflect.TypeOf(&istioioapimetav1alpha1.IstioStatus{}).Elem(),
		ProtoPackage: "istio.io/api/networking/v1alpha3", StatusPackage: "istio.io/api/meta/v1alpha1",
		ClusterScoped: false,
		Synthetic:     false,
		Builtin:       false,
	}.MustBuild()
	ValidatingWebhookConfiguration = collection.Builder{
		Identifier:    "ValidatingWebhookConfiguration",
		Group:         "admissionregistration.k8s.io",
		Kind:          "ValidatingWebhookConfiguration",
		Plural:        "validatingwebhookconfigurations",
		Version:       "v1",
		Proto:         "k8s.io.api.admissionregistration.v1.ValidatingWebhookConfiguration",
		ReflectType:   reflect.TypeOf(&k8sioapiadmissionregistrationv1.ValidatingWebhookConfiguration{}).Elem(),
		ProtoPackage:  "k8s.io/api/admissionregistration/v1",
		ClusterScoped: true,
		Synthetic:     false,
		Builtin:       true,
	}.MustBuild()
	MutatingWebhookConfiguration = collection.Builder{
		Identifier:    "MutatingWebhookConfiguration",
		Group:         "admissionregistration.k8s.io",
		Kind:          "MutatingWebhookConfiguration",
		Plural:        "mutatingwebhookconfigurations",
		Version:       "v1",
		Proto:         "k8s.io.api.admissionregistration.v1.MutatingWebhookConfiguration",
		ReflectType:   reflect.TypeOf(&k8sioapiadmissionregistrationv1.MutatingWebhookConfiguration{}).Elem(),
		ProtoPackage:  "k8s.io/api/admissionregistration/v1",
		ClusterScoped: true,
		Synthetic:     false,
		Builtin:       true,
	}.MustBuild()
	Gateway = collection.Builder{
		Identifier: "KubernetesGateway",
		Group:      "gateway.networking.k8s.io",
		Kind:       "Gateway",
		Plural:     "gateways",
		Version:    "v1",
		Proto:      "k8s.io.gateway_api.api.v1alpha1.GatewaySpec", StatusProto: "k8s.io.gateway_api.api.v1alpha1.GatewayStatus",
		ReflectType: reflect.TypeOf(&sigsk8siogatewayapiapisv1.GatewaySpec{}).Elem(), StatusType: reflect.TypeOf(&sigsk8siogatewayapiapisv1.GatewayStatus{}).Elem(),
		ProtoPackage: "sigs.k8s.io/gateway-api/apis/v1", StatusPackage: "sigs.k8s.io/gateway-api/apis/v1",
		ClusterScoped: false,
		Synthetic:     false,
		Builtin:       false,
		ValidateProto: validation.EmptyValidate,
	}.MustBuild()
	GatewayClass = collection.Builder{
		Identifier: "GatewayClass",
		Group:      "gateway.networking.k8s.io",
		Kind:       "GatewayClass",
		Plural:     "gatewayclasses",
		Version:    "v1",
		Proto:      "k8s.io.gateway_api.api.v1alpha1.GatewayClassSpec", StatusProto: "k8s.io.gateway_api.api.v1alpha1.GatewayClassStatus",
		ReflectType: reflect.TypeOf(&sigsk8siogatewayapiapisv1.GatewayClassSpec{}).Elem(), StatusType: reflect.TypeOf(&sigsk8siogatewayapiapisv1.GatewayClassStatus{}).Elem(),
		ProtoPackage: "sigs.k8s.io/gateway-api/apis/v1", StatusPackage: "sigs.k8s.io/gateway-api/apis/v1",
		ClusterScoped: true,
		Synthetic:     false,
		Builtin:       false,
		ValidateProto: validation.EmptyValidate,
	}.MustBuild()
	HTTPRoute = collection.Builder{
		Identifier: "HTTPRoute",
		Group:      "gateway.networking.k8s.io",
		Kind:       "HTTPRoute",
		Plural:     "httproutes",
		Version:    "v1",
		Proto:      "k8s.io.gateway_api.api.v1alpha1.HTTPRouteSpec", StatusProto: "k8s.io.gateway_api.api.v1alpha1.HTTPRouteStatus",
		ReflectType: reflect.TypeOf(&sigsk8siogatewayapiapisv1beta1.HTTPRouteSpec{}).Elem(), StatusType: reflect.TypeOf(&sigsk8siogatewayapiapisv1.HTTPRouteStatus{}).Elem(),
		ProtoPackage: "sigs.k8s.io/gateway-api/apis/v1", StatusPackage: "sigs.k8s.io/gateway-api/apis/v1",
		ClusterScoped: false,
		Synthetic:     false,
		Builtin:       false,
		ValidateProto: validation.EmptyValidate,
	}.MustBuild()

	Planet = collection.NewSchemasBuilder().
		MustAdd(PeerAuthentication).
		MustAdd(SubsetRule).
		MustAdd(ServiceRoute).
		Build()

	planetGatewayAPI = collection.NewSchemasBuilder().
				MustAdd(GatewayClass).
				MustAdd(Gateway).
				MustAdd(HTTPRoute).
				Build()

	All = collection.NewSchemasBuilder().
		MustAdd(PeerAuthentication).
		MustAdd(SubsetRule).
		MustAdd(ServiceRoute).
		MustAdd(MutatingWebhookConfiguration).
		MustAdd(ValidatingWebhookConfiguration).
		MustAdd(GatewayClass).
		MustAdd(Gateway).
		MustAdd(HTTPRoute).
		Build()
)
