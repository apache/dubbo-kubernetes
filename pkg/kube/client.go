package kube

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type client struct {
	dynamic dynamic.Interface
}

type Client interface {
	Dynamic() dynamic.Interface
}

type CLIClient interface {
	DynamicClientFor(gvk schema.GroupVersionKind, obj *unstructured.Unstructured, namespace string) (dynamic.ResourceInterface, error)
}

var (
	_ Client    = &client{}
	_ CLIClient = &client{}
)

func (c *client) Dynamic() dynamic.Interface {
	return c.dynamic
}

func (c *client) DynamicClientFor(gvk schema.GroupVersionKind, obj *unstructured.Unstructured, namespace string) (dynamic.ResourceInterface, error) {
	var dr dynamic.ResourceInterface
	return dr, nil
}
