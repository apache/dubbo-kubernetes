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
	"github.com/apache/dubbo-kubernetes/pkg/util/ptr"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/serviceregistry"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/serviceregistry/aggregate"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/serviceregistry/kube"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/serviceregistry/provider"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/visibility"
	kubelib "github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/network"
	"github.com/apache/dubbo-kubernetes/pkg/queue"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/atomic"
	"istio.io/api/label"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
)

type controllerInterface interface {
	Network(endpointIP string, labels labels.Instance) network.ID
}

var (
	log                          = dubbolog.RegisterScope("controller", "kube controller debugging")
	_   controllerInterface      = &Controller{}
	_   serviceregistry.Instance = &Controller{}
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
		log.Errorf("one or more errors force-syncing pods: %v", err)
	}
	if err := c.endpoints.initializeNamespace(metav1.NamespaceAll, true); err != nil {
		log.Errorf("one or more errors force-syncing endpoints: %v", err)
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

func (c *Controller) isControllerForProxy(proxy *model.Proxy) bool {
	return proxy.Metadata.ClusterID == "" || proxy.Metadata.ClusterID == c.Cluster()
}

func (c *Controller) GetProxyServiceTargets(proxy *model.Proxy) []model.ServiceTarget {
	if !c.isControllerForProxy(proxy) {
		log.Errorf("proxy is in cluster %v, but controller is for cluster %v", proxy.Metadata.ClusterID, c.Cluster())
		return nil
	}

	proxyNamespace := model.GetProxyConfigNamespace(proxy)
	if proxyNamespace == "" && proxy.Metadata != nil {
		proxyNamespace = proxy.Metadata.Namespace
	}

	// Get pod by proxy IP address to check if service selector matches pod labels
	// Also get pod to resolve targetPort from container ports
	// If IP is empty, try to find pod by node ID (format: podname.namespace)
	var podLabels labels.Instance
	var pod *v1.Pod
	if len(proxy.IPAddresses) > 0 {
		pods := c.pods.getPodsByIP(proxy.IPAddresses[0])
		if len(pods) > 0 {
			// Use the first pod's labels (in most cases there should be only one pod per IP)
			pod = pods[0]
			podLabels = labels.Instance(pod.Labels)
		}
	} else if proxy.ID != "" {
		// If IP address is empty, try to find pod by node ID
		// Node ID format for proxyless is: podname.namespace
		parts := strings.Split(proxy.ID, ".")
		if len(parts) >= 2 {
			podName := parts[0]
			podNamespace := parts[1]
			// Try to get pod by name and namespace
			podKey := types.NamespacedName{Name: podName, Namespace: podNamespace}
			pod = c.pods.getPodByKey(podKey)
			if pod != nil {
				// Extract IP from pod and set it to proxy
				if pod.Status.PodIP != "" {
					proxy.IPAddresses = []string{pod.Status.PodIP}
					log.Debugf("GetProxyServiceTargets: set proxy IP from pod %s/%s: %s", podNamespace, podName, pod.Status.PodIP)
				}
				podLabels = labels.Instance(pod.Labels)
			} else {
				log.Debugf("GetProxyServiceTargets: pod %s/%s not found by node ID", podNamespace, podName)
			}
		}
	}

	out := make([]model.ServiceTarget, 0)
	c.RLock()
	defer c.RUnlock()

	for _, svc := range c.servicesMap {
		// Check if service is visible to the proxy
		// 1. Service in the same namespace is always visible
		// 2. Service with ExportTo containing the namespace is visible
		// 3. Service with empty ExportTo is visible to all (default behavior)
		// 4. Service not exported (ExportTo contains None) is not visible
		visible := false

		// Check if service is exported at all
		if svc.Attributes.ExportTo != nil && svc.Attributes.ExportTo.Contains(visibility.None) {
			// Service is not exported, skip
			continue
		}

		if svc.Attributes.Namespace == proxyNamespace {
			// Service in the same namespace is always visible
			visible = true
		} else if len(svc.Attributes.ExportTo) == 0 {
			// Empty ExportTo means visible to all
			visible = true
		} else if proxyNamespace != "" {
			// Check if service is exported to this namespace
			if svc.Attributes.ExportTo.Contains(visibility.Instance(proxyNamespace)) {
				visible = true
			}
		}

		if !visible {
			continue
		}

		// For services in the same namespace, check if the service selector matches pod labels
		// This ensures we only return services that actually select this pod
		if svc.Attributes.Namespace == proxyNamespace && podLabels != nil {
			serviceSelector := labels.Instance(svc.Attributes.LabelSelectors)
			if len(serviceSelector) > 0 {
				// If service has a selector, check if it matches pod labels
				if !serviceSelector.Match(podLabels) {
					// Service selector doesn't match pod labels, skip this service
					continue
				}
			}
		}

		// Get the original Kubernetes Service to resolve targetPort
		var kubeSvc *v1.Service
		if c.services != nil {
			kubeSvc = c.services.Get(svc.Attributes.Name, svc.Attributes.Namespace)
		}

		// Create a ServiceTarget for each port
		for _, port := range svc.Ports {
			if port == nil {
				continue
			}
			targetPort := uint32(port.Port) // Default to service port

			// Try to resolve actual targetPort from Kubernetes Service and Pod
			if kubeSvc != nil {
				// Find the matching ServicePort in Kubernetes Service
				for _, kubePort := range kubeSvc.Spec.Ports {
					if kubePort.Name == port.Name && int32(kubePort.Port) == int32(port.Port) {
						// Resolve targetPort from ServicePort.TargetPort
						if kubePort.TargetPort.Type == intstr.Int {
							// TargetPort is a number
							targetPort = uint32(kubePort.TargetPort.IntVal)
						} else if kubePort.TargetPort.Type == intstr.String && pod != nil {
							// TargetPort is a string (port name), resolve from Pod container ports
							for _, container := range pod.Spec.Containers {
								for _, containerPort := range container.Ports {
									if containerPort.Name == kubePort.TargetPort.StrVal {
										targetPort = uint32(containerPort.ContainerPort)
										break
									}
								}
								if targetPort != uint32(port.Port) {
									break
								}
							}
						}
						break
					}
				}
			}

			target := model.ServiceTarget{
				Service: svc,
				Port: model.ServiceInstancePort{
					ServicePort: port,
					TargetPort:  targetPort,
				},
			}
			out = append(out, target)
		}
	}

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
	log.Debugf("Handle event %s for service %s in namespace %s", event, curr.Name, curr.Namespace)

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

	// Note: Endpoint updates are handled separately by EndpointSlice events, not here.
	// This matches Istio's behavior where service changes don't immediately update endpoints.
	// EndpointSlice events will trigger EDSUpdate (with logPushType=true) which will properly
	// log "Full push, new service" when a new endpoint shard is created.

	if serviceUpdateNeedsPush(pre, curr, prevConv, currConv) {
		log.Debugf("Service %s in namespace %s updated and needs push", currConv.Hostname, ns)
		c.handlers.NotifyServiceHandlers(prevConv, currConv, event)
	}
}

func (c *Controller) recomputeServiceForPod(pod *v1.Pod) {
	allServices := c.services.List(pod.Namespace, klabels.Everything())
	services := getPodServices(allServices, pod)
	for _, svc := range services {
		hostname := kube.ServiceHostname(svc.Name, svc.Namespace, c.opts.DomainSuffix)
		c.Lock()
		conv, f := c.servicesMap[hostname]
		c.Unlock()
		if !f {
			continue
		}
		shard := model.ShardKeyFromRegistry(c)
		endpoints := c.buildEndpointsForService(conv, true)
		// Use EDSUpdate instead of EDSCacheUpdate to trigger push notifications
		// This ensures that endpoint changes trigger XDS: Incremental Pushing logs
		c.opts.XDSUpdater.EDSUpdate(shard, string(hostname), svc.Namespace, endpoints)
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
				log.Warnf("kube controller for %s initial sync timed out", c.opts.ClusterID)
				c.initialSyncTimedout.Store(true)
			}
		})
	}
	st := time.Now()

	kubelib.WaitForCacheSync("kube controller", stop, c.informersSynced)
	log.Infof("kube controller for %s synced after %v", c.opts.ClusterID, time.Since(st))

	// after the in-order sync we can start processing the queue
	c.queue.Run(stop)
	log.Infof("Controller terminated")
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
