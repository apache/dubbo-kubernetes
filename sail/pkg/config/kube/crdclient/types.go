package crdclient

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
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

func create(c kube.Client, cfg config.Config, objMeta metav1.ObjectMeta) (metav1.Object, error) {
	switch cfg.GroupVersionKind {
	case gvk.DestinationRule:
		return c.Dubbo().NetworkingV1().DestinationRules(cfg.Namespace).Create(context.TODO(), &apiistioioapinetworkingv1.DestinationRule{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*istioioapinetworkingv1alpha3.DestinationRule)),
		}, metav1.CreateOptions{})
	case gvk.PeerAuthentication:
		return c.Dubbo().SecurityV1().PeerAuthentications(cfg.Namespace).Create(context.TODO(), &apiistioioapisecurityv1.PeerAuthentication{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*istioioapisecurityv1beta1.PeerAuthentication)),
		}, metav1.CreateOptions{})
	case gvk.RequestAuthentication:
		return c.Dubbo().SecurityV1().RequestAuthentications(cfg.Namespace).Create(context.TODO(), &apiistioioapisecurityv1.RequestAuthentication{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*istioioapisecurityv1beta1.RequestAuthentication)),
		}, metav1.CreateOptions{})
	case gvk.VirtualService:
		return c.Dubbo().NetworkingV1().VirtualServices(cfg.Namespace).Create(context.TODO(), &apiistioioapinetworkingv1.VirtualService{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*istioioapinetworkingv1alpha3.VirtualService)),
		}, metav1.CreateOptions{})
	default:
		return nil, fmt.Errorf("unsupported type: %v", cfg.GroupVersionKind)
	}
}

func update(c kube.Client, cfg config.Config, objMeta metav1.ObjectMeta) (metav1.Object, error) {
	switch cfg.GroupVersionKind {
	case gvk.DestinationRule:
		return c.Dubbo().NetworkingV1().DestinationRules(cfg.Namespace).Update(context.TODO(), &apiistioioapinetworkingv1.DestinationRule{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*istioioapinetworkingv1alpha3.DestinationRule)),
		}, metav1.UpdateOptions{})
	case gvk.PeerAuthentication:
		return c.Dubbo().SecurityV1().PeerAuthentications(cfg.Namespace).Update(context.TODO(), &apiistioioapisecurityv1.PeerAuthentication{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*istioioapisecurityv1beta1.PeerAuthentication)),
		}, metav1.UpdateOptions{})
	case gvk.RequestAuthentication:
		return c.Dubbo().SecurityV1().RequestAuthentications(cfg.Namespace).Update(context.TODO(), &apiistioioapisecurityv1.RequestAuthentication{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*istioioapisecurityv1beta1.RequestAuthentication)),
		}, metav1.UpdateOptions{})
	case gvk.VirtualService:
		return c.Dubbo().NetworkingV1().VirtualServices(cfg.Namespace).Update(context.TODO(), &apiistioioapinetworkingv1.VirtualService{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*istioioapinetworkingv1alpha3.VirtualService)),
		}, metav1.UpdateOptions{})
	default:
		return nil, fmt.Errorf("unsupported type: %v", cfg.GroupVersionKind)
	}
}

func updateStatus(c kube.Client, cfg config.Config, objMeta metav1.ObjectMeta) (metav1.Object, error) {
	switch cfg.GroupVersionKind {
	case gvk.DestinationRule:
		return c.Dubbo().NetworkingV1().DestinationRules(cfg.Namespace).UpdateStatus(context.TODO(), &apiistioioapinetworkingv1.DestinationRule{
			ObjectMeta: objMeta,
			Status:     *(cfg.Status.(*istioioapimetav1alpha1.IstioStatus)),
		}, metav1.UpdateOptions{})
	case gvk.PeerAuthentication:
		return c.Dubbo().SecurityV1().PeerAuthentications(cfg.Namespace).UpdateStatus(context.TODO(), &apiistioioapisecurityv1.PeerAuthentication{
			ObjectMeta: objMeta,
			Status:     *(cfg.Status.(*istioioapimetav1alpha1.IstioStatus)),
		}, metav1.UpdateOptions{})
	case gvk.RequestAuthentication:
		return c.Dubbo().SecurityV1().RequestAuthentications(cfg.Namespace).UpdateStatus(context.TODO(), &apiistioioapisecurityv1.RequestAuthentication{
			ObjectMeta: objMeta,
			Status:     *(cfg.Status.(*istioioapimetav1alpha1.IstioStatus)),
		}, metav1.UpdateOptions{})
	case gvk.VirtualService:
		return c.Dubbo().NetworkingV1().VirtualServices(cfg.Namespace).UpdateStatus(context.TODO(), &apiistioioapinetworkingv1.VirtualService{
			ObjectMeta: objMeta,
			Status:     *(cfg.Status.(*istioioapimetav1alpha1.IstioStatus)),
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
	case gvk.DestinationRule:
		oldRes := &apiistioioapinetworkingv1.DestinationRule{
			ObjectMeta: origMeta,
			Spec:       *(orig.Spec.(*istioioapinetworkingv1alpha3.DestinationRule)),
		}
		modRes := &apiistioioapinetworkingv1.DestinationRule{
			ObjectMeta: modMeta,
			Spec:       *(mod.Spec.(*istioioapinetworkingv1alpha3.DestinationRule)),
		}
		patchBytes, err := genPatchBytes(oldRes, modRes, typ)
		if err != nil {
			return nil, err
		}
		return c.Dubbo().NetworkingV1().DestinationRules(orig.Namespace).
			Patch(context.TODO(), orig.Name, typ, patchBytes, metav1.PatchOptions{FieldManager: "sail-discovery"})
	case gvk.PeerAuthentication:
		oldRes := &apiistioioapisecurityv1.PeerAuthentication{
			ObjectMeta: origMeta,
			Spec:       *(orig.Spec.(*istioioapisecurityv1beta1.PeerAuthentication)),
		}
		modRes := &apiistioioapisecurityv1.PeerAuthentication{
			ObjectMeta: modMeta,
			Spec:       *(mod.Spec.(*istioioapisecurityv1beta1.PeerAuthentication)),
		}
		patchBytes, err := genPatchBytes(oldRes, modRes, typ)
		if err != nil {
			return nil, err
		}
		return c.Dubbo().SecurityV1().PeerAuthentications(orig.Namespace).
			Patch(context.TODO(), orig.Name, typ, patchBytes, metav1.PatchOptions{FieldManager: "sail-discovery"})
	case gvk.RequestAuthentication:
		oldRes := &apiistioioapisecurityv1.RequestAuthentication{
			ObjectMeta: origMeta,
			Spec:       *(orig.Spec.(*istioioapisecurityv1beta1.RequestAuthentication)),
		}
		modRes := &apiistioioapisecurityv1.RequestAuthentication{
			ObjectMeta: modMeta,
			Spec:       *(mod.Spec.(*istioioapisecurityv1beta1.RequestAuthentication)),
		}
		patchBytes, err := genPatchBytes(oldRes, modRes, typ)
		if err != nil {
			return nil, err
		}
		return c.Dubbo().SecurityV1().RequestAuthentications(orig.Namespace).
			Patch(context.TODO(), orig.Name, typ, patchBytes, metav1.PatchOptions{FieldManager: "sail-discovery"})
	case gvk.VirtualService:
		oldRes := &apiistioioapinetworkingv1.VirtualService{
			ObjectMeta: origMeta,
			Spec:       *(orig.Spec.(*istioioapinetworkingv1alpha3.VirtualService)),
		}
		modRes := &apiistioioapinetworkingv1.VirtualService{
			ObjectMeta: modMeta,
			Spec:       *(mod.Spec.(*istioioapinetworkingv1alpha3.VirtualService)),
		}
		patchBytes, err := genPatchBytes(oldRes, modRes, typ)
		if err != nil {
			return nil, err
		}
		return c.Dubbo().NetworkingV1().VirtualServices(orig.Namespace).
			Patch(context.TODO(), orig.Name, typ, patchBytes, metav1.PatchOptions{FieldManager: "sail-discovery"})
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
