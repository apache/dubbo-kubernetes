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

package plugins

import (
	"fmt"
	"sort"
)

import (
	"github.com/pkg/errors"
)

type pluginType string

const (
	bootstrapPlugin     pluginType = "bootstrap"
	resourceStorePlugin pluginType = "resource-store"
	configStorePlugin   pluginType = "config-store"
	runtimePlugin       pluginType = "runtime"
	policyPlugin        pluginType = "policy"
	caPlugin            pluginType = "ca"
)

type PluginName string

const (
	Kubernetes PluginName = "k8s"
	Universal  PluginName = "universal"
	Memory     PluginName = "memory"

	CaBuiltin PluginName = "builtin"
)

type RegisteredPolicyPlugin struct {
	Plugin PolicyPlugin
	Name   PluginName
}

type Registry interface {
	BootstrapPlugins() []BootstrapPlugin
	ResourceStore(name PluginName) (ResourceStorePlugin, error)
	ConfigStore(name PluginName) (ConfigStorePlugin, error)
	RuntimePlugins() map[PluginName]RuntimePlugin
	PolicyPlugins([]PluginName) []RegisteredPolicyPlugin
}

type RegistryMutator interface {
	Register(PluginName, Plugin) error
}

type MutableRegistry interface {
	Registry
	RegistryMutator
}

func NewRegistry() MutableRegistry {
	return &registry{
		bootstrap:          make(map[PluginName]BootstrapPlugin),
		resourceStore:      make(map[PluginName]ResourceStorePlugin),
		configStore:        make(map[PluginName]ConfigStorePlugin),
		runtime:            make(map[PluginName]RuntimePlugin),
		registeredPolicies: make(map[PluginName]PolicyPlugin),
	}
}

var _ MutableRegistry = &registry{}

type registry struct {
	bootstrap          map[PluginName]BootstrapPlugin
	resourceStore      map[PluginName]ResourceStorePlugin
	configStore        map[PluginName]ConfigStorePlugin
	runtime            map[PluginName]RuntimePlugin
	registeredPolicies map[PluginName]PolicyPlugin
}

func (r *registry) ResourceStore(name PluginName) (ResourceStorePlugin, error) {
	if p, ok := r.resourceStore[name]; ok {
		return p, nil
	} else {
		return nil, noSuchPluginError(resourceStorePlugin, name)
	}
}

func (r *registry) ConfigStore(name PluginName) (ConfigStorePlugin, error) {
	if p, ok := r.configStore[name]; ok {
		return p, nil
	} else {
		return nil, noSuchPluginError(configStorePlugin, name)
	}
}

func (r *registry) RuntimePlugins() map[PluginName]RuntimePlugin {
	return r.runtime
}

func (r *registry) PolicyPlugins(ordered []PluginName) []RegisteredPolicyPlugin {
	var plugins []RegisteredPolicyPlugin
	for _, policy := range ordered {
		plugin, ok := r.registeredPolicies[policy]
		if !ok {
			panic(fmt.Sprintf("Couldn't find plugin %s", policy))
		}
		plugins = append(plugins, RegisteredPolicyPlugin{
			Plugin: plugin,
			Name:   policy,
		})
	}
	return plugins
}

func (r *registry) BootstrapPlugins() []BootstrapPlugin {
	var plugins []BootstrapPlugin
	for _, plugin := range r.bootstrap {
		plugins = append(plugins, plugin)
	}
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Order() < plugins[j].Order()
	})
	return plugins
}

func (r *registry) BootstrapPlugin(name PluginName) (BootstrapPlugin, error) {
	if p, ok := r.bootstrap[name]; ok {
		return p, nil
	} else {
		return nil, noSuchPluginError(bootstrapPlugin, name)
	}
}

func (r *registry) Register(name PluginName, plugin Plugin) error {
	if bp, ok := plugin.(BootstrapPlugin); ok {
		if old, exists := r.bootstrap[name]; exists {
			return pluginAlreadyRegisteredError(bootstrapPlugin, name, old, bp)
		}
		r.bootstrap[name] = bp
	}
	if rsp, ok := plugin.(ResourceStorePlugin); ok {
		r.resourceStore[name] = rsp
	}
	if csp, ok := plugin.(ConfigStorePlugin); ok {
		if old, exists := r.configStore[name]; exists {
			return pluginAlreadyRegisteredError(configStorePlugin, name, old, csp)
		}
		r.configStore[name] = csp
	}
	if rp, ok := plugin.(RuntimePlugin); ok {
		if old, exists := r.runtime[name]; exists {
			return pluginAlreadyRegisteredError(runtimePlugin, name, old, rp)
		}
		r.runtime[name] = rp
	}
	if policy, ok := plugin.(PolicyPlugin); ok {
		if old, exists := r.registeredPolicies[name]; exists {
			return pluginAlreadyRegisteredError(policyPlugin, name, old, policy)
		}
		r.registeredPolicies[name] = policy
	}
	return nil
}

func noSuchPluginError(typ pluginType, name PluginName) error {
	return errors.Errorf("there is no plugin registered with type=%q and name=%s", typ, name)
}

func pluginAlreadyRegisteredError(typ pluginType, name PluginName, old, new Plugin) error {
	return errors.Errorf("plugin with type=%q and name=%s has already been registered: old=%#v new=%#v",
		typ, name, old, new)
}
