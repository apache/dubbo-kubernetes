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

package kube

import (
	"context"
	"fmt"
	"net/http"
	gatewayapiclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/kube/informerfactory"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/lazy"
	"github.com/apache/dubbo-kubernetes/pkg/sleep"
	"go.uber.org/atomic"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	kubeExtClient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeVersion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
)

var log = dubbolog.RegisterScope("kube", "kube client debugging")

var NewCrdWatcher func(Client) kubetypes.CrdWatcher

type client struct {
	extSet                 kubeExtClient.Interface
	config                 *rest.Config
	revision               string
	clientFactory          *clientFactory
	version                lazy.Lazy[*kubeVersion.Info]
	informerFactory        informerfactory.InformerFactory
	dynamic                dynamic.Interface
	kube                   kubernetes.Interface
	mapper                 meta.ResettableRESTMapper
	metadata               metadata.Interface
	http                   *http.Client
	objectFilter           kubetypes.DynamicObjectFilter
	clusterID              cluster.ID
	informerWatchesPending *atomic.Int32
	started                atomic.Bool
	dubbo                  istioclient.Interface
	gatewayapi             gatewayapiclient.Interface
	crdWatcher             kubetypes.CrdWatcher
	fastSync               bool
}

type Client interface {
	RESTConfig() *rest.Config

	Ext() kubeExtClient.Interface

	Kube() kubernetes.Interface

	Dynamic() dynamic.Interface

	Metadata() metadata.Interface

	Informers() informerfactory.InformerFactory

	Dubbo() istioclient.Interface

	ObjectFilter() kubetypes.DynamicObjectFilter

	ClusterID() cluster.ID

	CrdWatcher() kubetypes.CrdWatcher

	GatewayAPI() gatewayapiclient.Interface

	RunAndWait(stop <-chan struct{}) bool

	WaitForCacheSync(name string, stop <-chan struct{}, cacheSyncs ...cache.InformerSynced) bool

	Shutdown()
}

type CLIClient interface {
	Client
	DynamicClientFor(gvk schema.GroupVersionKind, obj *unstructured.Unstructured, namespace string) (dynamic.ResourceInterface, error)
}

type ClientOption func(cliClient CLIClient) CLIClient

var (
	_ Client    = &client{}
	_ CLIClient = &client{}
)

func NewClient(clientCfg clientcmd.ClientConfig, cluster cluster.ID) (Client, error) {
	return newClientInternal(newClientFactory(clientCfg, false), WithCluster(cluster))
}

func newClientInternal(clientFactory *clientFactory, opts ...ClientOption) (*client, error) {
	var c client
	var err error

	c.clientFactory = clientFactory

	c.config, err = clientFactory.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(&c)
	}

	c.mapper, err = clientFactory.mapper.Get()
	if err != nil {
		return nil, err
	}

	c.informerFactory = informerfactory.NewSharedInformerFactory()

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

	c.dubbo, err = istioclient.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}

	c.gatewayapi, err = gatewayapiclient.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}

	c.extSet, err = kubeExtClient.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}

	c.http = &http.Client{}
	if c.config != nil && c.config.Timeout != 0 {
		c.http.Timeout = c.config.Timeout
	} else {
		c.http.Timeout = time.Second * 15
	}

	var clientWithTimeout kubernetes.Interface
	clientWithTimeout = c.kube
	restConfig := c.RESTConfig()
	if restConfig != nil {
		if restConfig.Timeout == 0 {
			restConfig.Timeout = time.Second * 5
		}

		kubeClient, err := kubernetes.NewForConfig(restConfig)
		if err == nil {
			clientWithTimeout = kubeClient
		}
	}
	c.version = lazy.NewWithRetry(clientWithTimeout.Discovery().ServerVersion)
	return &c, nil
}

func NewCLIClient(clientCfg clientcmd.ClientConfig, opts ...ClientOption) (CLIClient, error) {
	return newClientInternal(newClientFactory(clientCfg, false), opts...)
}

func (c *client) RESTConfig() *rest.Config {
	if c.config == nil {
		return nil
	}
	cpy := *c.config
	return &cpy
}

func EnableCrdWatcher(c Client) Client {
	if NewCrdWatcher == nil {
		panic("NewCrdWatcher is unset. Likely the crd watcher library is not imported anywhere")
	}
	if c.(*client).crdWatcher != nil {
		panic("EnableCrdWatcher called twice for the same client")
	}
	c.(*client).crdWatcher = NewCrdWatcher(c)
	return c
}

func (c *client) Ext() kubeExtClient.Interface {
	return c.extSet
}

func (c *client) Kube() kubernetes.Interface {
	return c.kube
}

func (c *client) ClusterID() cluster.ID {
	return c.clusterID
}

func (c *client) Dynamic() dynamic.Interface {
	return c.dynamic
}

func (c *client) Metadata() metadata.Interface {
	return c.metadata
}

func (c *client) Informers() informerfactory.InformerFactory {
	return c.informerFactory
}

func (c *client) Dubbo() istioclient.Interface {
	return c.dubbo
}

func (c *client) GatewayAPI() gatewayapiclient.Interface {
	return c.gatewayapi
}

func (c *client) ObjectFilter() kubetypes.DynamicObjectFilter {
	return c.objectFilter
}

func (c *client) CrdWatcher() kubetypes.CrdWatcher {
	return c.crdWatcher
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

func (c *client) WaitForCacheSync(name string, stop <-chan struct{}, cacheSyncs ...cache.InformerSynced) bool {
	if c.informerWatchesPending == nil {
		return WaitForCacheSync(name, stop, cacheSyncs...)
	}
	syncFns := append(cacheSyncs, func() bool {
		return c.informerWatchesPending.Load() == 0
	})
	return WaitForCacheSync(name, stop, syncFns...)
}

func (c *client) Run(stop <-chan struct{}) {
	c.informerFactory.Start(stop)
	if c.crdWatcher != nil {
		go c.crdWatcher.Run(stop)
	}
	alreadyStarted := c.started.Swap(true)
	if alreadyStarted {
		log.Debugf("cluster %q kube client started again", c.clusterID)
	} else {
		log.Infof("cluster %q kube client started", c.clusterID)
	}
}

func (c *client) RunAndWait(stop <-chan struct{}) bool {
	c.Run(stop)
	if c.fastSync {
		if c.crdWatcher != nil {
			if !c.WaitForCacheSync("crd watcher", stop, c.crdWatcher.HasSynced) {
				return false
			}
		}
		// WaitForCacheSync will virtually never be synced on the first call, as its called immediately after Start()
		// This triggers a 100ms delay per call, which is often called 2-3 times in a test, delaying tests.
		// Instead, we add an aggressive sync polling
		if !fastWaitForCacheSync(stop, c.informerFactory) {
			return false
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			<-stop
			cancel()
		}()
		err := wait.PollUntilContextTimeout(ctx, time.Microsecond*100, wait.ForeverTestTimeout, true, func(ctx context.Context) (bool, error) {
			select {
			case <-stop:
				return false, fmt.Errorf("channel closed")
			case <-ctx.Done():
				return false, ctx.Err()
			default:
			}
			if c.informerWatchesPending.Load() == 0 {
				return true, nil
			}
			return false, nil
		})
		return err == nil
	}
	if c.crdWatcher != nil {
		if !c.WaitForCacheSync("crd watcher", stop, c.crdWatcher.HasSynced) {
			return false
		}
	}
	return c.informerFactory.WaitForCacheSync(stop)
}

func (c *client) Shutdown() {
	c.informerFactory.Shutdown()
}

func (c *client) bestEffortToGVR(gvk schema.GroupVersionKind, obj *unstructured.Unstructured, namespace string) (schema.GroupVersionResource, bool) {
	if s, f := collections.All.FindByGroupVersionAliasesKind(config.FromKubernetesGVK(gvk)); f {
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

func WaitForCacheSync(name string, stop <-chan struct{}, cacheSyncs ...cache.InformerSynced) (r bool) {
	t0 := time.Now()
	maximum := time.Millisecond * 100
	delay := time.Millisecond
	f := func() bool {
		for _, syncFunc := range cacheSyncs {
			if !syncFunc() {
				return false
			}
		}
		return true
	}
	attempt := 0
	defer func() {
		if r {
			log.Infof("sync complete: name=%s, time=%v", name, time.Since(t0))
		} else {
			log.Infof("sync failed: name=%s, time=%v", name, time.Since(t0))
		}
	}()
	for {
		select {
		case <-stop:
			return false
		default:
		}
		attempt++
		res := f()
		if res {
			return true
		}
		delay *= 2
		if delay > maximum {
			delay = maximum
		}
		if attempt%50 == 0 {
			// Log every 50th attempt (5s) at info, to avoid too much noisy
			fmt.Printf("waiting for sync...: name=%s, time=%v\n", name, time.Since(t0))

		}
		if !sleep.Until(stop, delay) {
			return false
		}
	}
}

func fastWaitForCacheSync(stop <-chan struct{}, informerFactory informerfactory.InformerFactory) bool {
	returnImmediately := make(chan struct{})
	close(returnImmediately)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-stop
		cancel()
	}()
	err := wait.PollUntilContextTimeout(ctx, time.Microsecond*100, wait.ForeverTestTimeout, true, func(pollCtx context.Context) (bool, error) {
		select {
		case <-stop:
			return false, fmt.Errorf("channel closed")
		case <-pollCtx.Done():
			return false, pollCtx.Err()
		default:
		}
		return informerFactory.WaitForCacheSync(returnImmediately), nil
	})
	return err == nil
}

func SetObjectFilter(c Client, filter kubetypes.DynamicObjectFilter) Client {
	c.(*client).objectFilter = filter
	return c
}

func WithCluster(id cluster.ID) ClientOption {
	return func(c CLIClient) CLIClient {
		client := c.(*client)
		client.clusterID = id
		return client
	}
}

func WithRevision(revision string) ClientOption {
	return func(cliClient CLIClient) CLIClient {
		client := cliClient.(*client)
		client.revision = revision
		return client
	}
}
