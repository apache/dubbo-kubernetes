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

package kubeclient

import (
	"context"
	"fmt"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"
	"github.com/apache/dubbo-kubernetes/pkg/kube/informerfactory"
	ktypes "github.com/apache/dubbo-kubernetes/pkg/kube/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/util/ptr"
	apiistioioapinetworkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	apiistioioapisecurityv1 "istio.io/client-go/pkg/apis/security/v1"
	k8sioapiadmissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	k8sioapiappsv1 "k8s.io/api/apps/v1"
	k8sioapicertificatesv1 "k8s.io/api/certificates/v1"
	k8sioapicorev1 "k8s.io/api/core/v1"
	k8sioapidiscoveryv1 "k8s.io/api/discovery/v1"
	k8sioapiextensionsapiserverpkgapisapiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	sigsk8siogatewayapiapisv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func GetWriteClient[T runtime.Object](c ClientGetter, namespace string) ktypes.WriteAPI[T] {
	switch any(ptr.Empty[T]()).(type) {
	case *k8sioapiextensionsapiserverpkgapisapiextensionsv1.CustomResourceDefinition:
		return c.Ext().ApiextensionsV1().CustomResourceDefinitions().(ktypes.WriteAPI[T])
	case *k8sioapicertificatesv1.CertificateSigningRequest:
		return c.Kube().CertificatesV1().CertificateSigningRequests().(ktypes.WriteAPI[T])
	case *k8sioapicorev1.ConfigMap:
		return c.Kube().CoreV1().ConfigMaps(namespace).(ktypes.WriteAPI[T])
	case *k8sioapiappsv1.DaemonSet:
		return c.Kube().AppsV1().DaemonSets(namespace).(ktypes.WriteAPI[T])
	case *k8sioapiappsv1.Deployment:
		return c.Kube().AppsV1().Deployments(namespace).(ktypes.WriteAPI[T])
	case *k8sioapiadmissionregistrationv1.MutatingWebhookConfiguration:
		return c.Kube().AdmissionregistrationV1().MutatingWebhookConfigurations().(ktypes.WriteAPI[T])
	case *k8sioapiadmissionregistrationv1.ValidatingWebhookConfiguration:
		return c.Kube().AdmissionregistrationV1().ValidatingWebhookConfigurations().(ktypes.WriteAPI[T])
	case *k8sioapicorev1.Namespace:
		return c.Kube().CoreV1().Namespaces().(ktypes.WriteAPI[T])
	case *k8sioapicorev1.Node:
		return c.Kube().CoreV1().Nodes().(ktypes.WriteAPI[T])
	case *k8sioapicorev1.Pod:
		return c.Kube().CoreV1().Pods(namespace).(ktypes.WriteAPI[T])
	case *k8sioapicorev1.Secret:
		return c.Kube().CoreV1().Secrets(namespace).(ktypes.WriteAPI[T])
	case *k8sioapicorev1.Service:
		return c.Kube().CoreV1().Services(namespace).(ktypes.WriteAPI[T])
	case *k8sioapicorev1.ServiceAccount:
		return c.Kube().CoreV1().ServiceAccounts(namespace).(ktypes.WriteAPI[T])
	case *apiistioioapisecurityv1.PeerAuthentication:
		return c.Dubbo().SecurityV1().PeerAuthentications(namespace).(ktypes.WriteAPI[T])
	case *apiistioioapinetworkingv1.VirtualService:
		return c.Dubbo().NetworkingV1().VirtualServices(namespace).(ktypes.WriteAPI[T])
	case *apiistioioapinetworkingv1.DestinationRule:
		return c.Dubbo().NetworkingV1().DestinationRules(namespace).(ktypes.WriteAPI[T])
	case *sigsk8siogatewayapiapisv1.GatewayClass:
		return c.GatewayAPI().GatewayV1().GatewayClasses().(ktypes.WriteAPI[T])
	case *sigsk8siogatewayapiapisv1.Gateway:
		return c.GatewayAPI().GatewayV1().Gateways(namespace).(ktypes.WriteAPI[T])
	case *sigsk8siogatewayapiapisv1.HTTPRoute:
		return c.GatewayAPI().GatewayV1().HTTPRoutes(namespace).(ktypes.WriteAPI[T])
	default:
		panic(fmt.Sprintf("Unknown type %T", ptr.Empty[T]()))
	}
}

func gvrToObject(g schema.GroupVersionResource) runtime.Object {
	switch g {
	case gvr.ConfigMap:
		return &k8sioapicorev1.ConfigMap{}
	case gvr.CustomResourceDefinition:
		return &k8sioapiextensionsapiserverpkgapisapiextensionsv1.CustomResourceDefinition{}
	case gvr.DaemonSet:
		return &k8sioapiappsv1.DaemonSet{}
	case gvr.Deployment:
		return &k8sioapiappsv1.Deployment{}
	case gvr.Namespace:
		return &k8sioapicorev1.Namespace{}
	case gvr.Secret:
		return &k8sioapicorev1.Secret{}
	case gvr.EndpointSlice:
		return &k8sioapidiscoveryv1.EndpointSlice{}
	case gvr.Endpoints:
		return &k8sioapicorev1.Endpoints{}
	case gvr.Service:
		return &k8sioapicorev1.Service{}
	case gvr.ServiceAccount:
		return &k8sioapicorev1.ServiceAccount{}
	case gvr.StatefulSet:
		return &k8sioapiappsv1.StatefulSet{}
	case gvr.Pod:
		return &k8sioapicorev1.Pod{}
	case gvr.MutatingWebhookConfiguration:
		return &k8sioapiadmissionregistrationv1.MutatingWebhookConfiguration{}
	case gvr.ValidatingWebhookConfiguration:
		return &k8sioapiadmissionregistrationv1.ValidatingWebhookConfiguration{}
	case gvr.PeerAuthentication:
		return &apiistioioapisecurityv1.PeerAuthentication{}
	case gvr.ServiceRoute:
		return &apiistioioapinetworkingv1.VirtualService{}
	case gvr.SubsetRule:
		return &apiistioioapinetworkingv1.DestinationRule{}
	case gvr.GatewayClass:
		return &sigsk8siogatewayapiapisv1.GatewayClass{}
	case gvr.KubernetesGateway:
		return &sigsk8siogatewayapiapisv1.Gateway{}
	case gvr.HTTPRoute:
		return &sigsk8siogatewayapiapisv1.HTTPRoute{}
	default:
		panic(fmt.Sprintf("Unknown type %v", g))
	}
}

func getInformerFiltered(c ClientGetter, opts ktypes.InformerOptions, g schema.GroupVersionResource) informerfactory.StartableInformer {
	var l func(options metav1.ListOptions) (runtime.Object, error)
	var w func(options metav1.ListOptions) (watch.Interface, error)

	switch g {
	case gvr.ConfigMap:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().CoreV1().ConfigMaps(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().CoreV1().ConfigMaps(opts.Namespace).Watch(context.Background(), options)
		}
	case gvr.CustomResourceDefinition:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Ext().ApiextensionsV1().CustomResourceDefinitions().List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Ext().ApiextensionsV1().CustomResourceDefinitions().Watch(context.Background(), options)
		}
	case gvr.DaemonSet:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().AppsV1().DaemonSets(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().AppsV1().DaemonSets(opts.Namespace).Watch(context.Background(), options)
		}
	case gvr.Deployment:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().AppsV1().Deployments(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().AppsV1().Deployments(opts.Namespace).Watch(context.Background(), options)
		}
	case gvr.StatefulSet:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().AppsV1().StatefulSets(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().AppsV1().StatefulSets(opts.Namespace).Watch(context.Background(), options)
		}
	case gvr.Namespace:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().CoreV1().Namespaces().List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().CoreV1().Namespaces().Watch(context.Background(), options)
		}
	case gvr.Secret:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().CoreV1().Secrets(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().CoreV1().Secrets(opts.Namespace).Watch(context.Background(), options)
		}
	case gvr.EndpointSlice:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().DiscoveryV1().EndpointSlices(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().DiscoveryV1().EndpointSlices(opts.Namespace).Watch(context.Background(), options)
		}
	case gvr.Endpoints:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().CoreV1().Endpoints(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().CoreV1().Endpoints(opts.Namespace).Watch(context.Background(), options)
		}
	case gvr.Service:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().CoreV1().Services(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().CoreV1().Services(opts.Namespace).Watch(context.Background(), options)
		}
	case gvr.ServiceAccount:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().CoreV1().ServiceAccounts(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().CoreV1().ServiceAccounts(opts.Namespace).Watch(context.Background(), options)
		}
	case gvr.MutatingWebhookConfiguration:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().AdmissionregistrationV1().MutatingWebhookConfigurations().List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().AdmissionregistrationV1().MutatingWebhookConfigurations().Watch(context.Background(), options)
		}
	case gvr.ValidatingWebhookConfiguration:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().AdmissionregistrationV1().ValidatingWebhookConfigurations().List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().AdmissionregistrationV1().ValidatingWebhookConfigurations().Watch(context.Background(), options)
		}
	case gvr.ServiceRoute:
		// ServiceRoute uses networking.dubbo.apache.org API group, not networking.istio.io
		// Use Dynamic client to access it
		gvr := schema.GroupVersionResource{
			Group:    "networking.dubbo.apache.org",
			Version:  "v1",
			Resource: "serviceroutes",
		}
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Dynamic().Resource(gvr).Namespace(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Dynamic().Resource(gvr).Namespace(opts.Namespace).Watch(context.Background(), options)
		}
	case gvr.SubsetRule:
		// SubsetRule uses networking.dubbo.apache.org API group, not networking.istio.io
		// Use Dynamic client to access it
		gvr := schema.GroupVersionResource{
			Group:    "networking.dubbo.apache.org",
			Version:  "v1",
			Resource: "subsetrules",
		}
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			// Log the namespace being watched for diagnosis
			if opts.Namespace == "" {
				log.Infof("SubsetRule informer: List called for all namespaces")
			} else {
				log.Infof("SubsetRule informer: List called for namespace %s", opts.Namespace)
			}
			return c.Dynamic().Resource(gvr).Namespace(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			// Log the namespace being watched for diagnosis
			if opts.Namespace == "" {
				log.Infof("SubsetRule informer: Watch called for all namespaces")
			} else {
				log.Infof("SubsetRule informer: Watch called for namespace %s", opts.Namespace)
			}
			watchInterface, err := c.Dynamic().Resource(gvr).Namespace(opts.Namespace).Watch(context.Background(), options)
			if err != nil {
				log.Errorf("SubsetRule informer: Watch failed: %v", err)
			} else {
				log.Infof("SubsetRule informer: Watch connection established successfully")
			}
			return watchInterface, err
		}
	case gvr.PeerAuthentication:
		peerAuthGVR := schema.GroupVersionResource{
			Group:    "security.dubbo.apache.org",
			Version:  "v1",
			Resource: "peerauthentications",
		}
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			if opts.Namespace == "" {
				log.Infof("PeerAuthentication informer: List called for all namespaces")
			} else {
				log.Infof("PeerAuthentication informer: List called for namespace %s", opts.Namespace)
			}
			return c.Dynamic().Resource(peerAuthGVR).Namespace(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			if opts.Namespace == "" {
				log.Infof("PeerAuthentication informer: Watch called for all namespaces")
			} else {
				log.Infof("PeerAuthentication informer: Watch called for namespace %s", opts.Namespace)
			}
			watchInterface, err := c.Dynamic().Resource(peerAuthGVR).Namespace(opts.Namespace).Watch(context.Background(), options)
			if err != nil {
				log.Errorf("PeerAuthentication informer: Watch failed: %v", err)
			} else {
				log.Infof("PeerAuthentication informer: Watch connection established successfully")
			}
			return watchInterface, err
		}
	case gvr.Pod:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().CoreV1().Pods(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().CoreV1().Pods(opts.Namespace).Watch(context.Background(), options)
		}
	case gvr.GatewayClass:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.GatewayAPI().GatewayV1().GatewayClasses().List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.GatewayAPI().GatewayV1().GatewayClasses().Watch(context.Background(), options)
		}
	case gvr.KubernetesGateway:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.GatewayAPI().GatewayV1().Gateways(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.GatewayAPI().GatewayV1().Gateways(opts.Namespace).Watch(context.Background(), options)
		}
	case gvr.HTTPRoute:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.GatewayAPI().GatewayV1().HTTPRoutes(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.GatewayAPI().GatewayV1().HTTPRoutes(opts.Namespace).Watch(context.Background(), options)
		}
	default:
		panic(fmt.Sprintf("Unknown type %v", g))
	}
	return c.Informers().InformerFor(g, opts, func() cache.SharedIndexInformer {
		inf := cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					options.FieldSelector = opts.FieldSelector
					options.LabelSelector = opts.LabelSelector
					return l(options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					options.FieldSelector = opts.FieldSelector
					options.LabelSelector = opts.LabelSelector
					return w(options)
				},
			},
			gvrToObject(g),
			0,
			cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
		)
		setupInformer(opts, inf)
		return inf
	})
}
