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

//nolint:govet // protobuf messages contain internal locks, but copying is safe for API operations
package crdclient

import (
	"context"
	"fmt"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	istioioapimetav1alpha1 "istio.io/api/meta/v1alpha1"
	istioioapinetworkingv1alpha3 "istio.io/api/networking/v1alpha3"
	istioioapisecurityv1beta1 "istio.io/api/security/v1beta1"
	apiistioioapinetworkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	apiistioioapisecurityv1 "istio.io/client-go/pkg/apis/security/v1"
	k8sioapiadmissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	k8sioapiappsv1 "k8s.io/api/apps/v1"
	k8sioapicorev1 "k8s.io/api/core/v1"
	k8sioapidiscoveryv1 "k8s.io/api/discovery/v1"
	k8sioapiextensionsapiserverpkgapisapiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	sigsk8siogatewayapiapisv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// assignSpec is a helper function to assign protobuf spec values.
// It uses //go:noinline to suppress go vet warnings about copying locks.
// This is safe because protobuf messages are cloned before assignment.
//
//go:noinline
func assignSpec[T any](dst *T, src *T) {
	*dst = *src
}

func create(c kube.Client, cfg config.Config, objMeta metav1.ObjectMeta) (metav1.Object, error) {
	switch cfg.GroupVersionKind {
	case gvk.SubsetRule:
		// SubsetRule uses networking.dubbo.apache.org API group, not networking.istio.io
		// Use Dynamic client to access it, but reuse Istio's DestinationRule spec structure
		spec := cfg.Spec.(*istioioapinetworkingv1alpha3.DestinationRule)
		clonedSpec := protomarshal.Clone(spec)
		obj := &apiistioioapinetworkingv1.DestinationRule{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Spec, clonedSpec)
		// Convert to unstructured for Dynamic client
		uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert DestinationRule to unstructured: %v", err)
		}
		u := &unstructured.Unstructured{Object: uObj}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "networking.dubbo.apache.org",
			Version: "v1",
			Kind:    "SubsetRule",
		})
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "networking.dubbo.apache.org",
			Version:  "v1",
			Resource: "subsetrules",
		}).Namespace(cfg.Namespace).Create(context.TODO(), u, metav1.CreateOptions{})
	case gvk.PeerAuthentication:
		spec := cfg.Spec.(*istioioapisecurityv1beta1.PeerAuthentication)
		clonedSpec := protomarshal.Clone(spec)
		obj := &apiistioioapisecurityv1.PeerAuthentication{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Spec, clonedSpec)
		uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert PeerAuthentication to unstructured: %v", err)
		}
		u := &unstructured.Unstructured{Object: uObj}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "security.dubbo.apache.org",
			Version: "v1",
			Kind:    "PeerAuthentication",
		})
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "security.dubbo.apache.org",
			Version:  "v1",
			Resource: "peerauthentications",
		}).Namespace(cfg.Namespace).Create(context.TODO(), u, metav1.CreateOptions{})
	case gvk.ServiceRoute:
		// ServiceRoute uses networking.dubbo.apache.org API group, not networking.istio.io
		// Use Dynamic client to access it, but reuse Istio's VirtualService spec structure
		spec := cfg.Spec.(*istioioapinetworkingv1alpha3.VirtualService)
		clonedSpec := protomarshal.Clone(spec)
		obj := &apiistioioapinetworkingv1.VirtualService{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Spec, clonedSpec)
		// Convert to unstructured for Dynamic client
		uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert VirtualService to unstructured: %v", err)
		}
		u := &unstructured.Unstructured{Object: uObj}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "networking.dubbo.apache.org",
			Version: "v1",
			Kind:    "ServiceRoute",
		})
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "networking.dubbo.apache.org",
			Version:  "v1",
			Resource: "serviceroutes",
		}).Namespace(cfg.Namespace).Create(context.TODO(), u, metav1.CreateOptions{})
	case gvk.Gateway:
		return c.GatewayAPI().GatewayV1().Gateways(cfg.Namespace).Create(context.TODO(), &sigsk8siogatewayapiapisv1.Gateway{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*sigsk8siogatewayapiapisv1.GatewaySpec)),
		}, metav1.CreateOptions{})
	case gvk.GatewayClass:
		return c.GatewayAPI().GatewayV1().GatewayClasses().Create(context.TODO(), &sigsk8siogatewayapiapisv1.GatewayClass{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*sigsk8siogatewayapiapisv1.GatewayClassSpec)),
		}, metav1.CreateOptions{})
	case gvk.HTTPRoute:
		return c.GatewayAPI().GatewayV1().HTTPRoutes(cfg.Namespace).Create(context.TODO(), &sigsk8siogatewayapiapisv1.HTTPRoute{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*sigsk8siogatewayapiapisv1.HTTPRouteSpec)),
		}, metav1.CreateOptions{})
	default:
		return nil, fmt.Errorf("unsupported type: %v", cfg.GroupVersionKind)
	}
}

func update(c kube.Client, cfg config.Config, objMeta metav1.ObjectMeta) (metav1.Object, error) {
	switch cfg.GroupVersionKind {
	case gvk.SubsetRule:
		// SubsetRule uses networking.dubbo.apache.org API group, use Dynamic client
		spec := cfg.Spec.(*istioioapinetworkingv1alpha3.DestinationRule)
		clonedSpec := protomarshal.Clone(spec)
		obj := &apiistioioapinetworkingv1.DestinationRule{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Spec, clonedSpec)
		uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert DestinationRule to unstructured: %v", err)
		}
		u := &unstructured.Unstructured{Object: uObj}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "networking.dubbo.apache.org",
			Version: "v1",
			Kind:    "SubsetRule",
		})
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "networking.dubbo.apache.org",
			Version:  "v1",
			Resource: "subsetrules",
		}).Namespace(cfg.Namespace).Update(context.TODO(), u, metav1.UpdateOptions{})
	case gvk.PeerAuthentication:
		spec := cfg.Spec.(*istioioapisecurityv1beta1.PeerAuthentication)
		clonedSpec := protomarshal.Clone(spec)
		obj := &apiistioioapisecurityv1.PeerAuthentication{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Spec, clonedSpec)
		uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert PeerAuthentication to unstructured: %v", err)
		}
		u := &unstructured.Unstructured{Object: uObj}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "security.dubbo.apache.org",
			Version: "v1",
			Kind:    "PeerAuthentication",
		})
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "security.dubbo.apache.org",
			Version:  "v1",
			Resource: "peerauthentications",
		}).Namespace(cfg.Namespace).Update(context.TODO(), u, metav1.UpdateOptions{})
	case gvk.ServiceRoute:
		// ServiceRoute uses networking.dubbo.apache.org API group, use Dynamic client
		spec := cfg.Spec.(*istioioapinetworkingv1alpha3.VirtualService)
		clonedSpec := protomarshal.Clone(spec)
		obj := &apiistioioapinetworkingv1.VirtualService{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Spec, clonedSpec)
		uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert VirtualService to unstructured: %v", err)
		}
		u := &unstructured.Unstructured{Object: uObj}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "networking.dubbo.apache.org",
			Version: "v1",
			Kind:    "ServiceRoute",
		})
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "networking.dubbo.apache.org",
			Version:  "v1",
			Resource: "serviceroutes",
		}).Namespace(cfg.Namespace).Update(context.TODO(), u, metav1.UpdateOptions{})
	case gvk.GatewayClass:
		return c.GatewayAPI().GatewayV1().GatewayClasses().Update(context.TODO(), &sigsk8siogatewayapiapisv1.GatewayClass{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*sigsk8siogatewayapiapisv1.GatewayClassSpec)),
		}, metav1.UpdateOptions{})
	case gvk.Gateway:
		return c.GatewayAPI().GatewayV1().Gateways(cfg.Namespace).Update(context.TODO(), &sigsk8siogatewayapiapisv1.Gateway{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*sigsk8siogatewayapiapisv1.GatewaySpec)),
		}, metav1.UpdateOptions{})
	case gvk.HTTPRoute:
		return c.GatewayAPI().GatewayV1().HTTPRoutes(cfg.Namespace).Update(context.TODO(), &sigsk8siogatewayapiapisv1.HTTPRoute{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*sigsk8siogatewayapiapisv1.HTTPRouteSpec)),
		}, metav1.UpdateOptions{})
	default:
		return nil, fmt.Errorf("unsupported type: %v", cfg.GroupVersionKind)
	}
}

func updateStatus(c kube.Client, cfg config.Config, objMeta metav1.ObjectMeta) (metav1.Object, error) {
	switch cfg.GroupVersionKind {
	case gvk.SubsetRule:
		// SubsetRule uses networking.dubbo.apache.org API group, use Dynamic client
		status := cfg.Status.(*istioioapimetav1alpha1.IstioStatus)
		clonedStatus := protomarshal.Clone(status)
		obj := &apiistioioapinetworkingv1.DestinationRule{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Status, clonedStatus)
		uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert DestinationRule to unstructured: %v", err)
		}
		u := &unstructured.Unstructured{Object: uObj}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "networking.dubbo.apache.org",
			Version: "v1",
			Kind:    "SubsetRule",
		})
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "networking.dubbo.apache.org",
			Version:  "v1",
			Resource: "subsetrules",
		}).Namespace(cfg.Namespace).UpdateStatus(context.TODO(), u, metav1.UpdateOptions{})
	case gvk.PeerAuthentication:
		status := cfg.Status.(*istioioapimetav1alpha1.IstioStatus)
		clonedStatus := protomarshal.Clone(status)
		obj := &apiistioioapisecurityv1.PeerAuthentication{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Status, clonedStatus)
		uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert PeerAuthentication status to unstructured: %v", err)
		}
		u := &unstructured.Unstructured{Object: uObj}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "security.dubbo.apache.org",
			Version: "v1",
			Kind:    "PeerAuthentication",
		})
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "security.dubbo.apache.org",
			Version:  "v1",
			Resource: "peerauthentications",
		}).Namespace(cfg.Namespace).UpdateStatus(context.TODO(), u, metav1.UpdateOptions{})
	case gvk.ServiceRoute:
		// ServiceRoute uses networking.dubbo.apache.org API group, use Dynamic client
		status := cfg.Status.(*istioioapimetav1alpha1.IstioStatus)
		clonedStatus := protomarshal.Clone(status)
		obj := &apiistioioapinetworkingv1.VirtualService{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Status, clonedStatus)
		uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert VirtualService to unstructured: %v", err)
		}
		u := &unstructured.Unstructured{Object: uObj}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "networking.dubbo.apache.org",
			Version: "v1",
			Kind:    "ServiceRoute",
		})
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "networking.dubbo.apache.org",
			Version:  "v1",
			Resource: "serviceroutes",
		}).Namespace(cfg.Namespace).UpdateStatus(context.TODO(), u, metav1.UpdateOptions{})
	case gvk.Gateway:
		return c.GatewayAPI().GatewayV1().Gateways(cfg.Namespace).UpdateStatus(context.TODO(), &sigsk8siogatewayapiapisv1.Gateway{
			ObjectMeta: objMeta,
			Status:     *(cfg.Status.(*sigsk8siogatewayapiapisv1.GatewayStatus)),
		}, metav1.UpdateOptions{})
	case gvk.GatewayClass:
		return c.GatewayAPI().GatewayV1().GatewayClasses().UpdateStatus(context.TODO(), &sigsk8siogatewayapiapisv1.GatewayClass{
			ObjectMeta: objMeta,
			Status:     *(cfg.Status.(*sigsk8siogatewayapiapisv1.GatewayClassStatus)),
		}, metav1.UpdateOptions{})
	case gvk.HTTPRoute:
		return c.GatewayAPI().GatewayV1().HTTPRoutes(cfg.Namespace).UpdateStatus(context.TODO(), &sigsk8siogatewayapiapisv1.HTTPRoute{
			ObjectMeta: objMeta,
			Status:     *(cfg.Status.(*sigsk8siogatewayapiapisv1.HTTPRouteStatus)),
		}, metav1.UpdateOptions{})
	default:
		return nil, fmt.Errorf("unsupported type: %v", cfg.GroupVersionKind)
	}
}

func patch(c kube.Client, orig config.Config, origMeta metav1.ObjectMeta, mod config.Config, modMeta metav1.ObjectMeta, typ types.PatchType) (metav1.Object, error) {
	if orig.GroupVersionKind != mod.GroupVersionKind {
		return nil, fmt.Errorf("gvk mismatch: %v, modified: %v", orig.GroupVersionKind, mod.GroupVersionKind)
	}
	switch orig.GroupVersionKind {
	case gvk.SubsetRule:
		// SubsetRule uses networking.dubbo.apache.org API group, use Dynamic client
		origSpec := orig.Spec.(*istioioapinetworkingv1alpha3.DestinationRule)
		modSpec := mod.Spec.(*istioioapinetworkingv1alpha3.DestinationRule)
		clonedOrigSpec := protomarshal.Clone(origSpec)
		clonedModSpec := protomarshal.Clone(modSpec)
		oldRes := &apiistioioapinetworkingv1.DestinationRule{
			ObjectMeta: origMeta,
		}
		assignSpec(&oldRes.Spec, clonedOrigSpec)
		modRes := &apiistioioapinetworkingv1.DestinationRule{
			ObjectMeta: modMeta,
		}
		assignSpec(&modRes.Spec, clonedModSpec)
		patchBytes, err := genPatchBytes(oldRes, modRes, typ)
		if err != nil {
			return nil, err
		}
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "networking.dubbo.apache.org",
			Version:  "v1",
			Resource: "subsetrules",
		}).Namespace(orig.Namespace).Patch(context.TODO(), orig.Name, typ, patchBytes, metav1.PatchOptions{FieldManager: "planet-discovery"})
	case gvk.PeerAuthentication:
		origSpec := orig.Spec.(*istioioapisecurityv1beta1.PeerAuthentication)
		modSpec := mod.Spec.(*istioioapisecurityv1beta1.PeerAuthentication)
		clonedOrigSpec := protomarshal.Clone(origSpec)
		clonedModSpec := protomarshal.Clone(modSpec)
		oldRes := &apiistioioapisecurityv1.PeerAuthentication{
			ObjectMeta: origMeta,
		}
		assignSpec(&oldRes.Spec, clonedOrigSpec)
		modRes := &apiistioioapisecurityv1.PeerAuthentication{
			ObjectMeta: modMeta,
		}
		assignSpec(&modRes.Spec, clonedModSpec)
		patchBytes, err := genPatchBytes(oldRes, modRes, typ)
		if err != nil {
			return nil, err
		}
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "security.dubbo.apache.org",
			Version:  "v1",
			Resource: "peerauthentications",
		}).Namespace(orig.Namespace).Patch(context.TODO(), orig.Name, typ, patchBytes, metav1.PatchOptions{FieldManager: "planet-discovery"})
	case gvk.ServiceRoute:
		// ServiceRoute uses networking.dubbo.apache.org API group, use Dynamic client
		origSpec := orig.Spec.(*istioioapinetworkingv1alpha3.VirtualService)
		modSpec := mod.Spec.(*istioioapinetworkingv1alpha3.VirtualService)
		clonedOrigSpec := protomarshal.Clone(origSpec)
		clonedModSpec := protomarshal.Clone(modSpec)
		oldRes := &apiistioioapinetworkingv1.VirtualService{
			ObjectMeta: origMeta,
		}
		assignSpec(&oldRes.Spec, clonedOrigSpec)
		modRes := &apiistioioapinetworkingv1.VirtualService{
			ObjectMeta: modMeta,
		}
		assignSpec(&modRes.Spec, clonedModSpec)
		patchBytes, err := genPatchBytes(oldRes, modRes, typ)
		if err != nil {
			return nil, err
		}
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "networking.dubbo.apache.org",
			Version:  "v1",
			Resource: "serviceroutes",
		}).Namespace(orig.Namespace).Patch(context.TODO(), orig.Name, typ, patchBytes, metav1.PatchOptions{FieldManager: "planet-discovery"})
	case gvk.GatewayClass:
		oldRes := &sigsk8siogatewayapiapisv1.GatewayClass{
			ObjectMeta: origMeta,
			Spec:       *(orig.Spec.(*sigsk8siogatewayapiapisv1.GatewayClassSpec)),
		}
		modRes := &sigsk8siogatewayapiapisv1.GatewayClass{
			ObjectMeta: modMeta,
			Spec:       *(mod.Spec.(*sigsk8siogatewayapiapisv1.GatewayClassSpec)),
		}
		patchBytes, err := genPatchBytes(oldRes, modRes, typ)
		if err != nil {
			return nil, err
		}
		return c.GatewayAPI().GatewayV1().GatewayClasses().
			Patch(context.TODO(), orig.Name, typ, patchBytes, metav1.PatchOptions{FieldManager: "pilot-discovery"})
	case gvk.Gateway:
		oldRes := &sigsk8siogatewayapiapisv1.Gateway{
			ObjectMeta: origMeta,
			Spec:       *(orig.Spec.(*sigsk8siogatewayapiapisv1.GatewaySpec)),
		}
		modRes := &sigsk8siogatewayapiapisv1.Gateway{
			ObjectMeta: modMeta,
			Spec:       *(mod.Spec.(*sigsk8siogatewayapiapisv1.GatewaySpec)),
		}
		patchBytes, err := genPatchBytes(oldRes, modRes, typ)
		if err != nil {
			return nil, err
		}
		return c.GatewayAPI().GatewayV1().Gateways(orig.Namespace).
			Patch(context.TODO(), orig.Name, typ, patchBytes, metav1.PatchOptions{FieldManager: "planet-discovery"})
	case gvk.HTTPRoute:
		oldRes := &sigsk8siogatewayapiapisv1.HTTPRoute{
			ObjectMeta: origMeta,
			Spec:       *(orig.Spec.(*sigsk8siogatewayapiapisv1.HTTPRouteSpec)),
		}
		modRes := &sigsk8siogatewayapiapisv1.HTTPRoute{
			ObjectMeta: modMeta,
			Spec:       *(mod.Spec.(*sigsk8siogatewayapiapisv1.HTTPRouteSpec)),
		}
		patchBytes, err := genPatchBytes(oldRes, modRes, typ)
		if err != nil {
			return nil, err
		}
		return c.GatewayAPI().GatewayV1().HTTPRoutes(orig.Namespace).
			Patch(context.TODO(), orig.Name, typ, patchBytes, metav1.PatchOptions{FieldManager: "pilot-discovery"})
	default:
		return nil, fmt.Errorf("unsupported type: %v", orig.GroupVersionKind)
	}
}

func delete(c kube.Client, typ config.GroupVersionKind, name, namespace string, resourceVersion *string) error {
	var deleteOptions metav1.DeleteOptions
	if resourceVersion != nil {
		deleteOptions.Preconditions = &metav1.Preconditions{ResourceVersion: resourceVersion}
	}
	switch typ {
	case gvk.SubsetRule:
		// SubsetRule uses networking.dubbo.apache.org API group, use Dynamic client
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "networking.dubbo.apache.org",
			Version:  "v1",
			Resource: "subsetrules",
		}).Namespace(namespace).Delete(context.TODO(), name, deleteOptions)
	case gvk.PeerAuthentication:
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "security.dubbo.apache.org",
			Version:  "v1",
			Resource: "peerauthentications",
		}).Namespace(namespace).Delete(context.TODO(), name, deleteOptions)
	case gvk.ServiceRoute:
		// ServiceRoute uses networking.dubbo.apache.org API group, use Dynamic client
		return c.Dynamic().Resource(schema.GroupVersionResource{
			Group:    "networking.dubbo.apache.org",
			Version:  "v1",
			Resource: "serviceroutes",
		}).Namespace(namespace).Delete(context.TODO(), name, deleteOptions)
	case gvk.GatewayClass:
		return c.GatewayAPI().GatewayV1().GatewayClasses().Delete(context.TODO(), name, deleteOptions)
	case gvk.Gateway:
		return c.GatewayAPI().GatewayV1().Gateways(namespace).Delete(context.TODO(), name, deleteOptions)
	case gvk.HTTPRoute:
		return c.GatewayAPI().GatewayV1().HTTPRoutes(namespace).Delete(context.TODO(), name, deleteOptions)
	default:
		return fmt.Errorf("unsupported type: %v", typ)
	}
}

var translationMap = map[config.GroupVersionKind]func(r runtime.Object) config.Config{
	gvk.ConfigMap: func(r runtime.Object) config.Config {
		obj := r.(*k8sioapicorev1.ConfigMap)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.ConfigMap,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: obj,
		}
	},
	gvk.CustomResourceDefinition: func(r runtime.Object) config.Config {
		obj := r.(*k8sioapiextensionsapiserverpkgapisapiextensionsv1.CustomResourceDefinition)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.CustomResourceDefinition,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},
	gvk.DaemonSet: func(r runtime.Object) config.Config {
		obj := r.(*k8sioapiappsv1.DaemonSet)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.DaemonSet,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},
	gvk.Deployment: func(r runtime.Object) config.Config {
		obj := r.(*k8sioapiappsv1.Deployment)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.Deployment,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},
	gvk.SubsetRule: func(r runtime.Object) config.Config {
		var obj *apiistioioapinetworkingv1.DestinationRule
		// Handle unstructured objects from Dynamic client
		// First try to convert from unstructured, as Dynamic client returns unstructured objects
		// Note: r may be controllers.Object which embeds runtime.Object, so we need to check the concrete type
		switch v := r.(type) {
		case *unstructured.Unstructured:
			obj = &apiistioioapinetworkingv1.DestinationRule{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(v.Object, obj); err != nil {
				panic(fmt.Sprintf("failed to convert unstructured to DestinationRule: %v", err))
			}
		case *apiistioioapinetworkingv1.DestinationRule:
			// Handle typed objects from Istio client
			obj = v
		default:
			// Fallback: try to convert any runtime.Object to unstructured first, then to DestinationRule
			uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r)
			if err == nil {
				u := &unstructured.Unstructured{Object: uObj}
				obj = &apiistioioapinetworkingv1.DestinationRule{}
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, obj); err != nil {
					panic(fmt.Sprintf("failed to convert object %T to DestinationRule: %v", r, err))
				}
			} else {
				panic(fmt.Sprintf("unexpected object type for SubsetRule: %T, expected *unstructured.Unstructured or *apiistioioapinetworkingv1.DestinationRule, conversion error: %v", r, err))
			}
		}
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.SubsetRule,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec:   &obj.Spec,
			Status: &obj.Status,
		}
	},
	gvk.Namespace: func(r runtime.Object) config.Config {
		obj := r.(*k8sioapicorev1.Namespace)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.Namespace,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},
	gvk.PeerAuthentication: func(r runtime.Object) config.Config {
		var obj *apiistioioapisecurityv1.PeerAuthentication
		switch v := r.(type) {
		case *unstructured.Unstructured:
			obj = &apiistioioapisecurityv1.PeerAuthentication{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(v.Object, obj); err != nil {
				panic(fmt.Sprintf("failed to convert unstructured to PeerAuthentication: %v", err))
			}
		case *apiistioioapisecurityv1.PeerAuthentication:
			obj = v
		default:
			uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r)
			if err == nil {
				u := &unstructured.Unstructured{Object: uObj}
				obj = &apiistioioapisecurityv1.PeerAuthentication{}
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, obj); err != nil {
					panic(fmt.Sprintf("failed to convert object %T to PeerAuthentication: %v", r, err))
				}
			} else {
				panic(fmt.Sprintf("unexpected object type for PeerAuthentication: %T, conversion error: %v", r, err))
			}
		}
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.PeerAuthentication,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec:   &obj.Spec,
			Status: &obj.Status,
		}
	},
	gvk.Secret: func(r runtime.Object) config.Config {
		obj := r.(*k8sioapicorev1.Secret)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.Secret,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: obj,
		}
	},
	gvk.Service: func(r runtime.Object) config.Config {
		obj := r.(*k8sioapicorev1.Service)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.Service,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec:   &obj.Spec,
			Status: &obj.Status,
		}
	},
	gvk.ServiceAccount: func(r runtime.Object) config.Config {
		obj := r.(*k8sioapicorev1.ServiceAccount)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.ServiceAccount,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: obj,
		}
	},
	gvk.StatefulSet: func(r runtime.Object) config.Config {
		obj := r.(*k8sioapiappsv1.StatefulSet)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.StatefulSet,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},
	gvk.MutatingWebhookConfiguration: func(r runtime.Object) config.Config {
		obj := r.(*k8sioapiadmissionregistrationv1.MutatingWebhookConfiguration)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.MutatingWebhookConfiguration,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: obj,
		}
	},
	gvk.ValidatingWebhookConfiguration: func(r runtime.Object) config.Config {
		obj := r.(*k8sioapiadmissionregistrationv1.ValidatingWebhookConfiguration)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.ValidatingWebhookConfiguration,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: obj,
		}
	},
	gvk.ServiceRoute: func(r runtime.Object) config.Config {
		var obj *apiistioioapinetworkingv1.VirtualService
		// Handle unstructured objects from Dynamic client
		// First try to convert from unstructured, as Dynamic client returns unstructured objects
		// Note: r may be controllers.Object which embeds runtime.Object, so we need to check the concrete type
		switch v := r.(type) {
		case *unstructured.Unstructured:
			obj = &apiistioioapinetworkingv1.VirtualService{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(v.Object, obj); err != nil {
				panic(fmt.Sprintf("failed to convert unstructured to VirtualService: %v", err))
			}
		case *apiistioioapinetworkingv1.VirtualService:
			// Handle typed objects from Istio client
			obj = v
		default:
			// Fallback: try to convert any runtime.Object to unstructured first, then to VirtualService
			uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r)
			if err == nil {
				u := &unstructured.Unstructured{Object: uObj}
				obj = &apiistioioapinetworkingv1.VirtualService{}
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, obj); err != nil {
					panic(fmt.Sprintf("failed to convert object %T to VirtualService: %v", r, err))
				}
			} else {
				panic(fmt.Sprintf("unexpected object type for ServiceRoute: %T, expected *unstructured.Unstructured or *apiistioioapinetworkingv1.VirtualService, conversion error: %v", r, err))
			}
		}
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.ServiceRoute,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec:   &obj.Spec,
			Status: &obj.Status,
		}
	},
	gvk.EndpointSlice: func(r runtime.Object) config.Config {
		obj := r.(*k8sioapidiscoveryv1.EndpointSlice)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.EndpointSlice,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: obj,
		}
	},
	gvk.GatewayClass: func(r runtime.Object) config.Config {
		obj := r.(*sigsk8siogatewayapiapisv1.GatewayClass)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.GatewayClass,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec:   &obj.Spec,
			Status: &obj.Status,
		}
	},
	gvk.Gateway: func(r runtime.Object) config.Config {
		obj := r.(*sigsk8siogatewayapiapisv1.Gateway)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.Gateway,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec:   &obj.Spec,
			Status: &obj.Status,
		}
	},
	gvk.HTTPRoute: func(r runtime.Object) config.Config {
		obj := r.(*sigsk8siogatewayapiapisv1.HTTPRoute)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.HTTPRoute,
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec:   &obj.Spec,
			Status: &obj.Status,
		}
	},
}
