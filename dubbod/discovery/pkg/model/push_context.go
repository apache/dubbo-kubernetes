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

package model

import (
	"cmp"
	"sort"
	"strings"
	"sync"
	"time"

	networking "github.com/apache/dubbo-kubernetes/api/networking/v1alpha3"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	sigsk8siogatewayapiapisv1 "sigs.k8s.io/gateway-api/apis/v1"

	meshv1alpha1 "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/serviceregistry/provider"
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
	ConfigUpdate           TriggerReason = "config"
	DependentResource      TriggerReason = "depdendentresource"
	// NetworksTrigger describes a push triggered for Networks change
	NetworksTrigger TriggerReason = "networks"
)

var (
	LastPushStatus *PushContext
	LastPushMutex  sync.Mutex
)

type PushContext struct {
	Mesh            *meshv1alpha1.MeshGlobalConfig `json:"-"`
	initializeMutex sync.Mutex
	InitDone        atomic.Bool
	// GatewayAPIController holds a reference to the Gateway API controller.
	// When enabled, this controller is responsible for translating Kubernetes
	// Gateway API resources into internal Dubbo resources during push.
	GatewayAPIController   GatewayController
	networkMgr             *NetworkManager
	clusterLocalHosts      ClusterLocalHosts
	exportToDefaults       exportToDefaults
	ServiceIndex           serviceIndex
	virtualServiceIndex    virtualServiceIndex
	httpRouteIndex         httpRouteIndex
	destinationRuleIndex   destinationRuleIndex
	serviceAccounts        map[serviceAccountKey][]string
	AuthenticationPolicies *AuthenticationPolicies
	PushVersion            string
	ProxyStatus            map[string]map[string]ProxyPushStatus
	proxyStatusMutex       sync.RWMutex
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
	service         sets.Set[visibility.Instance]
	virtualService  sets.Set[visibility.Instance]
	destinationRule sets.Set[visibility.Instance]
}

type virtualServiceIndex struct {
	// Map of VS hostname -> referenced hostnames
	referencedDestinations map[string]sets.String

	// hostToRoutes keeps the resolved VirtualServices keyed by host
	hostToRoutes map[host.Name][]config.Config
}

type httpRouteIndex struct {
	// hostToRoutes keeps the Gateway API HTTPRoutes keyed by hostname
	hostToRoutes map[host.Name][]config.Config
}

type destinationRuleIndex struct {
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
		ServiceIndex:         newServiceIndex(),
		virtualServiceIndex:  newVirtualServiceIndex(),
		destinationRuleIndex: newDestinationRuleIndex(),
		serviceAccounts:      map[serviceAccountKey][]string{},
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

func newVirtualServiceIndex() virtualServiceIndex {
	out := virtualServiceIndex{
		referencedDestinations: map[string]sets.String{},
		hostToRoutes:           map[host.Name][]config.Config{},
	}
	return out
}

func newDestinationRuleIndex() destinationRuleIndex {
	return destinationRuleIndex{
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
	ps.exportToDefaults.destinationRule = sets.New[visibility.Instance]()
	if ps.Mesh.DefaultDestinationRuleExportTo != nil {
		for _, e := range ps.Mesh.DefaultDestinationRuleExportTo {
			ps.exportToDefaults.destinationRule.Insert(visibility.Instance(e))
		}
	} else {
		// default to *
		ps.exportToDefaults.destinationRule.Insert(visibility.Public)
	}

	ps.exportToDefaults.service = sets.New[visibility.Instance]()
	if ps.Mesh.DefaultServiceExportTo != nil {
		for _, e := range ps.Mesh.DefaultServiceExportTo {
			ps.exportToDefaults.service.Insert(visibility.Instance(e))
		}
	} else {
		ps.exportToDefaults.service.Insert(visibility.Public)
	}

	ps.exportToDefaults.virtualService = sets.New[visibility.Instance]()
	if ps.Mesh.DefaultVirtualServiceExportTo != nil {
		for _, e := range ps.Mesh.DefaultVirtualServiceExportTo {
			ps.exportToDefaults.virtualService.Insert(visibility.Instance(e))
		}
	} else {
		ps.exportToDefaults.virtualService.Insert(visibility.Public)
	}
}

func (ps *PushContext) InitContext(env *Environment, oldPushContext *PushContext, pushReq *PushRequest) {
	ps.initializeMutex.Lock()
	defer ps.initializeMutex.Unlock()
	if ps.InitDone.Load() {
		return
	}

	ps.Mesh = env.Mesh()

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
	log.Debug("creating new PushContext (full initialization)")
	ps.initServiceRegistry(env, nil)
	// Initialize Kubernetes Gateway API resources if the controller is enabled.
	// This mirrors Dubbo's behavior, where Gateway API is translated into
	// internal Gateway and VirtualService resources during push context creation.
	ps.initKubernetesGateways(env)
	ps.initVirtualServices(env)
	ps.initHTTPRoutes(env)
	ps.initDestinationRules(env)
	ps.initAuthenticationPolicies(env)
}

func (ps *PushContext) updateContext(env *Environment, oldPushContext *PushContext, pushReq *PushRequest) {
	// Check if services have changed based on:
	// 1. ServiceEntry updates in ConfigsUpdated
	// 2. Address changes
	// 3. Actual service count changes from environment (for Kubernetes Service changes)
	servicesChanged := pushReq != nil && (HasConfigsOfKind(pushReq.ConfigsUpdated, kind.ServiceEntry) ||
		len(pushReq.AddressesUpdated) > 0)

	// Check if virtualServices have changed base on:
	// 1. VirtualService updates in ConfigsUpdated
	// 2. Full push (Full: true) - always re-initialize on full push
	virtualServicesChanged := pushReq != nil && (pushReq.Full || HasConfigsOfKind(pushReq.ConfigsUpdated, kind.VirtualService) ||
		len(pushReq.AddressesUpdated) > 0)

	if pushReq != nil {
		virtualServiceCount := 0
		for cfg := range pushReq.ConfigsUpdated {
			if cfg.Kind == kind.VirtualService {
				virtualServiceCount++
			}
		}
		if virtualServiceCount > 0 {
			log.Debugf("detected %d VirtualService config changes", virtualServiceCount)
		}
	}

	// Check if destinationrules have changed base on:
	// 1. DestinationRule updates in ConfigsUpdated
	// 2. Full push (Full: true) - always re-initialize on full push
	destinationRuleChanged := pushReq != nil && (pushReq.Full || HasConfigsOfKind(pushReq.ConfigsUpdated, kind.DestinationRule) ||
		len(pushReq.AddressesUpdated) > 0)

	if pushReq != nil {
		destinationRuleCount := 0
		for cfg := range pushReq.ConfigsUpdated {
			if cfg.Kind == kind.DestinationRule {
				destinationRuleCount++
			}
		}
		if destinationRuleCount > 0 {
			log.Debugf("detected %d DestinationRule config changes", destinationRuleCount)
		}
		if pushReq.Full {
			log.Debugf("Full push requested, will re-initialize DestinationRule and VirtualService indexes")
		}
		log.Debugf("destinationRuleChanged=%v, virtualServicesChanged=%v, pushReq.ConfigsUpdated size=%d, Full=%v",
			destinationRuleChanged, virtualServicesChanged, len(pushReq.ConfigsUpdated), pushReq != nil && pushReq.Full)
	}

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

	// Initialize or reuse Gateway API controller state.
	// Gateway status and derived configs depend on services, so recompute
	// if services have changed, otherwise carry over the previous controller.
	if servicesChanged {
		ps.initKubernetesGateways(env)
	} else {
		ps.GatewayAPIController = oldPushContext.GatewayAPIController
	}

	httpRoutesChanged := pushReq != nil && HasConfigsOfKind(pushReq.ConfigsUpdated, kind.HTTPRoute)

	if virtualServicesChanged {
		log.Debugf("VirtualServices changed, re-initializing VirtualService index")
		ps.initVirtualServices(env)
	} else {
		log.Debugf("VirtualServices unchanged, reusing old VirtualService index")
		ps.virtualServiceIndex = oldPushContext.virtualServiceIndex
	}

	if httpRoutesChanged {
		log.Debugf("HTTPRoutes changed, re-initializing HTTPRoute index")
		ps.initHTTPRoutes(env)
	} else {
		log.Debugf("HTTPRoutes unchanged, reusing old HTTPRoute index")
		ps.httpRouteIndex = oldPushContext.httpRouteIndex
	}

	if destinationRuleChanged {
		log.Debugf("DestinationRules changed, re-initializing DestinationRule index")
		ps.initDestinationRules(env)
	} else {
		log.Debugf("DestinationRules unchanged, reusing old DestinationRule index")
		ps.destinationRuleIndex = oldPushContext.destinationRuleIndex
	}

	authnPoliciesChanged := pushReq != nil && (pushReq.Full || HasConfigsOfKind(pushReq.ConfigsUpdated, kind.PeerAuthentication))
	if authnPoliciesChanged || oldPushContext == nil || oldPushContext.AuthenticationPolicies == nil {
		log.Debugf("PeerAuthentication changed (full=%v, configsUpdatedContainingPeerAuth=%v), rebuilding authentication policies",
			pushReq != nil && pushReq.Full, func() bool {
				if pushReq == nil {
					return false
				}
				return HasConfigsOfKind(pushReq.ConfigsUpdated, kind.PeerAuthentication)
			}())
		ps.initAuthenticationPolicies(env)
	} else {
		ps.AuthenticationPolicies = oldPushContext.AuthenticationPolicies
	}
}

func (ps *PushContext) initAuthenticationPolicies(env *Environment) {
	if env == nil {
		ps.AuthenticationPolicies = nil
		return
	}
	ps.AuthenticationPolicies = initAuthenticationPolicies(env)
}

func (ps *PushContext) InboundMTLSModeForProxy(proxy *Proxy, port uint32) MutualTLSMode {
	if ps == nil || proxy == nil || ps.AuthenticationPolicies == nil {
		return MTLSUnknown
	}
	var namespace string
	if proxy.Metadata != nil {
		namespace = proxy.Metadata.Namespace
	}
	if namespace == "" {
		namespace = proxy.ConfigNamespace
	}
	return ps.AuthenticationPolicies.EffectiveMutualTLSMode(namespace, nil, port)
}

// initKubernetesGateways initializes Kubernetes Gateway API objects by delegating
// to the GatewayAPIController, if it is present in the Environment.
// This closely follows Dubbo's initKubernetesGateways behavior.
func (ps *PushContext) initKubernetesGateways(env *Environment) {
	if env == nil || env.GatewayAPIController == nil {
		return
	}
	ps.GatewayAPIController = env.GatewayAPIController
	env.GatewayAPIController.Reconcile(ps)
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

func (ps *PushContext) initVirtualServices(env *Environment) {
	log.Debugf("starting VirtualService initialization")
	ps.virtualServiceIndex.referencedDestinations = map[string]sets.String{}
	virtualservices := env.List(gvk.VirtualService, NamespaceAll)
	log.Debugf("found %d VirtualService configs", len(virtualservices))
	vsroutes := make([]config.Config, len(virtualservices))

	for i, r := range virtualservices {
		vsroutes[i] = resolveVirtualServiceShortnames(r)
		if vs, ok := r.Spec.(*networking.VirtualService); ok {
			log.Debugf("VirtualService %s/%s with hosts %v and %d HTTP routes",
				r.Namespace, r.Name, vs.Hosts, len(vs.Http))
		}
	}

	hostToRoutes := make(map[host.Name][]config.Config)
	for i := range vsroutes {
		vs := vsroutes[i].Spec.(*networking.VirtualService)
		for idx, h := range vs.Hosts {
			resolvedHost := string(ResolveShortnameToFQDN(h, vsroutes[i].Meta))
			vs.Hosts[idx] = resolvedHost
			hostName := host.Name(resolvedHost)
			hostToRoutes[hostName] = append(hostToRoutes[hostName], vsroutes[i])
			log.Debugf("indexed VirtualService %s/%s for hostname %s", vsroutes[i].Namespace, vsroutes[i].Name, hostName)
		}
	}
	ps.virtualServiceIndex.hostToRoutes = hostToRoutes
	log.Debugf("indexed VirtualServices for %d hostnames", len(hostToRoutes))
}

func (ps *PushContext) initHTTPRoutes(env *Environment) {
	log.Debugf("starting HTTPRoute initialization")
	httproutes := env.List(gvk.HTTPRoute, NamespaceAll)
	log.Debugf("found %d HTTPRoute configs", len(httproutes))

	hostToRoutes := make(map[host.Name][]config.Config)
	for _, hr := range httproutes {
		hrSpec, ok := hr.Spec.(*sigsk8siogatewayapiapisv1.HTTPRouteSpec)
		if !ok {
			log.Debugf("HTTPRoute %s/%s spec is not HTTPRouteSpec", hr.Namespace, hr.Name)
			continue
		}

		// Process hostnames from HTTPRoute
		if len(hrSpec.Hostnames) == 0 {
			// If no hostnames specified, match all
			hostToRoutes["*"] = append(hostToRoutes["*"], hr)
			log.Debugf("indexed HTTPRoute %s/%s for wildcard hostname (no hostnames specified)", hr.Namespace, hr.Name)
		} else {
			for _, hostname := range hrSpec.Hostnames {
				if hostname == "" {
					// Empty hostname means match all
					hostToRoutes["*"] = append(hostToRoutes["*"], hr)
					log.Debugf("indexed HTTPRoute %s/%s for wildcard hostname", hr.Namespace, hr.Name)
				} else {
					hostStr := string(hostname)
					// Resolve shortname to FQDN
					resolvedHost := string(ResolveShortnameToFQDN(hostStr, hr.Meta))
					hostName := host.Name(resolvedHost)
					hostToRoutes[hostName] = append(hostToRoutes[hostName], hr)
					log.Debugf("indexed HTTPRoute %s/%s for hostname %s", hr.Namespace, hr.Name, hostName)
				}
			}
		}
	}
	ps.httpRouteIndex.hostToRoutes = hostToRoutes
	log.Debugf("indexed HTTPRoutes for %d hostnames", len(hostToRoutes))
	if len(hostToRoutes) > 0 {
		for hostname, routes := range hostToRoutes {
			log.Infof("hostname %s has %d HTTPRoute(s)", hostname, len(routes))
		}
	}
}

// HTTPRouteForHost returns HTTPRoutes that match the given hostname.
func (ps *PushContext) HTTPRouteForHost(hostname host.Name) []config.Config {
	var routes []config.Config
	hostStr := string(hostname)

	// Special case: if hostname is "*", return ALL HTTPRoutes
	// This is needed for Gateway Pod inbound listeners that need to route traffic based on HTTPRoute hostnames
	if hostname == "*" {
		for _, routeList := range ps.httpRouteIndex.hostToRoutes {
			routes = append(routes, routeList...)
		}
		if len(routes) == 0 {
			log.Debugf("no HTTPRoute found for wildcard hostname")
			return nil
		}
		log.Infof("found %d HTTPRoute(s) for wildcard hostname", len(routes))
		return routes
	}

	// First check exact match
	if exactRoutes, ok := ps.httpRouteIndex.hostToRoutes[hostname]; ok {
		routes = append(routes, exactRoutes...)
	}

	// Check wildcard patterns (e.g., *.example.com)
	for patternHost, patternRoutes := range ps.httpRouteIndex.hostToRoutes {
		if patternHost == "*" {
			continue // Skip global wildcard, handle separately
		}
		patternStr := string(patternHost)
		if strings.HasPrefix(patternStr, "*.") {
			// Wildcard pattern like *.example.com
			suffix := patternStr[2:] // Remove "*."
			if strings.HasSuffix(hostStr, suffix) {
				routes = append(routes, patternRoutes...)
			}
		}
	}

	// Then check global wildcard
	if wildcardRoutes, ok := ps.httpRouteIndex.hostToRoutes["*"]; ok {
		routes = append(routes, wildcardRoutes...)
	}

	if len(routes) == 0 {
		log.Debugf("no HTTPRoute found for hostname %s", hostname)
		return nil
	}

	log.Infof("found %d HTTPRoute(s) for hostname %s", len(routes), hostname)
	return routes
}

// sortConfigBySelectorAndCreationTime sorts the list of config objects based on creation time.
func sortConfigBySelectorAndCreationTime(configs []config.Config) []config.Config {
	sort.Slice(configs, func(i, j int) bool {
		// Sort by creation time ordering
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

func (ps *PushContext) setDestinationRules(configs []config.Config) {
	sortConfigBySelectorAndCreationTime(configs)

	namespaceLocalSubRules := make(map[string]*consolidatedSubRules)
	exportedDestRulesByNamespace := make(map[string]*consolidatedSubRules)
	rootNamespaceLocalDestRules := newConsolidatedDestRules()

	for i := range configs {
		rule := configs[i].Spec.(*networking.DestinationRule)

		rule.Host = string(ResolveShortnameToFQDN(rule.Host, configs[i].Meta))
		var exportToSet sets.Set[visibility.Instance]

		exportToSet = sets.NewWithLength[visibility.Instance](len(rule.ExportTo))
		for _, e := range rule.ExportTo {
			exportToSet.Insert(visibility.Instance(e))
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
			ps.mergeDestinationRule(namespaceLocalSubRules[configs[i].Namespace], configs[i], exportToSet)
		}

		isPrivateOnly := false
		// No exportTo in destinationRule. Use the global default
		// We only honor . and *
		if exportToSet.IsEmpty() && ps.exportToDefaults.destinationRule.Contains(visibility.Private) {
			isPrivateOnly = true
		} else if exportToSet.Len() == 1 && (exportToSet.Contains(visibility.Private) || exportToSet.Contains(visibility.Instance(configs[i].Namespace))) {
			isPrivateOnly = true
		}

		if !isPrivateOnly {
			if _, exist := exportedDestRulesByNamespace[configs[i].Namespace]; !exist {
				exportedDestRulesByNamespace[configs[i].Namespace] = newConsolidatedDestRules()
			}
			ps.mergeDestinationRule(exportedDestRulesByNamespace[configs[i].Namespace], configs[i], exportToSet)
		} else if configs[i].Namespace == ps.Mesh.RootNamespace {
			ps.mergeDestinationRule(rootNamespaceLocalDestRules, configs[i], exportToSet)
		}
	}

	ps.destinationRuleIndex.namespaceLocal = namespaceLocalSubRules
	ps.destinationRuleIndex.exportedByNamespace = exportedDestRulesByNamespace
	ps.destinationRuleIndex.rootNamespaceLocal = rootNamespaceLocalDestRules

	// Log indexing results
	log.Debugf("indexed %d namespaces with local rules", len(namespaceLocalSubRules))
	for ns, rules := range namespaceLocalSubRules {
		totalRules := 0
		for hostname, ruleList := range rules.specificSubRules {
			totalRules += len(ruleList)
			// Log TLS configuration for each merged DestinationRule
			for _, rule := range ruleList {
				if dr, ok := rule.rule.Spec.(*networking.DestinationRule); ok {
					hasTLS := dr.TrafficPolicy != nil && dr.TrafficPolicy.Tls != nil
					tlsMode := "none"
					if hasTLS {
						tlsMode = dr.TrafficPolicy.Tls.Mode.String()
					}
					log.Debugf("namespace %s, hostname %s: DestinationRule has %d subsets, TLS mode: %s",
						ns, hostname, len(dr.Subsets), tlsMode)
				}
			}
		}
		log.Debugf("namespace %s has %d DestinationRules with %d specific hostnames", ns, totalRules, len(rules.specificSubRules))
	}
	log.Debugf("indexed %d namespaces with exported rules", len(exportedDestRulesByNamespace))
	if rootNamespaceLocalDestRules != nil {
		totalRootRules := 0
		for _, ruleList := range rootNamespaceLocalDestRules.specificSubRules {
			totalRootRules += len(ruleList)
		}
		log.Debugf("root namespace has %d DestinationRules with %d specific hostnames", totalRootRules, len(rootNamespaceLocalDestRules.specificSubRules))
	}
}

func (ps *PushContext) initDestinationRules(env *Environment) {
	configs := env.List(gvk.DestinationRule, NamespaceAll)
	log.Debugf("initDestinationRules: found %d DestinationRule configs", len(configs))

	// values returned from ConfigStore.List are immutable.
	// Therefore, we make a copy
	subRules := make([]config.Config, len(configs))
	for i := range subRules {
		subRules[i] = configs[i]
		if dr, ok := configs[i].Spec.(*networking.DestinationRule); ok {
			tlsMode := "none"
			if dr.TrafficPolicy != nil && dr.TrafficPolicy.Tls != nil {
				tlsMode = dr.TrafficPolicy.Tls.Mode.String()
			}
			log.Debugf("initDestinationRules: DestinationRule %s/%s for host %s with %d subsets, TLS mode: %s",
				configs[i].Namespace, configs[i].Name, dr.Host, len(dr.Subsets), tlsMode)
		}
	}

	ps.setDestinationRules(subRules)
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

// VirtualServiceForHost returns the first VirtualService that matches the given host.
func (ps *PushContext) VirtualServiceForHost(hostname host.Name) *networking.VirtualService {
	routes := ps.virtualServiceIndex.hostToRoutes[hostname]
	if len(routes) == 0 {
		log.Debugf("no VirtualService found for hostname %s", hostname)
		return nil
	}
	if vs, ok := routes[0].Spec.(*networking.VirtualService); ok {
		log.Infof("found VirtualService %s/%s for hostname %s with %d HTTP routes",
			routes[0].Namespace, routes[0].Name, hostname, len(vs.Http))
		return vs
	}
	log.Warnf("VirtualService %s/%s for hostname %s is not a VirtualService",
		routes[0].Namespace, routes[0].Name, hostname)
	return nil
}

// DestinationRuleForService returns the first DestinationRule applicable to the service hostname/namespace.
func (ps *PushContext) DestinationRuleForService(namespace string, hostname host.Name) *networking.DestinationRule {
	log.Debugf("looking for DestinationRule for %s/%s", namespace, hostname)

	// Check namespace-local rules first
	if nsRules := ps.destinationRuleIndex.namespaceLocal[namespace]; nsRules != nil {
		log.Debugf("checking namespace-local rules for %s (found %d specific rules)", namespace, len(nsRules.specificSubRules))
		if dr := firstDestinationRule(nsRules, hostname); dr != nil {
			hasTLS := dr.TrafficPolicy != nil && dr.TrafficPolicy.Tls != nil
			tlsMode := "none"
			if hasTLS {
				tlsMode = dr.TrafficPolicy.Tls.Mode.String()
			}
			log.Infof("found DestinationRule in namespace-local index for %s/%s with %d subsets (has TrafficPolicy: %v, has TLS: %v, TLS mode: %s)",
				namespace, hostname, len(dr.Subsets), dr.TrafficPolicy != nil, hasTLS, tlsMode)
			return dr
		}
	} else {
		log.Debugf("no namespace-local rules for namespace %s", namespace)
	}

	// Check exported rules
	log.Debugf("checking exported rules (found %d exported namespaces)", len(ps.destinationRuleIndex.exportedByNamespace))
	for ns, exported := range ps.destinationRuleIndex.exportedByNamespace {
		if dr := firstDestinationRule(exported, hostname); dr != nil {
			hasTLS := dr.TrafficPolicy != nil && dr.TrafficPolicy.Tls != nil
			tlsMode := "none"
			if hasTLS {
				tlsMode = dr.TrafficPolicy.Tls.Mode.String()
			}
			log.Infof("found DestinationRule in exported rules from namespace %s for %s/%s with %d subsets (has TrafficPolicy: %v, has TLS: %v, TLS mode: %s)",
				ns, namespace, hostname, len(dr.Subsets), dr.TrafficPolicy != nil, hasTLS, tlsMode)
			return dr
		}
	}

	// Finally, check root namespace scoped rules
	if rootRules := ps.destinationRuleIndex.rootNamespaceLocal; rootRules != nil {
		log.Debugf("checking root namespace rules (found %d specific rules)", len(rootRules.specificSubRules))
		if dr := firstDestinationRule(rootRules, hostname); dr != nil {
			hasTLS := dr.TrafficPolicy != nil && dr.TrafficPolicy.Tls != nil
			tlsMode := "none"
			if hasTLS {
				tlsMode = dr.TrafficPolicy.Tls.Mode.String()
			}
			log.Infof("found DestinationRule in root namespace for %s/%s with %d subsets (has TrafficPolicy: %v, has TLS: %v, TLS mode: %s)",
				namespace, hostname, len(dr.Subsets), dr.TrafficPolicy != nil, hasTLS, tlsMode)
			return dr
		}
	}

	log.Warnf("no DestinationRule found for %s/%s", namespace, hostname)
	return nil
}

// SubsetLabelsForHost returns the label selector for a subset defined in DestinationRule.
func (ps *PushContext) SubsetLabelsForHost(namespace string, hostname host.Name, subset string) labels.Instance {
	if subset == "" {
		return nil
	}
	rule := ps.DestinationRuleForService(namespace, hostname)
	if rule == nil {
		return nil
	}
	for _, ss := range rule.Subsets {
		if ss.Name == subset {
			return labels.Instance(ss.Labels)
		}
	}
	return nil
}

func firstDestinationRule(csr *consolidatedSubRules, hostname host.Name) *networking.DestinationRule {
	if csr == nil {
		log.Debugf("consolidatedSubRules is nil for hostname %s", hostname)
		return nil
	}
	if rules := csr.specificSubRules[hostname]; len(rules) > 0 {
		log.Infof("found %d rules for hostname %s", len(rules), hostname)
		// The first rule should contain the merged result if merge was successful.
		// However, if merge failed (e.g., EnableEnhancedDestinationRuleMerge is disabled),
		// we need to check all rules and prefer the one with TLS configuration.
		// we return the one that has TLS if available, or the first one otherwise.
		var bestRule *networking.DestinationRule
		var bestRuleHasTLS bool
		for i, rule := range rules {
			if dr, ok := rule.rule.Spec.(*networking.DestinationRule); ok {
				hasTLS := dr.TrafficPolicy != nil && dr.TrafficPolicy.Tls != nil
				if hasTLS {
					tlsModeStr := dr.TrafficPolicy.Tls.Mode.String()
					hasTLS = (tlsModeStr == "DUBBO_MUTUAL")
				}
				if i == 0 {
					// Always use first rule as fallback
					bestRule = dr
					bestRuleHasTLS = hasTLS
				} else if hasTLS && !bestRuleHasTLS {
					// Prefer rule with TLS over rule without TLS
					log.Infof("found rule %d with TLS for hostname %s, preferring it over rule 0", i, hostname)
					bestRule = dr
					bestRuleHasTLS = hasTLS
				}
			}
		}
		if bestRule != nil {
			tlsMode := "none"
			if bestRuleHasTLS {
				tlsMode = bestRule.TrafficPolicy.Tls.Mode.String()
			}
			log.Infof("returning DestinationRule for hostname %s (has TrafficPolicy: %v, has TLS: %v, TLS mode: %s, has %d subsets)",
				hostname, bestRule.TrafficPolicy != nil, bestRuleHasTLS, tlsMode, len(bestRule.Subsets))
			return bestRule
		} else {
			log.Warnf("failed to cast any rule to DestinationRule for hostname %s", hostname)
		}
	} else {
		log.Debugf("no specific rules found for hostname %s (available hostnames: %v)", hostname, func() []string {
			hosts := make([]string, 0, len(csr.specificSubRules))
			for h := range csr.specificSubRules {
				hosts = append(hosts, string(h))
			}
			return hosts
		}())
	}
	// TODO: support wildcard hosts
	return nil
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
