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

package serviceentry

import (
	"sort"
	"strconv"
	"sync"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/serviceregistry/provider"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/config/visibility"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	networking "github.com/kdubbo/api/networking/v1alpha3"
)

type Options struct {
	ConfigController model.ConfigStoreController
	XDSUpdater       model.XDSUpdater
	ClusterID        cluster.ID
}

type Controller struct {
	mu               sync.RWMutex
	reconcileMu      sync.Mutex
	configController model.ConfigStoreController
	xdsUpdater       model.XDSUpdater
	clusterID        cluster.ID
	services         map[host.Name]*model.Service
	endpoints        map[string][]*model.DubboEndpoint
}

func NewController(opts Options) *Controller {
	c := &Controller{
		configController: opts.ConfigController,
		xdsUpdater:       opts.XDSUpdater,
		clusterID:        opts.ClusterID,
		services:         make(map[host.Name]*model.Service),
		endpoints:        make(map[string][]*model.DubboEndpoint),
	}
	opts.ConfigController.RegisterEventHandler(gvk.ServiceEntry, c.onConfigEvent)
	opts.ConfigController.RegisterEventHandler(gvk.WorkloadEntry, c.onConfigEvent)
	return c
}

func (c *Controller) onConfigEvent(_, _ config.Config, _ model.Event) {
	c.reconcile()
}

func (c *Controller) reconcile() {
	c.reconcileMu.Lock()
	defer c.reconcileMu.Unlock()

	entries := c.configController.List(gvk.ServiceEntry, model.NamespaceAll)
	workloads := c.configController.List(gvk.WorkloadEntry, model.NamespaceAll)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].CreationTimestamp.Equal(entries[j].CreationTimestamp) {
			if entries[i].Namespace == entries[j].Namespace {
				return entries[i].Name < entries[j].Name
			}
			return entries[i].Namespace < entries[j].Namespace
		}
		return entries[i].CreationTimestamp.Before(entries[j].CreationTimestamp)
	})

	services := make(map[host.Name]*model.Service)
	endpoints := make(map[string][]*model.DubboEndpoint)
	for _, cfg := range entries {
		entry, ok := cfg.Spec.(*networking.ServiceEntry)
		if !ok {
			continue
		}
		selected := selectWorkloads(cfg.Namespace, entry, workloads)
		for _, hostname := range entry.GetHosts() {
			h := host.Name(hostname)
			if _, found := services[h]; found {
				continue
			}
			svc := c.convertService(cfg, entry, h)
			services[h] = svc
			endpoints[svc.Key()] = buildEndpoints(cfg, entry, svc, selected)
		}
	}

	c.mu.Lock()
	previous := c.services
	previousEndpoints := c.endpoints
	c.services = services
	c.endpoints = endpoints
	c.mu.Unlock()

	if c.xdsUpdater == nil {
		return
	}
	shard := model.ShardKeyFromRegistry(c)
	for hostname, svc := range services {
		oldService := previous[hostname]
		if oldService == nil {
			c.xdsUpdater.ServiceUpdate(shard, string(hostname), svc.Attributes.Namespace, model.EventAdd)
			c.xdsUpdater.EDSUpdate(shard, string(hostname), svc.Attributes.Namespace, endpoints[svc.Key()])
			continue
		}
		if oldService.Attributes.Namespace != svc.Attributes.Namespace {
			c.xdsUpdater.ServiceUpdate(shard, string(hostname), oldService.Attributes.Namespace, model.EventDelete)
			c.xdsUpdater.EDSUpdate(shard, string(hostname), oldService.Attributes.Namespace, nil)
			c.xdsUpdater.ServiceUpdate(shard, string(hostname), svc.Attributes.Namespace, model.EventAdd)
			c.xdsUpdater.EDSUpdate(shard, string(hostname), svc.Attributes.Namespace, endpoints[svc.Key()])
			continue
		}
		if !oldService.Equals(svc) {
			c.xdsUpdater.ServiceUpdate(shard, string(hostname), svc.Attributes.Namespace, model.EventUpdate)
		}
		if !slices.EqualFunc(previousEndpoints[oldService.Key()], endpoints[svc.Key()], func(a, b *model.DubboEndpoint) bool {
			return a.Equals(b)
		}) {
			c.xdsUpdater.EDSUpdate(shard, string(hostname), svc.Attributes.Namespace, endpoints[svc.Key()])
		}
	}
	for hostname, svc := range previous {
		if _, found := services[hostname]; found {
			continue
		}
		c.xdsUpdater.ServiceUpdate(shard, string(hostname), svc.Attributes.Namespace, model.EventDelete)
		c.xdsUpdater.EDSUpdate(shard, string(hostname), svc.Attributes.Namespace, nil)
	}
}

func (c *Controller) convertService(cfg config.Config, entry *networking.ServiceEntry, hostname host.Name) *model.Service {
	addresses := append([]string(nil), entry.GetAddresses()...)
	defaultAddress := constants.UnspecifiedIP
	if len(addresses) > 0 {
		defaultAddress = addresses[0]
	} else {
		addresses = []string{defaultAddress}
	}
	ports := make(model.PortList, 0, len(entry.GetPorts()))
	for _, port := range entry.GetPorts() {
		ports = append(ports, &model.Port{Name: port.GetName(), Port: int(port.GetNumber()), Protocol: protocol.Parse(port.GetProtocol())})
	}
	exportTo := sets.New[visibility.Instance]()
	for _, namespace := range entry.GetExportTo() {
		exportTo.Insert(visibility.Instance(namespace))
	}
	service := &model.Service{
		Hostname:        hostname,
		Ports:           ports,
		ServiceAccounts: append([]string(nil), entry.GetSubjectAltNames()...),
		ClusterVIPs: model.AddressMap{Addresses: map[cluster.ID][]string{
			c.clusterID: addresses,
		}},
		CreationTime:    cfg.CreationTimestamp,
		DefaultAddress:  defaultAddress,
		ResourceVersion: cfg.ResourceVersion,
		Resolution:      convertResolution(entry.GetResolution()),
		MeshExternal:    entry.GetLocation() == networking.ServiceEntry_MESH_EXTERNAL,
		Attributes: model.ServiceAttributes{
			Name:            cfg.Name,
			Namespace:       cfg.Namespace,
			ExportTo:        exportTo,
			ServiceRegistry: provider.External,
		},
	}
	service.Attributes.ObjectName = cfg.Name
	return service
}

func convertResolution(resolution networking.ServiceEntry_Resolution) model.Resolution {
	switch resolution {
	case networking.ServiceEntry_STATIC:
		return model.ClientSideLB
	case networking.ServiceEntry_DNS:
		return model.DNSLB
	case networking.ServiceEntry_DNS_ROUND_ROBIN:
		return model.DNSRoundRobinLB
	default:
		return model.Passthrough
	}
}

func selectWorkloads(namespace string, entry *networking.ServiceEntry, configs []config.Config) []namedWorkload {
	selector := entry.GetWorkloadSelector()
	if selector == nil {
		out := make([]namedWorkload, 0, len(entry.GetEndpoints()))
		for i, workload := range entry.GetEndpoints() {
			out = append(out, namedWorkload{name: entryName(i), workload: workload})
		}
		return out
	}
	wanted := labels.Instance(selector.GetMatchLabels())
	out := make([]namedWorkload, 0)
	for _, cfg := range configs {
		if cfg.Namespace != namespace {
			continue
		}
		workload, ok := cfg.Spec.(*networking.WorkloadEntry)
		if ok && wanted.Match(labels.Instance(workload.GetLabels())) {
			out = append(out, namedWorkload{name: cfg.Name, workload: workload})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

type namedWorkload struct {
	name     string
	workload *networking.WorkloadEntry
}

func entryName(index int) string {
	return "inline-" + strconv.Itoa(index)
}

func buildEndpoints(cfg config.Config, entry *networking.ServiceEntry, svc *model.Service, workloads []namedWorkload) []*model.DubboEndpoint {
	if len(workloads) == 0 && (entry.GetResolution() == networking.ServiceEntry_DNS || entry.GetResolution() == networking.ServiceEntry_DNS_ROUND_ROBIN) {
		workloads = []namedWorkload{{name: cfg.Name, workload: &networking.WorkloadEntry{Address: string(svc.Hostname)}}}
	}
	result := make([]*model.DubboEndpoint, 0, len(workloads)*len(entry.GetPorts()))
	for _, named := range workloads {
		for _, port := range entry.GetPorts() {
			targetPort := named.workload.GetPorts()[port.GetName()]
			if targetPort == 0 {
				targetPort = port.GetTargetPort()
			}
			if targetPort == 0 {
				targetPort = port.GetNumber()
			}
			result = append(result, &model.DubboEndpoint{
				ServiceAccount:  named.workload.GetServiceAccount(),
				Addresses:       []string{named.workload.GetAddress()},
				ServicePortName: port.GetName(),
				Labels:          labels.Instance(named.workload.GetLabels()),
				HealthStatus:    model.Healthy,
				EndpointPort:    targetPort,
				WorkloadName:    named.name,
				Namespace:       cfg.Namespace,
			})
		}
	}
	return result
}

func (c *Controller) Services() []*model.Service {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]*model.Service, 0, len(c.services))
	for _, svc := range c.services {
		out = append(out, svc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Hostname < out[j].Hostname })
	return out
}

func (c *Controller) GetService(hostname host.Name) *model.Service {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.services[hostname]
}

func (c *Controller) GetProxyServiceTargets(proxy *model.Proxy) []model.ServiceTarget {
	if proxy == nil {
		return nil
	}
	addresses := sets.New(proxy.IPAddresses...)
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []model.ServiceTarget
	for _, svc := range c.services {
		for _, endpoint := range c.endpoints[svc.Key()] {
			if !addresses.Contains(endpoint.FirstAddressOrNil()) {
				continue
			}
			for _, port := range svc.Ports {
				if port.Name == endpoint.ServicePortName {
					out = append(out, model.ServiceTarget{Service: svc, Port: model.ServiceInstancePort{ServicePort: port, TargetPort: endpoint.EndpointPort}})
				}
			}
		}
	}
	return out
}

func (c *Controller) Provider() provider.ID { return provider.External }
func (c *Controller) Cluster() cluster.ID   { return c.clusterID }
func (c *Controller) HasSynced() bool       { return c.configController.HasSynced() }

func (c *Controller) Run(stop <-chan struct{}) {
	c.reconcile()
	<-stop
}
