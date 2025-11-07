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
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/features"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"k8s.io/klog/v2"
)

type XdsCacheImpl struct {
	cds typedXdsCache[uint64]
	eds typedXdsCache[uint64]
	rds typedXdsCache[uint64]
	sds typedXdsCache[string]
}

const (
	CDSType = "cds"
	EDSType = "eds"
	RDSType = "rds"
	SDSType = "sds"
)

type XdsCache interface {
	Run(stop <-chan struct{})
	Add(entry XdsCacheEntry, pushRequest *PushRequest, value *discovery.Resource)
	Get(entry XdsCacheEntry) *discovery.Resource
	Clear(sets.Set[ConfigKey])
	ClearAll()
	Keys(t string) []any
	Snapshot() []*discovery.Resource
}

func NewXdsCache() XdsCache {
	cache := XdsCacheImpl{
		eds: newTypedXdsCache[uint64](),
	}
	if features.EnableCDSCaching {
		cache.cds = newTypedXdsCache[uint64]()
	} else {
		cache.cds = disabledCache[uint64]{}
	}
	if features.EnableRDSCaching {
		cache.rds = newTypedXdsCache[uint64]()
	} else {
		cache.rds = disabledCache[uint64]{}
	}

	cache.sds = newTypedXdsCache[string]()

	return cache
}

type XdsCacheEntry interface {
	Type() string
	Key() any
	DependentConfigs() []ConfigHash
	Cacheable() bool
}

func (x XdsCacheImpl) Run(stop <-chan struct{}) {
	interval := features.XDSCacheIndexClearInterval
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				x.cds.Flush()
				x.eds.Flush()
				x.rds.Flush()
				x.sds.Flush()
			case <-stop:
				return
			}
		}
	}()
}

func (x XdsCacheImpl) Add(entry XdsCacheEntry, pushRequest *PushRequest, value *discovery.Resource) {
	if !entry.Cacheable() {
		return
	}
	k := entry.Key()
	switch entry.Type() {
	case CDSType:
		key := k.(uint64)
		x.cds.Add(key, entry, pushRequest, value)
	case EDSType:
		key := k.(uint64)
		x.eds.Add(key, entry, pushRequest, value)
	case SDSType:
		key := k.(string)
		x.sds.Add(key, entry, pushRequest, value)
	case RDSType:
		key := k.(uint64)
		x.rds.Add(key, entry, pushRequest, value)
	default:
		klog.Errorf("unknown type %s", entry.Type())
	}
}

func (x XdsCacheImpl) Get(entry XdsCacheEntry) *discovery.Resource {
	if !entry.Cacheable() {
		return nil
	}

	k := entry.Key()
	switch entry.Type() {
	case CDSType:
		key := k.(uint64)
		return x.cds.Get(key)
	case EDSType:
		key := k.(uint64)
		return x.eds.Get(key)
	case SDSType:
		key := k.(string)
		return x.sds.Get(key)
	case RDSType:
		key := k.(uint64)
		return x.rds.Get(key)
	default:
		klog.Errorf("unknown type %s", entry.Type())
		return nil
	}
}

func (x XdsCacheImpl) Clear(s sets.Set[ConfigKey]) {
	x.cds.Clear(s)
	// clear all EDS cache for PA change
	if HasConfigsOfKind(s, kind.PeerAuthentication) {
		x.eds.ClearAll()
	} else {
		x.eds.Clear(s)
	}
	x.rds.Clear(s)
	x.sds.Clear(s)
}

func (x XdsCacheImpl) ClearAll() {
	x.cds.ClearAll()
	x.eds.ClearAll()
	x.rds.ClearAll()
	x.sds.ClearAll()
}

func (x XdsCacheImpl) Keys(t string) []any {
	switch t {
	case CDSType:
		keys := x.cds.Keys()
		return convertToAnySlices(keys)
	case EDSType:
		keys := x.eds.Keys()
		return convertToAnySlices(keys)
	case SDSType:
		keys := x.sds.Keys()
		return convertToAnySlices(keys)
	case RDSType:
		keys := x.rds.Keys()
		return convertToAnySlices(keys)
	default:
		return nil
	}
}

func convertToAnySlices[K comparable](in []K) []any {
	out := make([]any, len(in))
	for i, k := range in {
		out[i] = k
	}
	return out
}

func (x XdsCacheImpl) Snapshot() []*discovery.Resource {
	var out []*discovery.Resource
	out = append(out, x.cds.Snapshot()...)
	out = append(out, x.eds.Snapshot()...)
	out = append(out, x.rds.Snapshot()...)
	out = append(out, x.sds.Snapshot()...)
	return out
}

// DisabledCache is a cache that is always empty
type DisabledCache struct{}

func (d DisabledCache) Run(stop <-chan struct{}) {
}

func (d DisabledCache) Add(entry XdsCacheEntry, pushRequest *PushRequest, value *discovery.Resource) {
}

func (d DisabledCache) Get(entry XdsCacheEntry) *discovery.Resource {
	return nil
}

func (d DisabledCache) Clear(s sets.Set[ConfigKey]) {
}

func (d DisabledCache) ClearAll() {
}

func (d DisabledCache) Keys(t string) []any {
	return nil
}

func (d DisabledCache) Snapshot() []*discovery.Resource {
	return nil
}

var _ XdsCache = &DisabledCache{}
