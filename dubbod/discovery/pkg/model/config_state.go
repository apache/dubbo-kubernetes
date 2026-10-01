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

import "github.com/apache/dubbo-kubernetes/pkg/util/sets"

// ConfigChange describes source changes without carrying an xDS connection or delivery state.
type ConfigChange struct {
	ConfigsUpdated   sets.Set[ConfigKey]
	AddressesUpdated sets.String
	Full             bool
	Forced           bool
	EndpointsChanged bool
	Global           bool
}

func (e *Environment) ConfigSnapshot() *ConfigSnapshot {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.configSnapshot
}

// ReconcileConfig publishes one completed snapshot before notifying its consumers.
func (e *Environment) ReconcileConfig(change *ConfigChange) *ConfigSnapshot {
	e.configMutex.Lock()
	defer e.configMutex.Unlock()
	current := NewConfigSnapshot()
	current.Initialize(e, e.ConfigSnapshot(), change)
	e.mutex.Lock()
	e.configSnapshot = current
	e.mutex.Unlock()
	e.NotifyConfigChange(change)
	return current
}

func (e *Environment) AddConfigHandler(handler func(*ConfigChange)) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.configHandlers = append(e.configHandlers, handler)
}

// NotifyConfigChange also covers endpoint changes that reuse the policy snapshot.
func (e *Environment) NotifyConfigChange(change *ConfigChange) {
	e.mutex.RLock()
	handlers := append([]func(*ConfigChange){}, e.configHandlers...)
	e.mutex.RUnlock()
	for _, handler := range handlers {
		handler(change)
	}
}
