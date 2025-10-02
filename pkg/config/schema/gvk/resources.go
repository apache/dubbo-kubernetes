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

package gvk

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	Namespace                      = config.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"}
	CustomResourceDefinition       = config.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}
	MutatingWebhookConfiguration   = config.GroupVersionKind{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "MutatingWebhookConfiguration"}
	ValidatingWebhookConfiguration = config.GroupVersionKind{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "ValidatingWebhookConfiguration"}
	Deployment                     = config.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	StatefulSet                    = config.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}
	DaemonSet                      = config.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"}
	Job                            = config.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"}
	ConfigMap                      = config.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}
	Secret                         = config.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}
	Service                        = config.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"}
	ServiceAccount                 = config.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"}
	MeshConfig                     = config.GroupVersionKind{Group: "", Version: "v1alpha1", Kind: "MeshConfig"}
	RequestAuthentication          = config.GroupVersionKind{Group: "security.dubbo.io", Version: "v1", Kind: "RequestAuthentication"}
	PeerAuthentication             = config.GroupVersionKind{Group: "security.dubbo.io", Version: "v1", Kind: "PeerAuthentication"}
	AuthorizationPolicy            = config.GroupVersionKind{Group: "security.dubbo.io", Version: "v1", Kind: "AuthorizationPolicy"}
	DestinationRule                = config.GroupVersionKind{Group: "networking.dubbo.io", Version: "v1", Kind: "DestinationRule"}
	VirtualService                 = config.GroupVersionKind{Group: "networking.dubbo.io", Version: "v1", Kind: "VirtualService"}
)

func ToGVR(g config.GroupVersionKind) (schema.GroupVersionResource, bool) {
	switch g {
	case CustomResourceDefinition:
		return gvr.CustomResourceDefinition, true
	case MutatingWebhookConfiguration:
		return gvr.MutatingWebhookConfiguration, true
	case ValidatingWebhookConfiguration:
		return gvr.ValidatingWebhookConfiguration, true
	case Deployment:
		return gvr.Deployment, true
	case StatefulSet:
		return gvr.StatefulSet, true
	case DaemonSet:
		return gvr.DaemonSet, true
	case ConfigMap:
		return gvr.ConfigMap, true
	case Secret:
		return gvr.Secret, true
	case Service:
		return gvr.Service, true
	case ServiceAccount:
		return gvr.ServiceAccount, true
	case Namespace:
		return gvr.Namespace, true
	case Job:
		return gvr.Job, true
	case MeshConfig:
		return gvr.MeshConfig, true
	case RequestAuthentication:
		return gvr.RequestAuthentication, true
	case PeerAuthentication:
		return gvr.PeerAuthentication, true
	case AuthorizationPolicy:
		return gvr.AuthorizationPolicy, true
	case DestinationRule:
		return gvr.DestinationRule, true
	case VirtualService:
		return gvr.VirtualService, true
	}
	return schema.GroupVersionResource{}, false
}
func MustToGVR(g config.GroupVersionKind) schema.GroupVersionResource {
	r, ok := ToGVR(g)
	if !ok {
		panic("unknown kind: " + g.String())
	}
	return r
}

func FromGVR(g schema.GroupVersionResource) (config.GroupVersionKind, bool) {
	switch g {
	case gvr.CustomResourceDefinition:
		return CustomResourceDefinition, true
	case gvr.Namespace:
		return Namespace, true
	case gvr.Deployment:
		return Deployment, true
	case gvr.StatefulSet:
		return StatefulSet, true
	case gvr.DaemonSet:
		return DaemonSet, true
	case gvr.Job:
		return Job, true
	case gvr.PeerAuthentication:
		return PeerAuthentication, true
	case gvr.RequestAuthentication:
		return RequestAuthentication, true
	case gvr.AuthorizationPolicy:
		return AuthorizationPolicy, true
	case gvr.VirtualService:
		return VirtualService, true
	case gvr.DestinationRule:
		return DestinationRule, true
	}
	return config.GroupVersionKind{}, false
}

func MustFromGVR(g schema.GroupVersionResource) config.GroupVersionKind {
	r, ok := FromGVR(g)
	if !ok {
		panic("unknown kind: " + g.String())
	}
	return r
}
