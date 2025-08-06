package kubeclient

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"
	"github.com/apache/dubbo-kubernetes/pkg/kube/informerfactory"
	ktypes "github.com/apache/dubbo-kubernetes/pkg/kube/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	k8sioapiadmissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	k8sioapiappsv1 "k8s.io/api/apps/v1"
	k8sioapicertificatesv1 "k8s.io/api/certificates/v1"
	k8sioapicertificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	k8sioapicoordinationv1 "k8s.io/api/coordination/v1"
	k8sioapicorev1 "k8s.io/api/core/v1"
	k8sioapidiscoveryv1 "k8s.io/api/discovery/v1"
	k8sioapinetworkingv1 "k8s.io/api/networking/v1"
	k8sioapipolicyv1 "k8s.io/api/policy/v1"
	k8sioapiextensionsapiserverpkgapisapiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func GetWriteClient[T runtime.Object](c ClientGetter, namespace string) ktypes.WriteAPI[T] {
	switch any(ptr.Empty[T]()).(type) {
	case *k8sioapicertificatesv1.CertificateSigningRequest:
		return c.Kube().CertificatesV1().CertificateSigningRequests().(ktypes.WriteAPI[T])
	case *k8sioapicertificatesv1beta1.ClusterTrustBundle:
		return c.Kube().CertificatesV1beta1().ClusterTrustBundles().(ktypes.WriteAPI[T])
	case *k8sioapicorev1.ConfigMap:
		return c.Kube().CoreV1().ConfigMaps(namespace).(ktypes.WriteAPI[T])
	case *k8sioapiextensionsapiserverpkgapisapiextensionsv1.CustomResourceDefinition:
		return c.Ext().ApiextensionsV1().CustomResourceDefinitions().(ktypes.WriteAPI[T])
	case *k8sioapiappsv1.DaemonSet:
		return c.Kube().AppsV1().DaemonSets(namespace).(ktypes.WriteAPI[T])
	case *k8sioapiappsv1.Deployment:
		return c.Kube().AppsV1().Deployments(namespace).(ktypes.WriteAPI[T])
	case *k8sioapidiscoveryv1.EndpointSlice:
		return c.Kube().DiscoveryV1().EndpointSlices(namespace).(ktypes.WriteAPI[T])
	case *k8sioapicorev1.Endpoints:
		return c.Kube().CoreV1().Endpoints(namespace).(ktypes.WriteAPI[T])
	case *k8sioapinetworkingv1.Ingress:
		return c.Kube().NetworkingV1().Ingresses(namespace).(ktypes.WriteAPI[T])
	case *k8sioapinetworkingv1.IngressClass:
		return c.Kube().NetworkingV1().IngressClasses().(ktypes.WriteAPI[T])
	case *k8sioapicoordinationv1.Lease:
		return c.Kube().CoordinationV1().Leases(namespace).(ktypes.WriteAPI[T])
	case *k8sioapiadmissionregistrationv1.MutatingWebhookConfiguration:
		return c.Kube().AdmissionregistrationV1().MutatingWebhookConfigurations().(ktypes.WriteAPI[T])
	case *k8sioapicorev1.Namespace:
		return c.Kube().CoreV1().Namespaces().(ktypes.WriteAPI[T])
	case *k8sioapicorev1.Node:
		return c.Kube().CoreV1().Nodes().(ktypes.WriteAPI[T])
	case *k8sioapicorev1.Pod:
		return c.Kube().CoreV1().Pods(namespace).(ktypes.WriteAPI[T])
	case *k8sioapipolicyv1.PodDisruptionBudget:
		return c.Kube().PolicyV1().PodDisruptionBudgets(namespace).(ktypes.WriteAPI[T])
	case *k8sioapicorev1.Secret:
		return c.Kube().CoreV1().Secrets(namespace).(ktypes.WriteAPI[T])
	case *k8sioapicorev1.Service:
		return c.Kube().CoreV1().Services(namespace).(ktypes.WriteAPI[T])
	case *k8sioapicorev1.ServiceAccount:
		return c.Kube().CoreV1().ServiceAccounts(namespace).(ktypes.WriteAPI[T])
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
	case gvr.MutatingWebhookConfiguration:
		return &k8sioapiadmissionregistrationv1.MutatingWebhookConfiguration{}
	case gvr.Namespace:
		return &k8sioapicorev1.Namespace{}
	case gvr.Secret:
		return &k8sioapicorev1.Secret{}
	case gvr.Service:
		return &k8sioapicorev1.Service{}
	case gvr.ServiceAccount:
		return &k8sioapicorev1.ServiceAccount{}
	case gvr.StatefulSet:
		return &k8sioapiappsv1.StatefulSet{}
	case gvr.ValidatingWebhookConfiguration:
		return &k8sioapiadmissionregistrationv1.ValidatingWebhookConfiguration{}
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
	case gvr.MutatingWebhookConfiguration:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().AdmissionregistrationV1().MutatingWebhookConfigurations().List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().AdmissionregistrationV1().MutatingWebhookConfigurations().Watch(context.Background(), options)
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
	case gvr.StatefulSet:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().AppsV1().StatefulSets(opts.Namespace).List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().AppsV1().StatefulSets(opts.Namespace).Watch(context.Background(), options)
		}
	case gvr.ValidatingWebhookConfiguration:
		l = func(options metav1.ListOptions) (runtime.Object, error) {
			return c.Kube().AdmissionregistrationV1().ValidatingWebhookConfigurations().List(context.Background(), options)
		}
		w = func(options metav1.ListOptions) (watch.Interface, error) {
			return c.Kube().AdmissionregistrationV1().ValidatingWebhookConfigurations().Watch(context.Background(), options)
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
