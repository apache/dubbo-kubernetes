package kclient

import (
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	klabels "k8s.io/apimachinery/pkg/labels"
	apitypes "k8s.io/apimachinery/pkg/types"
)

type Reader[T controllers.Object] interface {
	// Get looks up an object by name and namespace. If it does not exist, nil is returned
	Get(name, namespace string) T
	// List looks up an object by namespace and labels.
	// Use metav1.NamespaceAll and klabels.Everything() to select everything.
	List(namespace string, selector klabels.Selector) []T
}

type Writer[T controllers.Object] interface {
	// Create creates a resource, returning the newly applied resource.
	Create(object T) (T, error)
	// Update updates a resource, returning the newly applied resource.
	Update(object T) (T, error)
	// UpdateStatus updates a resource's status, returning the newly applied resource.
	UpdateStatus(object T) (T, error)
	// Patch patches the resource, returning the newly applied resource.
	Patch(name, namespace string, pt apitypes.PatchType, data []byte) (T, error)
	// PatchStatus patches the resource's status, returning the newly applied resource.
	PatchStatus(name, namespace string, pt apitypes.PatchType, data []byte) (T, error)
	// ApplyStatus does a server-side Apply of the the resource's status, returning the newly applied resource.
	// fieldManager is a required field; see https://kubernetes.io/docs/reference/using-api/server-side-apply/#managers.
	ApplyStatus(name, namespace string, pt apitypes.PatchType, data []byte, fieldManager string) (T, error)
	// Delete removes a resource.
	Delete(name, namespace string) error
}

type Informer[T controllers.Object] interface {
	Start(stop <-chan struct{})
}

type Client[T controllers.Object] interface {
	Reader[T]
	Writer[T]
	Informer[T]
}
