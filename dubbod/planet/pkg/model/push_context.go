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

package model

import (
	"cmp"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	networking "istio.io/api/networking/v1alpha3"
	"sort"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/serviceregistry/provider"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/config/visibility"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	"github.com/apache/dubbo-kubernetes/pkg/spiffe"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	"go.uber.org/atomic"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

type TriggerReason string

const (
	UnknownTrigger         TriggerReason = "unknown"
	ProxyRequest           TriggerReason = "proxyrequest"
	GlobalUpdate           TriggerReason = "global"
	HeadlessEndpointUpdate TriggerReason = "headlessendpoint"
	EndpointUpdate         TriggerReason = "endpoint"
	ProxyUpdate            TriggerReason = "proxy"
	DependentResource      TriggerReason = "depdendentresource"
)

var (
	LastPushStatus *PushContext
	LastPushMutex  sync.Mutex
)

type PushContext struct {
	Mesh              *meshconfig.MeshConfig `json:"-"`
	initializeMutex   sync.Mutex
	InitDone          atomic.Bool
	Networks          *meshconfig.MeshNetworks
	networkMgr        *NetworkManager
	clusterLocalHosts ClusterLocalHosts
	exportToDefaults  exportToDefaults
	ServiceIndex      serviceIndex
	serviceRouteIndex serviceRouteIndex
	subsetRuleIndex   subsetRuleIndex
	serviceAccounts   map[serviceAccountKey][]string
	PushVersion       string
	ProxyStatus       map[string]map[string]ProxyPushStatus
	proxyStatusMutex  sync.RWMutex
}

type PushRequest struct {
	Reason           ReasonStats
	ConfigsUpdated   sets.Set[ConfigKey]
	AddressesUpdated sets.Set[string]
	Forced           bool
	Full             bool
	Push             *PushContext
	Start            time.Time
	Delta            ResourceDelta
}

type XDSUpdater interface {
	ConfigUpdate(req *PushRequest)
	ServiceUpdate(shard ShardKey, hostname string, namespace string, event Event)
	EDSUpdate(shard ShardKey, hostname string, namespace string, entry []*DubboEndpoint)
	EDSCacheUpdate(shard ShardKey, hostname string, namespace string, entry []*DubboEndpoint)
	ProxyUpdate(clusterID cluster.ID, ip string)
}

type ProxyPushStatus struct {
	Proxy   string `json:"proxy,omitempty"`
	Message string `json:"message,omitempty"`
}

type serviceAccountKey struct {
	hostname  host.Name
	namespace string
}

type serviceIndex struct {
	privateByNamespace   map[string][]*Service
	public               []*Service
	exportedToNamespace  map[string][]*Service
	HostnameAndNamespace map[host.Name]map[string]*Service `json:"-"`
	instancesByPort      map[string]map[int][]*DubboEndpoint
}

type ReasonStats map[TriggerReason]int

type ResourceDelta = xds.ResourceDelta

type ConfigKey struct {
	Kind      kind.Kind
	Name      string
	Namespace string
}

type ConsolidatedSubRule struct {
	exportTo sets.Set[visibility.Instance]
	rule     *config.Config
	from     []types.NamespacedName
}

type exportToDefaults struct {
	service      sets.Set[visibility.Instance]
	serviceRoute sets.Set[visibility.Instance]
	subsetRule   sets.Set[visibility.Instance]
}

type serviceRouteIndex struct {
	// root vs namespace/name ->delegate vs virtualservice gvk/namespace/name
	delegates map[ConfigKey][]ConfigKey

	// Map of VS hostname -> referenced hostnames
	referencedDestinations map[string]sets.String
}

type subsetRuleIndex struct {
	namespaceLocal      map[string]*consolidatedSubRules
	exportedByNamespace map[string]*consolidatedSubRules
	rootNamespaceLocal  *consolidatedSubRules
}

type consolidatedSubRules struct {
	specificSubRules map[host.Name][]*ConsolidatedSubRule
	wildcardSubRules map[host.Name][]*ConsolidatedSubRule
}

func NewPushContext() *PushContext {
	return &PushContext{
		ServiceIndex:      newServiceIndex(),
		serviceRouteIndex: newServiceRouteIndex(),
		subsetRuleIndex:   newSubsetRuleIndex(),
		serviceAccounts:   map[serviceAccountKey][]string{},
	}
}

func newServiceIndex() serviceIndex {
	return serviceIndex{
		public:               []*Service{},
		privateByNamespace:   map[string][]*Service{},
		exportedToNamespace:  map[string][]*Service{},
		HostnameAndNamespace: map[host.Name]map[string]*Service{},
		instancesByPort:      map[string]map[int][]*DubboEndpoint{},
	}
}

func newServiceRouteIndex() serviceRouteIndex {
	out := serviceRouteIndex{
		delegates:              map[ConfigKey][]ConfigKey{},
		referencedDestinations: map[string]sets.String{},
	}
	return out
}

func newSubsetRuleIndex() subsetRuleIndex {
	return subsetRuleIndex{
		namespaceLocal:      map[string]*consolidatedSubRules{},
		exportedByNamespace: map[string]*consolidatedSubRules{},
	}
}

func newConsolidatedDestRules() *consolidatedSubRules {
	return &consolidatedSubRules{
		specificSubRules: map[host.Name][]*ConsolidatedSubRule{},
		wildcardSubRules: map[host.Name][]*ConsolidatedSubRule{},
	}
}

func NewReasonStats(reasons ...TriggerReason) ReasonStats {
	ret := make(ReasonStats)
	for _, reason := range reasons {
		ret.Add(reason)
	}
	return ret
}

func (r ReasonStats) Has(reason TriggerReason) bool {
	return r[reason] > 0
}

func (r ReasonStats) Add(reason TriggerReason) {
	r[reason]++
}

func (r ReasonStats) Merge(other ReasonStats) {
	for reason, count := range other {
		r[reason] += count
	}
}

func (r ReasonStats) Count() int {
	var ret int
	for _, count := range r {
		ret += count
	}
	return ret
}

func (pr *PushRequest) Merge(other *PushRequest) *PushRequest {
	if pr == nil {
		return other
	}
	if other == nil {
		return pr
	}

	// Keep the first (older) start time

	// Merge the two reasons. Note that we shouldn't deduplicate here, or we would under count
	if len(other.Reason) > 0 {
		if pr.Reason == nil {
			pr.Reason = make(map[TriggerReason]int)
		}
		pr.Reason.Merge(other.Reason)
	}

	// If either is full we need a full push
	pr.Full = pr.Full || other.Full

	// If either is forced we need a forced push
	pr.Forced = pr.Forced || other.Forced

	// The other push context is presumed to be later and more up to date
	if other.Push != nil {
		pr.Push = other.Push
	}

	if pr.ConfigsUpdated == nil {
		pr.ConfigsUpdated = other.ConfigsUpdated
	} else {
		pr.ConfigsUpdated.Merge(other.ConfigsUpdated)
	}

	if pr.AddressesUpdated == nil {
		pr.AddressesUpdated = other.AddressesUpdated
	} else {
		pr.AddressesUpdated.Merge(other.AddressesUpdated)
	}

	return pr
}

func (pr *PushRequest) CopyMerge(other *PushRequest) *PushRequest {
	if pr == nil {
		return other
	}
	if other == nil {
		return pr
	}

	merged := &PushRequest{}
	return merged
}

func (pr *PushRequest) IsProxyUpdate() bool {
	return pr.Reason.Has(ProxyUpdate)
}

func (pr *PushRequest) IsRequest() bool {
	return len(pr.Reason) == 1 && pr.Reason.Has(ProxyRequest)
}

func (pr *PushRequest) PushReason() string {
	if pr.IsRequest() {
		return " request"
	}
	return ""
}

func (ps *PushContext) initDefaultExportMaps() {
	ps.exportToDefaults.subsetRule = sets.New[visibility.Instance]()
	if ps.Mesh.DefaultDestinationRuleExportTo != nil {
		for _, e := range ps.Mesh.DefaultDestinationRuleExportTo {
			ps.exportToDefaults.subsetRule.Insert(visibility.Instance(e))
		}
	} else {
		// default to *
		ps.exportToDefaults.subsetRule.Insert(visibility.Public)
	}

	ps.exportToDefaults.service = sets.New[visibility.Instance]()
	if ps.Mesh.DefaultServiceExportTo != nil {
		for _, e := range ps.Mesh.DefaultServiceExportTo {
			ps.exportToDefaults.service.Insert(visibility.Instance(e))
		}
	} else {
		ps.exportToDefaults.service.Insert(visibility.Public)
	}

	ps.exportToDefaults.serviceRoute = sets.New[visibility.Instance]()
	if ps.Mesh.DefaultVirtualServiceExportTo != nil {
		for _, e := range ps.Mesh.DefaultVirtualServiceExportTo {
			ps.exportToDefaults.serviceRoute.Insert(visibility.Instance(e))
		}
	} else {
		ps.exportToDefaults.serviceRoute.Insert(visibility.Public)
	}
}

func (ps *PushContext) InitContext(env *Environment, oldPushContext *PushContext, pushReq *PushRequest) {
	ps.initializeMutex.Lock()
	defer ps.initializeMutex.Unlock()
	if ps.InitDone.Load() {
		return
	}

	ps.Mesh = env.Mesh()
	ps.Networks = env.MeshNetworks()

	ps.initDefaultExportMaps()

	if pushReq == nil || oldPushContext == nil || !oldPushContext.InitDone.Load() || pushReq.Forced {
		ps.createNewContext(env)
	} else {
		ps.updateContext(env, oldPushContext, pushReq)
	}

	ps.networkMgr = env.NetworkManager

	ps.clusterLocalHosts = env.ClusterLocal().GetClusterLocalHosts()

	ps.InitDone.Store(true)
}

func SortServicesByCreationTime(services []*Service) []*Service {
	slices.SortStableFunc(services, func(i, j *Service) int {
		if r := i.CreationTime.Compare(j.CreationTime); r != 0 {
			return r
		}
		if r := cmp.Compare(i.Attributes.Name, j.Attributes.Name); r != 0 {
			return r
		}
		return cmp.Compare(i.Attributes.Namespace, j.Attributes.Namespace)
	})
	return services
}

func resolveServiceAliases(allServices []*Service, configsUpdated sets.Set[ConfigKey]) {
	rawAlias := map[NamespacedHostname]host.Name{}
	for _, s := range allServices {
		if s.Resolution != Alias {
			continue
		}
		nh := NamespacedHostname{
			Hostname:  s.Hostname,
			Namespace: s.Attributes.Namespace,
		}
		rawAlias[nh] = host.Name(s.Attributes.K8sAttributes.ExternalName)
	}

	unnamespacedRawAlias := make(map[host.Name]host.Name, len(rawAlias))
	for k, v := range rawAlias {
		unnamespacedRawAlias[k.Hostname] = v
	}

	resolvedAliases := make(map[NamespacedHostname]host.Name, len(rawAlias))
	for alias, referencedService := range rawAlias {
		if _, f := unnamespacedRawAlias[referencedService]; !f {
			// Common case: alias pointing to a concrete service
			resolvedAliases[alias] = referencedService
			continue
		}
		seen := sets.New(alias.Hostname, referencedService)
		for {
			n, f := unnamespacedRawAlias[referencedService]
			if !f {
				resolvedAliases[alias] = referencedService
				break
			}
			if seen.InsertContains(n) {
				break
			}
			referencedService = n
		}
	}

	aliasesForService := map[host.Name][]NamespacedHostname{}
	for alias, concrete := range resolvedAliases {
		aliasesForService[concrete] = append(aliasesForService[concrete], alias)

		aliasKey := ConfigKey{
			Kind:      kind.ServiceEntry,
			Name:      alias.Hostname.String(),
			Namespace: alias.Namespace,
		}
		if configsUpdated.Contains(aliasKey) {
			for _, svc := range allServices {
				if svc.Hostname == concrete {
					configsUpdated.Insert(ConfigKey{
						Kind:      kind.ServiceEntry,
						Name:      concrete.String(),
						Namespace: svc.Attributes.Namespace,
					})
				}
			}
		}
	}
	for _, v := range aliasesForService {
		slices.SortFunc(v, func(a, b NamespacedHostname) int {
			if r := cmp.Compare(a.Namespace, b.Namespace); r != 0 {
				return r
			}
			return cmp.Compare(a.Hostname, b.Hostname)
		})
	}

	for i, s := range allServices {
		if aliases, f := aliasesForService[s.Hostname]; f {
			// This service has an alias; set it. We need to make a copy since the underlying Service is shared
			s = s.DeepCopy()
			s.Attributes.Aliases = aliases
			allServices[i] = s
		}
	}
}

func (ps *PushContext) initServiceRegistry(env *Environment, configsUpdate sets.Set[ConfigKey]) {
	allServices := SortServicesByCreationTime(env.Services())
	resolveServiceAliases(allServices, configsUpdate)

	for _, s := range allServices {
		portMap := map[string]int{}
		ports := sets.New[int]()
		for _, port := range s.Ports {
			portMap[port.Name] = port.Port
			ports.Insert(port.Port)
		}

		svcKey := s.Key()
		if _, ok := ps.ServiceIndex.instancesByPort[svcKey]; !ok {
			ps.ServiceIndex.instancesByPort[svcKey] = make(map[int][]*DubboEndpoint)
		}
		shards, ok := env.EndpointIndex.ShardsForService(string(s.Hostname), s.Attributes.Namespace)
		if ok {
			instancesByPort := shards.CopyEndpoints(portMap, ports)
			// Iterate over the instances and add them to the service index to avoid overriding the existing port instances.
			for port, instances := range instancesByPort {
				ps.ServiceIndex.instancesByPort[svcKey][port] = instances
			}
		}
		if _, f := ps.ServiceIndex.HostnameAndNamespace[s.Hostname]; !f {
			ps.ServiceIndex.HostnameAndNamespace[s.Hostname] = map[string]*Service{}
		}
		if existing := ps.ServiceIndex.HostnameAndNamespace[s.Hostname][s.Attributes.Namespace]; existing != nil &&
			!(existing.Attributes.ServiceRegistry != provider.Kubernetes && s.Attributes.ServiceRegistry == provider.Kubernetes) {
			log.Debugf("Service %s/%s from registry %s ignored by %s/%s/%s", s.Attributes.Namespace, s.Hostname, s.Attributes.ServiceRegistry,
				existing.Attributes.ServiceRegistry, existing.Attributes.Namespace, existing.Hostname)
		} else {
			ps.ServiceIndex.HostnameAndNamespace[s.Hostname][s.Attributes.Namespace] = s
		}

		ns := s.Attributes.Namespace
		if s.Attributes.ExportTo.IsEmpty() {
			if ps.exportToDefaults.service.Contains(visibility.Private) {
				ps.ServiceIndex.privateByNamespace[ns] = append(ps.ServiceIndex.privateByNamespace[ns], s)
			} else if ps.exportToDefaults.service.Contains(visibility.Public) {
				ps.ServiceIndex.public = append(ps.ServiceIndex.public, s)
			}
		} else {
			if s.Attributes.ExportTo.Contains(visibility.Public) {
				ps.ServiceIndex.public = append(ps.ServiceIndex.public, s)
				continue
			} else if s.Attributes.ExportTo.Contains(visibility.None) {
				continue
			}
			// . or other namespaces
			for exportTo := range s.Attributes.ExportTo {
				if exportTo == visibility.Private || string(exportTo) == ns {
					ps.ServiceIndex.privateByNamespace[ns] = append(ps.ServiceIndex.privateByNamespace[ns], s)
				} else {
					ps.ServiceIndex.exportedToNamespace[string(exportTo)] = append(ps.ServiceIndex.exportedToNamespace[string(exportTo)], s)
				}
			}
		}
	}

	ps.initServiceAccounts(env, allServices)
}

func (ps *PushContext) createNewContext(env *Environment) {
	ps.initServiceRegistry(env, nil)
}

func (ps *PushContext) updateContext(env *Environment, oldPushContext *PushContext, pushReq *PushRequest) {
	// Check if services have changed based on:
	// 1. ServiceEntry updates in ConfigsUpdated
	// 2. Address changes
	// 3. Actual service count changes from environment (for Kubernetes Service changes)
	servicesChanged := pushReq != nil && (HasConfigsOfKind(pushReq.ConfigsUpdated, kind.ServiceEntry) ||
		len(pushReq.AddressesUpdated) > 0)

	// Check if serviceRoutes have changed base on:
	// 1. ServiceRoute updates in ConfigsUpdated
	serviceRoutesChanged := pushReq != nil && (HasConfigsOfKind(pushReq.ConfigsUpdated, kind.ServiceRoute) ||
		len(pushReq.AddressesUpdated) > 0)

	// Check if serviceRoutes have changed base on:
	// 1. SubsetRule updates in ConfigsUpdated
	subsetRulesChanged := pushReq != nil && (HasConfigsOfKind(pushReq.ConfigsUpdated, kind.SubsetRule) ||
		len(pushReq.AddressesUpdated) > 0)

	// Also check if the actual number of services has changed
	// This handles cases where Kubernetes Services are added/removed without ServiceEntry updates
	if !servicesChanged && oldPushContext != nil {
		currentServices := env.Services()
		// Count services in old ServiceIndex
		oldServiceCount := 0
		for _, namespaces := range oldPushContext.ServiceIndex.HostnameAndNamespace {
			oldServiceCount += len(namespaces)
		}
		// If service count differs, services have changed
		if len(currentServices) != oldServiceCount {
			servicesChanged = true
		}
	}

	if servicesChanged {
		// Services have changed. initialize service registry
		ps.initServiceRegistry(env, pushReq.ConfigsUpdated)
	} else {
		// make sure we copy over things that would be generated in initServiceRegistry
		ps.ServiceIndex = oldPushContext.ServiceIndex
		ps.serviceAccounts = oldPushContext.serviceAccounts
	}

	if serviceRoutesChanged {
		ps.initServiceRoutes(env)
	} else {
		ps.serviceRouteIndex = oldPushContext.serviceRouteIndex
	}

	if subsetRulesChanged {
		ps.initSubsetRules(env)
	} else {
		ps.subsetRuleIndex = oldPushContext.subsetRuleIndex
	}

}

func (ps *PushContext) ServiceForHostname(proxy *Proxy, hostname host.Name) *Service {
	for _, service := range ps.ServiceIndex.HostnameAndNamespace[hostname] {
		return service
	}

	// No service found
	return nil
}

func (ps *PushContext) UpdateMetrics() {
	ps.proxyStatusMutex.RLock()
	defer ps.proxyStatusMutex.RUnlock()
}

func (ps *PushContext) OnConfigChange() {
	LastPushMutex.Lock()
	LastPushStatus = ps
	LastPushMutex.Unlock()
	ps.UpdateMetrics()
}

func (ps *PushContext) servicesExportedToNamespace(ns string) []*Service {
	var out []*Service

	// First add private services and explicitly exportedTo services
	if ns == NamespaceAll {
		out = make([]*Service, 0, len(ps.ServiceIndex.privateByNamespace)+len(ps.ServiceIndex.public))
		for _, privateServices := range ps.ServiceIndex.privateByNamespace {
			out = append(out, privateServices...)
		}
	} else {
		out = make([]*Service, 0, len(ps.ServiceIndex.privateByNamespace[ns])+
			len(ps.ServiceIndex.exportedToNamespace[ns])+len(ps.ServiceIndex.public))
		out = append(out, ps.ServiceIndex.privateByNamespace[ns]...)
		out = append(out, ps.ServiceIndex.exportedToNamespace[ns]...)
	}

	// Second add public services
	out = append(out, ps.ServiceIndex.public...)

	return out
}

func (ps *PushContext) GetAllServices() []*Service {
	return ps.servicesExportedToNamespace(NamespaceAll)
}

func (ps *PushContext) initServiceRoutes(env *Environment) {
	ps.serviceRouteIndex.referencedDestinations = map[string]sets.String{}
	serviceroutes := env.List(gvk.ServiceRoute, NamespaceAll)
	sroutes := make([]config.Config, len(serviceroutes))

	for i, r := range serviceroutes {
		sroutes[i] = resolveServiceRouteShortnames(r)
	}
	sroutes, ps.serviceRouteIndex.delegates = mergeServiceRoutesIfNeeded(sroutes, ps.exportToDefaults.serviceRoute)

}

// sortConfigBySelectorAndCreationTime sorts the list of config objects based on priority and creation time.
func sortConfigBySelectorAndCreationTime(configs []config.Config) []config.Config {
	sort.Slice(configs, func(i, j int) bool {
		// check if one of the configs has priority
		idr := configs[i].Spec.(*networking.DestinationRule)
		jdr := configs[j].Spec.(*networking.DestinationRule)
		if idr.GetWorkloadSelector() != nil && jdr.GetWorkloadSelector() == nil {
			return true
		}
		if idr.GetWorkloadSelector() == nil && jdr.GetWorkloadSelector() != nil {
			return false
		}

		// If priority is the same or neither has priority, fallback to creation time ordering
		if r := configs[i].CreationTimestamp.Compare(configs[j].CreationTimestamp); r != 0 {
			return r == -1 // -1 means i is less than j, so return true.
		}
		if r := cmp.Compare(configs[i].Name, configs[j].Name); r != 0 {
			return r == -1
		}
		return cmp.Compare(configs[i].Namespace, configs[j].Namespace) == -1
	})
	return configs
}

func (ps *PushContext) setSubsetRules(configs []config.Config) {
	sortConfigBySelectorAndCreationTime(configs)

	namespaceLocalSubRules := make(map[string]*consolidatedSubRules)
	exportedDestRulesByNamespace := make(map[string]*consolidatedSubRules)
	rootNamespaceLocalDestRules := newConsolidatedDestRules()

	for i := range configs {
		rule := configs[i].Spec.(*networking.DestinationRule)

		rule.Host = string(ResolveShortnameToFQDN(rule.Host, configs[i].Meta))
		var exportToSet sets.Set[visibility.Instance]

		// destination rules with workloadSelector should not be exported to other namespaces
		if rule.GetWorkloadSelector() == nil {
			exportToSet = sets.NewWithLength[visibility.Instance](len(rule.ExportTo))
			for _, e := range rule.ExportTo {
				exportToSet.Insert(visibility.Instance(e))
			}
		} else {
			exportToSet = sets.New[visibility.Instance](visibility.Private)
		}

		// add only if the dest rule is exported with . or * or explicit exportTo containing this namespace
		// The global exportTo doesn't matter here (its either . or * - both of which are applicable here)
		if exportToSet.IsEmpty() || exportToSet.Contains(visibility.Public) || exportToSet.Contains(visibility.Private) ||
			exportToSet.Contains(visibility.Instance(configs[i].Namespace)) {
			// Store in an index for the config's namespace
			// a proxy from this namespace will first look here for the destination rule for a given service
			// This pool consists of both public/private destination rules.
			if _, exist := namespaceLocalSubRules[configs[i].Namespace]; !exist {
				namespaceLocalSubRules[configs[i].Namespace] = newConsolidatedDestRules()
			}
			// Merge this destination rule with any public/private dest rules for same host in the same namespace
			// If there are no duplicates, the dest rule will be added to the list
			ps.mergeSubsetRule(namespaceLocalSubRules[configs[i].Namespace], configs[i], exportToSet)
		}

		isPrivateOnly := false
		// No exportTo in destinationRule. Use the global default
		// We only honor . and *
		if exportToSet.IsEmpty() && ps.exportToDefaults.subsetRule.Contains(visibility.Private) {
			isPrivateOnly = true
		} else if exportToSet.Len() == 1 && (exportToSet.Contains(visibility.Private) || exportToSet.Contains(visibility.Instance(configs[i].Namespace))) {
			isPrivateOnly = true
		}

		if !isPrivateOnly {
			if _, exist := exportedDestRulesByNamespace[configs[i].Namespace]; !exist {
				exportedDestRulesByNamespace[configs[i].Namespace] = newConsolidatedDestRules()
			}
			ps.mergeSubsetRule(exportedDestRulesByNamespace[configs[i].Namespace], configs[i], exportToSet)
		} else if configs[i].Namespace == ps.Mesh.RootNamespace {
			ps.mergeSubsetRule(rootNamespaceLocalDestRules, configs[i], exportToSet)
		}
	}

	ps.subsetRuleIndex.namespaceLocal = namespaceLocalSubRules
	ps.subsetRuleIndex.exportedByNamespace = exportedDestRulesByNamespace
	ps.subsetRuleIndex.rootNamespaceLocal = rootNamespaceLocalDestRules
}

func (ps *PushContext) initSubsetRules(env *Environment) {
	configs := env.List(gvk.SubsetRule, NamespaceAll)

	// values returned from ConfigStore.List are immutable.
	// Therefore, we make a copy
	subRules := make([]config.Config, len(configs))
	for i := range subRules {
		subRules[i] = configs[i]
	}

	ps.setSubsetRules(subRules)
}

func (ps *PushContext) initServiceAccounts(env *Environment, services []*Service) {
	for _, svc := range services {
		var accounts sets.String
		// First get endpoint level service accounts
		shard, f := env.EndpointIndex.ShardsForService(string(svc.Hostname), svc.Attributes.Namespace)
		if f {
			shard.RLock()
			accounts = shard.ServiceAccounts.Copy()
			shard.RUnlock()
		}
		if len(svc.ServiceAccounts) > 0 {
			if accounts == nil {
				accounts = sets.New(svc.ServiceAccounts...)
			} else {
				accounts = accounts.InsertAll(svc.ServiceAccounts...)
			}
		}
		sa := sets.SortedList(spiffe.ExpandWithTrustDomains(accounts, ps.Mesh.TrustDomainAliases))
		key := serviceAccountKey{
			hostname:  svc.Hostname,
			namespace: svc.Attributes.Namespace,
		}
		ps.serviceAccounts[key] = sa
	}
}

func (ps *PushContext) DelegateServiceRoutes(vses []config.Config) []ConfigHash {
	var out []ConfigHash
	for _, vs := range vses {
		for _, delegate := range ps.serviceRouteIndex.delegates[ConfigKey{Kind: kind.ServiceRoute, Namespace: vs.Namespace, Name: vs.Name}] {
			out = append(out, delegate.HashCode())
		}
	}
	return out
}

func ConfigNamesOfKind(configs sets.Set[ConfigKey], k kind.Kind) sets.String {
	ret := sets.New[string]()

	for conf := range configs {
		if conf.Kind == k {
			ret.Insert(conf.Name)
		}
	}

	return ret
}
