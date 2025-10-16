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

package controller

import (
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/config/visibility"
	kubelib "github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/network"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"github.com/apache/dubbo-kubernetes/pkg/queue"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/aggregate"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/kube"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/atomic"
	"istio.io/api/label"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sort"
	"sync"
	"time"
)

type controllerInterface interface {
	Network(endpointIP string, labels labels.Instance) network.ID
}

var (
	_ controllerInterface      = &Controller{}
	_ serviceregistry.Instance = &Controller{}
)

type Controller struct {
	opts   Options
	client kubelib.Client
	sync.RWMutex
	servicesMap         map[host.Name]*model.Service
	queue               queue.Instance
	initialSyncTimedout *atomic.Bool
	configCluster       bool
	services            kclient.Client[*v1.Service]
	endpoints           *endpointSliceController
	podsClient          kclient.Client[*v1.Pod]
	namespaces          kclient.Client[*v1.Namespace]

	meshWatcher mesh.Watcher
	handlers    model.ControllerHandlers
	pods        *PodCache
	*networkManager
}

func NewController(kubeClient kubelib.Client, options Options) *Controller {
	c := &Controller{
		opts:                options,
		client:              kubeClient,
		queue:               queue.NewQueueWithID(1*time.Second, string(options.ClusterID)),
		servicesMap:         make(map[host.Name]*model.Service),
		initialSyncTimedout: atomic.NewBool(false),

		configCluster: options.ConfigCluster,
	}
	c.networkManager = initNetworkManager(c, options)

	c.namespaces = kclient.New[*v1.Namespace](kubeClient)
	if c.opts.SystemNamespace != "" {
		registerHandlers[*v1.Namespace](
			c,
			c.namespaces,
			"Namespaces",
			func(old *v1.Namespace, cur *v1.Namespace, event model.Event) error {
				if cur.Name == c.opts.SystemNamespace {
					return c.onSystemNamespaceEvent(old, cur, event)
				}
				return nil
			},
			nil,
		)
	}

	c.services = kclient.NewFiltered[*v1.Service](kubeClient, kclient.Filter{ObjectFilter: kubeClient.ObjectFilter()})
	registerHandlers(c, c.services, "Services", c.onServiceEvent, nil)
	c.endpoints = newEndpointSliceController(c)

	c.podsClient = kclient.NewFiltered[*v1.Pod](kubeClient, kclient.Filter{
		ObjectFilter:    kubeClient.ObjectFilter(),
		ObjectTransform: kubelib.StripPodUnusedFields,
	})
	c.pods = newPodCache(c, c.podsClient, func(key types.NamespacedName) {
		c.queue.Push(func() error {
			return c.endpoints.podArrived(key.Name, key.Namespace)
		})
	})
	registerHandlers[*v1.Pod](c, c.podsClient, "Pods", c.pods.onEvent, nil)
	c.meshWatcher = options.MeshWatcher
	return c
}

func (c *Controller) onSystemNamespaceEvent(_, ns *v1.Namespace, ev model.Event) error {
	if ev == model.EventDelete {
		return nil
	}
	if c.setNetworkFromNamespace(ns) {
		// network changed, rarely happen
		// refresh pods/endpoints/services
		c.onNetworkChange()
	}
	return nil
}

func (c *Controller) onNetworkChange() {
	// the network for endpoints are computed when we process the events; this will fix the cache
	// NOTE: this must run before the other network watcher handler that creates a force push
	if err := c.syncPods(); err != nil {
		klog.Errorf("one or more errors force-syncing pods: %v", err)
	}
	if err := c.endpoints.initializeNamespace(metav1.NamespaceAll, true); err != nil {
		klog.Errorf("one or more errors force-syncing endpoints: %v", err)
	}

}

func (c *Controller) syncPods() error {
	var err *multierror.Error
	pods := c.podsClient.List(metav1.NamespaceAll, klabels.Everything())
	for _, s := range pods {
		err = multierror.Append(err, c.pods.onEvent(nil, s, model.EventAdd))
	}
	return err.ErrorOrNil()
}

type Options struct {
	KubernetesAPIQPS      float32
	KubernetesAPIBurst    int
	DomainSuffix          string
	XDSUpdater            model.XDSUpdater
	MeshNetworksWatcher   mesh.NetworksWatcher
	MeshWatcher           meshwatcher.WatcherCollection
	ClusterID             cluster.ID
	ClusterAliases        map[string]string
	SystemNamespace       string
	MeshServiceController *aggregate.Controller
	KrtDebugger           *krt.DebugHandler
	SyncTimeout           time.Duration
	Revision              string
	ConfigCluster         bool
}

func (c *Controller) Services() []*model.Service {
	c.RLock()
	out := make([]*model.Service, 0, len(c.servicesMap))
	for _, svc := range c.servicesMap {
		out = append(out, svc)
	}
	c.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Hostname < out[j].Hostname })
	return out
}

// GetService implements a service catalog operation by hostname specified.
func (c *Controller) GetService(hostname host.Name) *model.Service {
	c.RLock()
	svc := c.servicesMap[hostname]
	c.RUnlock()
	return svc
}

func (c *Controller) onServiceEvent(pre, curr *v1.Service, event model.Event) error {
	klog.V(2).Infof("Handle event %s for service %s in namespace %s", event, curr.Name, curr.Namespace)

	// Create the standard (cluster.local) service.
	svcConv := kube.ConvertService(*curr, c.opts.DomainSuffix, c.Cluster(), c.meshWatcher.Mesh())

	switch event {
	case model.EventDelete:
		c.deleteService(svcConv)
	default:
		c.addOrUpdateService(pre, curr, svcConv, event, false)
	}

	return nil
}

func (c *Controller) deleteService(svc *model.Service) {
	c.Lock()
	delete(c.servicesMap, svc.Hostname)
	c.Unlock()

	shard := model.ShardKeyFromRegistry(c)
	event := model.EventDelete
	c.opts.XDSUpdater.ServiceUpdate(shard, string(svc.Hostname), svc.Attributes.Namespace, event)
	if !svc.Attributes.ExportTo.Contains(visibility.None) {
		c.handlers.NotifyServiceHandlers(nil, svc, event)
	}
}

func (c *Controller) addOrUpdateService(pre, curr *v1.Service, currConv *model.Service, event model.Event, updateEDSCache bool) {
	c.Lock()
	prevConv := c.servicesMap[currConv.Hostname]
	c.servicesMap[currConv.Hostname] = currConv
	c.Unlock()
	// This full push needed to update all endpoints, even though we do a full push on service add/update
	// as that full push is only triggered for the specific service.

	shard := model.ShardKeyFromRegistry(c)
	ns := currConv.Attributes.Namespace

	c.opts.XDSUpdater.ServiceUpdate(shard, string(currConv.Hostname), ns, event)
	if serviceUpdateNeedsPush(pre, curr, prevConv, currConv) {
		klog.V(2).Infof("Service %s in namespace %s updated and needs push", currConv.Hostname, ns)
		c.handlers.NotifyServiceHandlers(prevConv, currConv, event)
	}
}

func (c *Controller) recomputeServiceForPod(pod *v1.Pod) {
	allServices := c.services.List(pod.Namespace, klabels.Everything())
	cu := sets.New[model.ConfigKey]()
	services := getPodServices(allServices, pod)
	for _, svc := range services {
		hostname := kube.ServiceHostname(svc.Name, svc.Namespace, c.opts.DomainSuffix)
		c.Lock()
		conv, f := c.servicesMap[hostname]
		c.Unlock()
		if !f {
			return
		}
		shard := model.ShardKeyFromRegistry(c)
		endpoints := c.buildEndpointsForService(conv, true)
		if len(endpoints) > 0 {
			c.opts.XDSUpdater.EDSCacheUpdate(shard, string(hostname), svc.Namespace, endpoints)
		}
		cu.Insert(model.ConfigKey{
			Kind:      kind.ServiceEntry,
			Name:      string(hostname),
			Namespace: svc.Namespace,
		})
	}
	if len(cu) > 0 {
		c.opts.XDSUpdater.ConfigUpdate(&model.PushRequest{
			Full:           false,
			ConfigsUpdated: cu,
			Reason:         model.NewReasonStats(model.EndpointUpdate),
		})
	}
}

func (c *Controller) buildEndpointsForService(svc *model.Service, updateCache bool) []*model.DubboEndpoint {
	endpoints := c.endpoints.buildDubboEndpointsWithService(svc.Attributes.Name, svc.Attributes.Namespace, svc.Hostname, updateCache)
	return endpoints
}

func serviceUpdateNeedsPush(prev, curr *v1.Service, preConv, currConv *model.Service) bool {
	// New Service - If it is not exported, no need to push.
	if preConv == nil {
		return !currConv.Attributes.ExportTo.Contains(visibility.None)
	}
	// if service Visibility is None and has not changed in the update/delete, no need to push.
	if preConv.Attributes.ExportTo.Contains(visibility.None) &&
		currConv.Attributes.ExportTo.Contains(visibility.None) {
		return false
	}
	// Check if there are any changes we care about by comparing `model.Service`s
	if !preConv.Equals(currConv) {
		return true
	}
	// Also check if target ports are changed since they are not included in `model.Service`
	// `preConv.Equals(currConv)` already makes sure the length of ports is not changed
	if prev != nil && curr != nil {
		if !slices.EqualFunc(prev.Spec.Ports, curr.Spec.Ports, func(a, b v1.ServicePort) bool {
			return a.TargetPort == b.TargetPort
		}) {
			return true
		}
	}
	return false
}

func (c *Controller) Provider() provider.ID {
	return provider.Kubernetes
}

func (c *Controller) Cluster() cluster.ID {
	return c.opts.ClusterID
}

func (c *Controller) Run(stop <-chan struct{}) {
	if c.opts.SyncTimeout != 0 {
		time.AfterFunc(c.opts.SyncTimeout, func() {
			if !c.queue.HasSynced() {
				klog.Warningf("kube controller for %s initial sync timed out", c.opts.ClusterID)
				c.initialSyncTimedout.Store(true)
			}
		})
	}
	st := time.Now()

	kubelib.WaitForCacheSync("kube controller", stop, c.informersSynced)
	klog.Infof("kube controller for %s synced after %v", c.opts.ClusterID, time.Since(st))

	// after the in-order sync we can start processing the queue
	c.queue.Run(stop)
	klog.Infof("Controller terminated")
}

func (c *Controller) HasSynced() bool {
	if c.initialSyncTimedout.Load() {
		return true
	}
	return c.queue.HasSynced()
}

func (c *Controller) informersSynced() bool {
	return c.namespaces.HasSynced() &&
		c.pods.pods.HasSynced() &&
		c.services.HasSynced() &&
		c.endpoints.slices.HasSynced() &&
		c.networkManager.HasSynced()
}

func (c *Controller) hostNamesForNamespacedName(name types.NamespacedName) []host.Name {
	return []host.Name{
		kube.ServiceHostname(name.Name, name.Namespace, c.opts.DomainSuffix),
	}
}

type FilterOutFunc[T controllers.Object] func(old, cur T) bool

func registerHandlers[T controllers.ComparableObject](c *Controller,
	informer kclient.Informer[T], otype string,
	handler func(T, T, model.Event) error, filter FilterOutFunc[T],
) {
	wrappedHandler := func(prev, curr T, event model.Event) error {
		curr = informer.Get(curr.GetName(), curr.GetNamespace())
		if controllers.IsNil(curr) {
			// this can happen when an immediate delete after update
			// the delete event can be handled later
			return nil
		}
		return handler(prev, curr, event)
	}

	informer.AddEventHandler(
		controllers.EventHandler[T]{
			AddFunc: func(obj T) {
				c.queue.Push(func() error {
					return wrappedHandler(ptr.Empty[T](), obj, model.EventAdd)
				})
			},
			UpdateFunc: func(old, cur T) {
				if filter != nil {
					if filter(old, cur) {
						return
					}
				}
				c.queue.Push(func() error {
					return wrappedHandler(old, cur, model.EventUpdate)
				})
			},
			DeleteFunc: func(obj T) {
				c.queue.Push(func() error {
					return handler(ptr.Empty[T](), obj, model.EventDelete)
				})
			},
		})
}

func (c *Controller) servicesForNamespacedName(name types.NamespacedName) []*model.Service {
	if svc := c.GetService(kube.ServiceHostname(name.Name, name.Namespace, c.opts.DomainSuffix)); svc != nil {
		return []*model.Service{svc}
	}
	return nil
}

func (c *Controller) Network(endpointIP string, labels labels.Instance) network.ID {
	// 1. check the pod/workloadEntry label
	if nw := labels[label.TopologyNetwork.Name]; nw != "" {
		return network.ID(nw)
	}
	// 2. check the system namespace labels
	if nw := c.networkFromSystemNamespace(); nw != "" {
		return nw
	}

	// 3. check the meshNetworks config
	if nw := c.networkFromMeshNetworks(endpointIP); nw != "" {
		return nw
	}
	return ""
}
