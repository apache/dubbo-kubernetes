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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	case gvk.DestinationRule:
		spec := cfg.Spec.(*istioioapinetworkingv1alpha3.DestinationRule)
		clonedSpec := protomarshal.Clone(spec)
		obj := &apiistioioapinetworkingv1.DestinationRule{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Spec, clonedSpec)
		return c.Dubbo().NetworkingV1().DestinationRules(cfg.Namespace).Create(context.TODO(), obj, metav1.CreateOptions{})
	case gvk.PeerAuthentication:
		spec := cfg.Spec.(*istioioapisecurityv1beta1.PeerAuthentication)
		clonedSpec := protomarshal.Clone(spec)
		obj := &apiistioioapisecurityv1.PeerAuthentication{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Spec, clonedSpec)
		return c.Dubbo().SecurityV1().PeerAuthentications(cfg.Namespace).Create(context.TODO(), obj, metav1.CreateOptions{})
	case gvk.RequestAuthentication:
		spec := cfg.Spec.(*istioioapisecurityv1beta1.RequestAuthentication)
		clonedSpec := protomarshal.Clone(spec)
		obj := &apiistioioapisecurityv1.RequestAuthentication{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Spec, clonedSpec)
		return c.Dubbo().SecurityV1().RequestAuthentications(cfg.Namespace).Create(context.TODO(), obj, metav1.CreateOptions{})
	case gvk.VirtualService:
		spec := cfg.Spec.(*istioioapinetworkingv1alpha3.VirtualService)
		clonedSpec := protomarshal.Clone(spec)
		obj := &apiistioioapinetworkingv1.VirtualService{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Spec, clonedSpec)
		return c.Dubbo().NetworkingV1().VirtualServices(cfg.Namespace).Create(context.TODO(), obj, metav1.CreateOptions{})
	default:
		return nil, fmt.Errorf("unsupported type: %v", cfg.GroupVersionKind)
	}
}

func update(c kube.Client, cfg config.Config, objMeta metav1.ObjectMeta) (metav1.Object, error) {
	switch cfg.GroupVersionKind {
	case gvk.DestinationRule:
		spec := cfg.Spec.(*istioioapinetworkingv1alpha3.DestinationRule)
		clonedSpec := protomarshal.Clone(spec)
		obj := &apiistioioapinetworkingv1.DestinationRule{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Spec, clonedSpec)
		return c.Dubbo().NetworkingV1().DestinationRules(cfg.Namespace).Update(context.TODO(), obj, metav1.UpdateOptions{})
	case gvk.PeerAuthentication:
		spec := cfg.Spec.(*istioioapisecurityv1beta1.PeerAuthentication)
		clonedSpec := protomarshal.Clone(spec)
		obj := &apiistioioapisecurityv1.PeerAuthentication{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Spec, clonedSpec)
		return c.Dubbo().SecurityV1().PeerAuthentications(cfg.Namespace).Update(context.TODO(), obj, metav1.UpdateOptions{})
	case gvk.RequestAuthentication:
		spec := cfg.Spec.(*istioioapisecurityv1beta1.RequestAuthentication)
		clonedSpec := protomarshal.Clone(spec)
		obj := &apiistioioapisecurityv1.RequestAuthentication{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Spec, clonedSpec)
		return c.Dubbo().SecurityV1().RequestAuthentications(cfg.Namespace).Update(context.TODO(), obj, metav1.UpdateOptions{})
	case gvk.VirtualService:
		spec := cfg.Spec.(*istioioapinetworkingv1alpha3.VirtualService)
		clonedSpec := protomarshal.Clone(spec)
		obj := &apiistioioapinetworkingv1.VirtualService{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Spec, clonedSpec)
		return c.Dubbo().NetworkingV1().VirtualServices(cfg.Namespace).Update(context.TODO(), obj, metav1.UpdateOptions{})
	default:
		return nil, fmt.Errorf("unsupported type: %v", cfg.GroupVersionKind)
	}
}

func updateStatus(c kube.Client, cfg config.Config, objMeta metav1.ObjectMeta) (metav1.Object, error) {
	switch cfg.GroupVersionKind {
	case gvk.DestinationRule:
		status := cfg.Status.(*istioioapimetav1alpha1.IstioStatus)
		clonedStatus := protomarshal.Clone(status)
		obj := &apiistioioapinetworkingv1.DestinationRule{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Status, clonedStatus)
		return c.Dubbo().NetworkingV1().DestinationRules(cfg.Namespace).UpdateStatus(context.TODO(), obj, metav1.UpdateOptions{})
	case gvk.PeerAuthentication:
		status := cfg.Status.(*istioioapimetav1alpha1.IstioStatus)
		clonedStatus := protomarshal.Clone(status)
		obj := &apiistioioapisecurityv1.PeerAuthentication{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Status, clonedStatus)
		return c.Dubbo().SecurityV1().PeerAuthentications(cfg.Namespace).UpdateStatus(context.TODO(), obj, metav1.UpdateOptions{})
	case gvk.RequestAuthentication:
		status := cfg.Status.(*istioioapimetav1alpha1.IstioStatus)
		clonedStatus := protomarshal.Clone(status)
		obj := &apiistioioapisecurityv1.RequestAuthentication{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Status, clonedStatus)
		return c.Dubbo().SecurityV1().RequestAuthentications(cfg.Namespace).UpdateStatus(context.TODO(), obj, metav1.UpdateOptions{})
	case gvk.VirtualService:
		status := cfg.Status.(*istioioapimetav1alpha1.IstioStatus)
		clonedStatus := protomarshal.Clone(status)
		obj := &apiistioioapinetworkingv1.VirtualService{
			ObjectMeta: objMeta,
		}
		assignSpec(&obj.Status, clonedStatus)
		return c.Dubbo().NetworkingV1().VirtualServices(cfg.Namespace).UpdateStatus(context.TODO(), obj, metav1.UpdateOptions{})
	default:
		return nil, fmt.Errorf("unsupported type: %v", cfg.GroupVersionKind)
	}
}

func patch(c kube.Client, orig config.Config, origMeta metav1.ObjectMeta, mod config.Config, modMeta metav1.ObjectMeta, typ types.PatchType) (metav1.Object, error) {
	if orig.GroupVersionKind != mod.GroupVersionKind {
		return nil, fmt.Errorf("gvk mismatch: %v, modified: %v", orig.GroupVersionKind, mod.GroupVersionKind)
	}
	switch orig.GroupVersionKind {
	case gvk.DestinationRule:
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
		return c.Dubbo().NetworkingV1().DestinationRules(orig.Namespace).
			Patch(context.TODO(), orig.Name, typ, patchBytes, metav1.PatchOptions{FieldManager: "planet-discovery"})
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
		return c.Dubbo().SecurityV1().PeerAuthentications(orig.Namespace).
			Patch(context.TODO(), orig.Name, typ, patchBytes, metav1.PatchOptions{FieldManager: "planet-discovery"})
	case gvk.RequestAuthentication:
		origSpec := orig.Spec.(*istioioapisecurityv1beta1.RequestAuthentication)
		modSpec := mod.Spec.(*istioioapisecurityv1beta1.RequestAuthentication)
		clonedOrigSpec := protomarshal.Clone(origSpec)
		clonedModSpec := protomarshal.Clone(modSpec)
		oldRes := &apiistioioapisecurityv1.RequestAuthentication{
			ObjectMeta: origMeta,
		}
		assignSpec(&oldRes.Spec, clonedOrigSpec)
		modRes := &apiistioioapisecurityv1.RequestAuthentication{
			ObjectMeta: modMeta,
		}
		assignSpec(&modRes.Spec, clonedModSpec)
		patchBytes, err := genPatchBytes(oldRes, modRes, typ)
		if err != nil {
			return nil, err
		}
		return c.Dubbo().SecurityV1().RequestAuthentications(orig.Namespace).
			Patch(context.TODO(), orig.Name, typ, patchBytes, metav1.PatchOptions{FieldManager: "planet-discovery"})
	case gvk.VirtualService:
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
		return c.Dubbo().NetworkingV1().VirtualServices(orig.Namespace).
			Patch(context.TODO(), orig.Name, typ, patchBytes, metav1.PatchOptions{FieldManager: "planet-discovery"})
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
	case gvk.DestinationRule:
		return c.Dubbo().NetworkingV1().DestinationRules(namespace).Delete(context.TODO(), name, deleteOptions)
	case gvk.PeerAuthentication:
		return c.Dubbo().SecurityV1().PeerAuthentications(namespace).Delete(context.TODO(), name, deleteOptions)
	case gvk.RequestAuthentication:
		return c.Dubbo().SecurityV1().RequestAuthentications(namespace).Delete(context.TODO(), name, deleteOptions)
	case gvk.VirtualService:
		return c.Dubbo().NetworkingV1().VirtualServices(namespace).Delete(context.TODO(), name, deleteOptions)
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
	gvk.DestinationRule: func(r runtime.Object) config.Config {
		obj := r.(*apiistioioapinetworkingv1.DestinationRule)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.DestinationRule,
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
		obj := r.(*apiistioioapisecurityv1.PeerAuthentication)
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
	gvk.RequestAuthentication: func(r runtime.Object) config.Config {
		obj := r.(*apiistioioapisecurityv1.RequestAuthentication)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.RequestAuthentication,
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
	gvk.VirtualService: func(r runtime.Object) config.Config {
		obj := r.(*apiistioioapinetworkingv1.VirtualService)
		return config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.VirtualService,
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
}
