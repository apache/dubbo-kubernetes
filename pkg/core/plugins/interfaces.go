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
	"context"
	core_ca "github.com/apache/dubbo-kubernetes/pkg/core/ca"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	secret_store "github.com/apache/dubbo-kubernetes/pkg/core/secrets/store"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/pkg/errors"
)

type Plugin interface{}

type PluginConfig interface{}

type PluginContext = core_runtime.BuilderContext

type MutablePluginContext = core_runtime.Builder

// EnvironmentPreparingOrder describes an order at which base environment plugins (Universal/Kubernetes) configure the control plane.
var EnvironmentPreparingOrder = 0

// EnvironmentPreparedOrder describes an order at which you can put a plugin and expect that
// the base environment is already configured by Universal/Kubernetes plugins.
var EnvironmentPreparedOrder = EnvironmentPreparingOrder + 1

// BootstrapPlugin is responsible for environment-specific initialization at start up,
// e.g. Kubernetes-specific part of configuration.
// Unlike other plugins, can mutate plugin context directly.
type BootstrapPlugin interface {
	Plugin
	BeforeBootstrap(*MutablePluginContext, PluginConfig) error
	AfterBootstrap(*MutablePluginContext, PluginConfig) error
	Name() PluginName
	// Order defines an order in which plugins are applied on the control plane.
	// If you don't have specific need, consider using EnvironmentPreparedOrder
	Order() int
}

// ResourceStorePlugin is responsible for instantiating a particular ResourceStore.
type (
	DbVersion           = uint
	ResourceStorePlugin interface {
		Plugin
		NewResourceStore(PluginContext, PluginConfig) (core_store.ResourceStore, core_store.Transactions, error)
		Migrate(PluginContext, PluginConfig) (DbVersion, error)
		EventListener(PluginContext, events.Emitter) error
	}
)

var AlreadyMigrated = errors.New("database already migrated")

// ConfigStorePlugin is responsible for instantiating a particular ConfigStore.
type ConfigStorePlugin interface {
	Plugin
	NewConfigStore(PluginContext, PluginConfig) (core_store.ResourceStore, error)
}

// RuntimePlugin is responsible for registering environment-specific components,
// e.g. Kubernetes admission web hooks.
type RuntimePlugin interface {
	Plugin
	Customize(core_runtime.Runtime) error
}

// PolicyPlugin a plugin to add a Policy to dubbo
type PolicyPlugin interface {
	Plugin
	// MatchedPolicies accessible in Apply through `proxy.Policies.Dynamic`
	MatchedPolicies(dataplane *core_mesh.DataplaneResource, resource xds_context.Resources) (core_xds.TypedMatchingPolicies, error)
	// Apply to `rs` using the `ctx` and `proxy` the mutation for all policies of the type this plugin implements.
	// You can access matching policies by using `proxy.Policies.Dynamic`.
	Apply(rs *core_xds.ResourceSet, ctx xds_context.Context, proxy *core_xds.Proxy) error
}

type CaPlugin interface {
	Plugin
	NewCaManager(PluginContext, PluginConfig) (core_ca.Manager, error)
}

// AuthnAPIServerPlugin is responsible for providing authenticator for API Server.

type SecretStorePlugin interface {
	Plugin
	NewSecretStore(PluginContext, PluginConfig) (secret_store.SecretStore, error)
}

// AuthnAPIServerPlugin is responsible for providing authenticator for API Server.
type AuthnAPIServerPlugin interface {
	Plugin
	NewAuthenticator(PluginContext) (authn.Authenticator, error)
}

type MatchedPoliciesOption func(*MatchedPoliciesConfig)

func IncludeShadow() MatchedPoliciesOption {
	return func(cfg *MatchedPoliciesConfig) {
		cfg.IncludeShadow = true
	}
}

type MatchedPoliciesConfig struct {
	IncludeShadow bool
}

func NewMatchedPoliciesConfig(opts ...MatchedPoliciesOption) *MatchedPoliciesConfig {
	cfg := &MatchedPoliciesConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

type EgressPolicyPlugin interface {
	PolicyPlugin
	// EgressMatchedPolicies returns all the policies of the plugins' type matching the external service that
	// should be applied on the zone egress.
	EgressMatchedPolicies(tags map[string]string, resources xds_context.Resources, opts ...MatchedPoliciesOption) (core_xds.TypedMatchingPolicies, error)
}

// ProxyPlugin a plugin to modify the proxy. This happens before any `PolicyPlugin` or any envoy generation. and it is applied both for Dataplanes and ZoneProxies
type ProxyPlugin interface {
	Plugin
	// Apply mutate the proxy as needed.
	Apply(ctx context.Context, meshCtx xds_context.MeshContext, proxy *core_xds.Proxy) error
}
