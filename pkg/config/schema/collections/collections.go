//go:build !agent
// +build !agent

package collections

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/resource"
	istioioapimetav1alpha1 "istio.io/api/meta/v1alpha1"
	istioioapinetworkingv1alpha3 "istio.io/api/networking/v1alpha3"
	istioioapisecurityv1beta1 "istio.io/api/security/v1beta1"
	"reflect"
)

var (
	DestinationRule = resource.Builder{
		Identifier: "DestinationRule",
		Group:      "networking.istio.io",
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

	PeerAuthentication = resource.Builder{
		Identifier: "PeerAuthentication",
		Group:      "security.istio.io",
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

	RequestAuthentication = resource.Builder{
		Identifier: "RequestAuthentication",
		Group:      "security.istio.io",
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

	VirtualService = resource.Builder{
		Identifier: "VirtualService",
		Group:      "networking.istio.io",
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

	Sail = collection.NewSchemasBuilder().
		MustAdd(PeerAuthentication).
		MustAdd(RequestAuthentication).
		MustAdd(VirtualService).
		MustAdd(DestinationRule).
		Build()
)
