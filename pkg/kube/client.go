package kube

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type client struct {
	dynamic    dynamic.Interface
	config     *rest.Config
	revision   string
	factory    *clientFactory
	restClient *rest.RESTClient
}

type Client interface {
	Dynamic() dynamic.Interface
}

type CLIClient interface {
	Client
	DynamicClientFor(gvk schema.GroupVersionKind, obj *unstructured.Unstructured, namespace string) (dynamic.ResourceInterface, error)
}

type ClientOption func(cliClient CLIClient) CLIClient

func NewCLIClient(clientCfg clientcmd.ClientConfig, opts ...ClientOption) (CLIClient, error) {
	return newInternalClient(newClientFactory(clientCfg, true), opts...)
}

func newInternalClient(factory *clientFactory, opts ...ClientOption) (CLIClient, error) {
	var c client
	var err error
	c.factory = factory
	c.config, err = factory.ToRestConfig()
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(&c)
	}

	return nil, err
}

var (
	_ Client    = &client{}
	_ CLIClient = &client{}
)

func (c *client) Dynamic() dynamic.Interface {
	return c.dynamic
}

func (c *client) DynamicClientFor(gvk schema.GroupVersionKind, obj *unstructured.Unstructured, namespace string) (dynamic.ResourceInterface, error) {
	// TODO
	var dr dynamic.ResourceInterface
	return dr, nil
}

func (c *client) bestEffortToGVR(gvk schema.GroupVersionKind, obj *unstructured.Unstructured, namespace string) (schema.GroupVersionResource, bool) {
	// TODO
	return schema.GroupVersionResource{}, false
}

func WithRevision(revision string) ClientOption {
	return func(cliClient CLIClient) CLIClient {
		client := cliClient.(*client)
		client.revision = revision
		return client
	}
}

var (
	DubboScheme = dubboScheme()
	DubboCodec  = serializer.NewCodecFactory(DubboScheme)
)

func dubboScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(kubescheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	return scheme
}
