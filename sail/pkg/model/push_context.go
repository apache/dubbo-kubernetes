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
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/config/visibility"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"go.uber.org/atomic"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sync"
	"time"
)

type TriggerReason string

const (
	UnknownTrigger TriggerReason = "unknown"
)

type PushContext struct {
	Mesh                 *meshconfig.MeshConfig `json:"-"`
	initializeMutex      sync.Mutex
	InitDone             atomic.Bool
	Networks             *meshconfig.MeshNetworks
	networkMgr           *NetworkManager
	clusterLocalHosts    ClusterLocalHosts
	exportToDefaults     exportToDefaults
	AuthnPolicies        *AuthenticationPolicies `json:"-"`
	AuthzPolicies        *AuthorizationPolicies  `json:"-"`
	virtualServiceIndex  virtualServiceIndex
	destinationRuleIndex destinationRuleIndex
	ServiceIndex         serviceIndex
	serviceAccounts      map[serviceAccountKey][]string
}

type serviceAccountKey struct {
	hostname  host.Name
	namespace string
}

type virtualServiceIndex struct {
}

type destinationRuleIndex struct {
}

type consolidatedDestRules struct {
	specificDestRules map[host.Name][]*ConsolidatedDestRule
	wildcardDestRules map[host.Name][]*ConsolidatedDestRule
}

type serviceIndex struct {
	privateByNamespace   map[string][]*Service
	public               []*Service
	exportedToNamespace  map[string][]*Service
	HostnameAndNamespace map[host.Name]map[string]*Service `json:"-"`
}

type ConsolidatedDestRule struct {
	exportTo sets.Set[visibility.Instance]
	rule     *config.Config
	from     []types.NamespacedName
}

type ReasonStats map[TriggerReason]int

type PushRequest struct {
	Reason           ReasonStats
	ConfigsUpdated   sets.Set[ConfigKey]
	Forced           bool
	Full             bool
	Push             *PushContext
	LastPushContext  *PushContext
	AddressesUpdated sets.Set[string]
	Start            time.Time
}

func NewPushContext() *PushContext {
	return &PushContext{}
}

type ConfigKey struct {
	Kind      kind.Kind
	Name      string
	Namespace string
}

type exportToDefaults struct {
	service         sets.Set[visibility.Instance]
	virtualService  sets.Set[visibility.Instance]
	destinationRule sets.Set[visibility.Instance]
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

type XDSUpdater interface{}

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

func (ps *PushContext) initServiceRegistry(env *Environment, configsUpdate sets.Set[ConfigKey]) {

}

func (ps *PushContext) initServiceAccounts(env *Environment, services []*Service) {
}

func (ps *PushContext) initVirtualServices(env *Environment) {
}

func (ps *PushContext) initDestinationRules(env *Environment) {
}

func (ps *PushContext) initAuthnPolicies(env *Environment) {
	ps.AuthnPolicies = initAuthenticationPolicies(env)
}

func (ps *PushContext) initAuthorizationPolicies(env *Environment) {
	ps.AuthzPolicies = GetAuthorizationPolicies(env)
}

func (ps *PushContext) createNewContext(env *Environment) {
	ps.initServiceRegistry(env, nil)
	ps.initVirtualServices(env)
	ps.initDestinationRules(env)
	ps.initAuthnPolicies(env)
	ps.initAuthorizationPolicies(env)
}

func (ps *PushContext) updateContext(env *Environment, oldPushContext *PushContext, pushReq *PushRequest) {
	var servicesChanged, virtualServicesChanged, destinationRulesChanged, authnChanged, authzChanged bool

	// We do not need to watch Ingress or Gateway API changes. Both of these have their own controllers which will send
	// events for Istio types (Gateway and VirtualService).
	for conf := range pushReq.ConfigsUpdated {
		switch conf.Kind {
		case kind.DestinationRule:
			destinationRulesChanged = true
		case kind.VirtualService:
			virtualServicesChanged = true
		case kind.AuthorizationPolicy:
			authzChanged = true
		case kind.RequestAuthentication,
			kind.PeerAuthentication:
			authnChanged = true
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

	if virtualServicesChanged {
		ps.initVirtualServices(env)
	} else {
		ps.virtualServiceIndex = oldPushContext.virtualServiceIndex
	}

	if destinationRulesChanged {
		ps.initDestinationRules(env)
	} else {
		ps.destinationRuleIndex = oldPushContext.destinationRuleIndex
	}

	if authnChanged {
		ps.initAuthnPolicies(env)
	} else {
		ps.AuthnPolicies = oldPushContext.AuthnPolicies
	}

	if authzChanged {
		ps.initAuthorizationPolicies(env)
	} else {
		ps.AuthzPolicies = oldPushContext.AuthzPolicies
	}
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

func NewReasonStats(reasons ...TriggerReason) ReasonStats {
	ret := make(ReasonStats)
	for _, reason := range reasons {
		ret.Add(reason)
	}
	return ret
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

func (ps *PushContext) GetAllServices() []*Service {
	return ps.servicesExportedToNamespace(NamespaceAll)
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
