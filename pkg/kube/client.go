package kube

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/kube/collections"
	"github.com/apache/dubbo-kubernetes/pkg/kube/informerfactory"
	"github.com/apache/dubbo-kubernetes/pkg/laziness"
	kubeExtClient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubeVersion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"time"
)

type client struct {
	extSet          kubeExtClient.Interface
	config          *rest.Config
	revision        string
	factory         *clientFactory
	version         laziness.Laziness[*kubeVersion.Info]
	informerFactory informerfactory.InformerFactory
	restClient      *rest.RESTClient
	discoveryClient discovery.CachedDiscoveryInterface
	dynamic         dynamic.Interface
	kube            kubernetes.Interface
	metadata        metadata.Interface
	mapper          meta.ResettableRESTMapper
	http            *http.Client
}

type Client interface {
	Ext() kubeExtClient.Interface
	Kube() kubernetes.Interface
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
	c.restClient, err = factory.RestClient()
	if err != nil {
		return nil, err
	}
	c.discoveryClient, err = factory.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	c.mapper, err = factory.mapper.Get()
	if err != nil {
		return nil, err
	}
	c.kube, err = kubernetes.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}
	c.metadata, err = metadata.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}
	c.dynamic, err = dynamic.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}
	c.informerFactory = informerfactory.NewSharedInformerFactory()
	c.http = &http.Client{
		Timeout: time.Second * 15,
	}
	var clientWithTimeout kubernetes.Interface
	clientWithTimeout = c.kube
	restConfig := c.RESTConfig()
	if restConfig != nil {
		restConfig.Timeout = time.Second * 5
		kubeClient, err := kubernetes.NewForConfig(restConfig)
		if err == nil {
			clientWithTimeout = kubeClient
		}
	}
	c.version = laziness.NewWithRetry(clientWithTimeout.Discovery().ServerVersion)
	return &c, nil
}

func (c *client) RESTConfig() *rest.Config {
	if c.config == nil {
		return nil
	}
	cpy := *c.config
	return &cpy
}

var (
	_ Client    = &client{}
	_ CLIClient = &client{}
)

func (c *client) Dynamic() dynamic.Interface {
	return c.dynamic
}

func (c *client) Ext() kubeExtClient.Interface {
	return c.extSet
}

func (c *client) Kube() kubernetes.Interface {
	return c.kube
}

func (c *client) DynamicClientFor(gvk schema.GroupVersionKind, obj *unstructured.Unstructured, namespace string) (dynamic.ResourceInterface, error) {
	gvr, namespaced := c.bestEffortToGVR(gvk, obj, namespace)
	var dr dynamic.ResourceInterface
	if namespaced {
		ns := ""
		if obj != nil {
			ns = obj.GetNamespace()
		}
		if ns == "" {
			ns = namespace
		} else if namespace != "" && ns != namespace {
			return nil, fmt.Errorf("object %v/%v provided namespace %q but apply called with %q", gvk, obj.GetName(), ns, namespace)
		}
		dr = c.dynamic.Resource(gvr).Namespace(ns)
	} else {
		dr = c.dynamic.Resource(gvr)
	}
	return dr, nil
}

func (c *client) bestEffortToGVR(gvk schema.GroupVersionKind, obj *unstructured.Unstructured, namespace string) (schema.GroupVersionResource, bool) {
	if s, f := collections.All.FindByGroupVersionAliasesKind(config.FromK8sGVK(gvk)); f {
		gvr := s.GroupVersionResource()
		gvr.Version = gvk.Version
		return gvr, !s.IsClusterScoped()
	}
	if c.mapper != nil {
		mapping, err := c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err == nil {
			return mapping.Resource, mapping.Scope.Name() == meta.RESTScopeNameNamespace
		}
	}
	gvr, _ := meta.UnsafeGuessKindToResource(gvk)
	namespaced := (obj != nil && obj.GetNamespace() != "") || namespace != ""
	return gvr, namespaced
}

func WithRevision(revision string) ClientOption {
	return func(cliClient CLIClient) CLIClient {
		client := cliClient.(*client)
		client.revision = revision
		return client
	}
}
