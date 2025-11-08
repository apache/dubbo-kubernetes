//go:build !agent
// +build !agent

/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package collections

import (
	"reflect"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	istioioapimetav1alpha1 "istio.io/api/meta/v1alpha1"
	istioioapinetworkingv1alpha3 "istio.io/api/networking/v1alpha3"
	istioioapisecurityv1beta1 "istio.io/api/security/v1beta1"
	k8sioapiadmissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	k8sioapicorev1 "k8s.io/api/core/v1"
	k8sioapidiscoveryv1 "k8s.io/api/discovery/v1"
)

var (
	PeerAuthentication = collection.Builder{
		Identifier: "PeerAuthentication",
		Group:      "security.dubbo.apache.org",
		Kind:       "PeerAuthentication",
		Plural:     "peerauthentications",
		Version:    "v1",
		VersionAliases: []string{
			"v1beta1",
		},
		Proto: "istio.security.v1beta1.PeerAuthentication", StatusProto: "istio.meta.v1alpha1.IstioStatus",
		ReflectType: reflect.TypeOf(&istioioapisecurityv1beta1.PeerAuthentication{}).Elem(), StatusType: reflect.TypeOf(&istioioapimetav1alpha1.IstioStatus{}).Elem(),
		ProtoPackage: "istio.io/api/security/v1beta1", StatusPackage: "istio.io/api/meta/v1alpha1",
		ClusterScoped: false,
		Synthetic:     false,
		Builtin:       false,
	}.MustBuild()
	RequestAuthentication = collection.Builder{
		Identifier: "RequestAuthentication",
		Group:      "security.dubbo.apache.org",
		Kind:       "RequestAuthentication",
		Plural:     "requestauthentications",
		Version:    "v1",
		VersionAliases: []string{
			"v1beta1",
		},
		Proto: "istio.security.v1beta1.RequestAuthentication", StatusProto: "istio.meta.v1alpha1.IstioStatus",
		ReflectType: reflect.TypeOf(&istioioapisecurityv1beta1.RequestAuthentication{}).Elem(), StatusType: reflect.TypeOf(&istioioapimetav1alpha1.IstioStatus{}).Elem(),
		ProtoPackage: "istio.io/api/security/v1beta1", StatusPackage: "istio.io/api/meta/v1alpha1",
		ClusterScoped: false,
		Synthetic:     false,
		Builtin:       false,
	}.MustBuild()
	DestinationRule = collection.Builder{
		Identifier: "DestinationRule",
		Group:      "networking.dubbo.apache.org",
		Kind:       "DestinationRule",
		Plural:     "destinationrules",
		Version:    "v1",
		VersionAliases: []string{
			"v1alpha3",
			"v1beta1",
		},
		Proto: "istio.networking.v1alpha3.DestinationRule", StatusProto: "istio.meta.v1alpha1.IstioStatus",
		ReflectType: reflect.TypeOf(&istioioapinetworkingv1alpha3.DestinationRule{}).Elem(), StatusType: reflect.TypeOf(&istioioapimetav1alpha1.IstioStatus{}).Elem(),
		ProtoPackage: "istio.io/api/networking/v1alpha3", StatusPackage: "istio.io/api/meta/v1alpha1",
		ClusterScoped: false,
		Synthetic:     false,
		Builtin:       false,
	}.MustBuild()
	VirtualService = collection.Builder{
		Identifier: "VirtualService",
		Group:      "networking.dubbo.apache.org",
		Kind:       "VirtualService",
		Plural:     "virtualservices",
		Version:    "v1",
		VersionAliases: []string{
			"v1alpha3",
			"v1beta1",
		},
		Proto: "istio.networking.v1alpha3.VirtualService", StatusProto: "istio.meta.v1alpha1.IstioStatus",
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
	EndpointSlice = collection.Builder{
		Identifier:    "EndpointSlice",
		Group:         "discovery.k8s.io",
		Kind:          "EndpointSlice",
		Plural:        "endpointslices",
		Version:       "v1",
		Proto:         "k8s.io.api.discovery.v1.EndpointSlice",
		ReflectType:   reflect.TypeOf(&k8sioapidiscoveryv1.EndpointSlice{}).Elem(),
		ProtoPackage:  "k8s.io/api/discovery/v1",
		ClusterScoped: false,
		Synthetic:     false,
		Builtin:       true,
	}.MustBuild()

	Endpoints = collection.Builder{
		Identifier:    "Endpoints",
		Group:         "",
		Kind:          "Endpoints",
		Plural:        "endpoints",
		Version:       "v1",
		Proto:         "k8s.io.api.core.v1.Endpoints",
		ReflectType:   reflect.TypeOf(&k8sioapicorev1.Endpoints{}).Elem(),
		ProtoPackage:  "k8s.io/api/core/v1",
		ClusterScoped: false,
		Synthetic:     false,
		Builtin:       true,
	}.MustBuild()
	Service = collection.Builder{
		Identifier: "Service",
		Group:      "",
		Kind:       "Service",
		Plural:     "services",
		Version:    "v1",
		Proto:      "k8s.io.api.core.v1.ServiceSpec", StatusProto: "k8s.io.api.core.v1.ServiceStatus",
		ReflectType: reflect.TypeOf(&k8sioapicorev1.ServiceSpec{}).Elem(), StatusType: reflect.TypeOf(&k8sioapicorev1.ServiceStatus{}).Elem(),
		ProtoPackage: "k8s.io/api/core/v1", StatusPackage: "k8s.io/api/core/v1",
		ClusterScoped: false,
		Synthetic:     false,
		Builtin:       true,
	}.MustBuild()

	All = collection.NewSchemasBuilder().
		MustAdd(PeerAuthentication).
		MustAdd(RequestAuthentication).
		MustAdd(DestinationRule).
		MustAdd(VirtualService).
		MustAdd(EndpointSlice).
		MustAdd(Endpoints).
		MustAdd(Service).
		MustAdd(MutatingWebhookConfiguration).
		MustAdd(ValidatingWebhookConfiguration).
		Build()

	Planet = collection.NewSchemasBuilder().
		MustAdd(PeerAuthentication).
		MustAdd(RequestAuthentication).
		MustAdd(DestinationRule).
		MustAdd(VirtualService).
		Build()
)
