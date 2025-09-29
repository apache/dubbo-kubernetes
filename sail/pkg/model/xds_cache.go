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
	"github.com/apache/dubbo-kubernetes/sail/pkg/features"
)

type XdsCache interface {
	Run(stop <-chan struct{})
}

type DisabledCache struct{}

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

func (x XdsCacheImpl) Run(stop <-chan struct{}) {}

func (d DisabledCache) Run(stop <-chan struct{}) {
}
