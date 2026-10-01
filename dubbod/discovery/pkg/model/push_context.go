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
	"encoding/json"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
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
)

var (
	LastPushStatus *PushContext
	LastPushMutex  sync.Mutex
)

type PushContext struct {
	*ConfigSnapshot
	PushVersion      string
	ProxyStatus      map[string]map[string]ProxyPushStatus
	proxyStatusMutex sync.RWMutex
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

type ReasonStats map[TriggerReason]int

type ResourceDelta = xds.ResourceDelta

type ConfigKey struct {
	Kind      kind.Kind
	Name      string
	Namespace string
}

func NewPushContext() *PushContext {
	return &PushContext{ConfigSnapshot: NewConfigSnapshot(), ProxyStatus: map[string]map[string]ProxyPushStatus{}}
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

func (r ReasonStats) Copy() ReasonStats {
	if len(r) == 0 {
		return nil
	}
	out := make(ReasonStats, len(r))
	for reason, count := range r {
		out[reason] = count
	}
	return out
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

	if pr.Start.IsZero() {
		pr.Start = other.Start
	}

	if pr.ConfigsUpdated == nil {
		if other.ConfigsUpdated != nil {
			pr.ConfigsUpdated = other.ConfigsUpdated.Copy()
		}
	} else {
		pr.ConfigsUpdated.Merge(other.ConfigsUpdated)
	}

	if pr.AddressesUpdated == nil {
		if other.AddressesUpdated != nil {
			pr.AddressesUpdated = other.AddressesUpdated.Copy()
		}
	} else {
		pr.AddressesUpdated.Merge(other.AddressesUpdated)
	}

	pr.Delta = mergeResourceDelta(pr.Delta, other.Delta)

	return pr
}

func mergeResourceDelta(first, second ResourceDelta) ResourceDelta {
	out := copyResourceDelta(first)
	if len(second.Subscribed) > 0 {
		if out.Subscribed == nil {
			out.Subscribed = second.Subscribed.Copy()
		} else {
			out.Subscribed.Merge(second.Subscribed)
		}
	}
	if len(second.Unsubscribed) > 0 {
		if out.Unsubscribed == nil {
			out.Unsubscribed = second.Unsubscribed.Copy()
		} else {
			out.Unsubscribed.Merge(second.Unsubscribed)
		}
	}
	return out
}

func copyResourceDelta(delta ResourceDelta) ResourceDelta {
	out := ResourceDelta{}
	if delta.Subscribed != nil {
		out.Subscribed = delta.Subscribed.Copy()
	}
	if delta.Unsubscribed != nil {
		out.Unsubscribed = delta.Unsubscribed.Copy()
	}
	return out
}

func (pr *PushRequest) Copy() *PushRequest {
	if pr == nil {
		return nil
	}
	out := *pr
	out.Reason = pr.Reason.Copy()
	if pr.ConfigsUpdated != nil {
		out.ConfigsUpdated = pr.ConfigsUpdated.Copy()
	}
	if pr.AddressesUpdated != nil {
		out.AddressesUpdated = pr.AddressesUpdated.Copy()
	}
	out.Delta = copyResourceDelta(pr.Delta)
	return &out
}

func (pr *PushRequest) CopyMerge(other *PushRequest) *PushRequest {
	if pr == nil {
		return other.Copy()
	}
	if other == nil {
		return pr.Copy()
	}

	merged := pr.Copy()
	return merged.Merge(other)
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

func (ps *PushContext) StatusJSON() ([]byte, error) {
	if ps == nil {
		return []byte{'{', '}'}, nil
	}
	ps.proxyStatusMutex.RLock()
	defer ps.proxyStatusMutex.RUnlock()
	if len(ps.ProxyStatus) == 0 {
		return []byte{'{', '}'}, nil
	}
	return json.MarshalIndent(ps.ProxyStatus, "", "    ")
}

func (ps *PushContext) InitContext(env *Environment, old *PushContext, req *PushRequest) {
	var previous *ConfigSnapshot
	if old != nil {
		previous = old.ConfigSnapshot
	}
	ps.Initialize(env, previous, req.ConfigChange())
}

func (req *PushRequest) ConfigChange() *ConfigChange {
	if req == nil {
		return nil
	}
	return &ConfigChange{
		ConfigsUpdated:   req.ConfigsUpdated,
		AddressesUpdated: req.AddressesUpdated,
		Full:             req.Full,
		Forced:           req.Forced,
		EndpointsChanged: req.Reason.Has(EndpointUpdate) || req.Reason.Has(HeadlessEndpointUpdate),
		Global:           req.Reason.Has(GlobalUpdate),
	}
}
