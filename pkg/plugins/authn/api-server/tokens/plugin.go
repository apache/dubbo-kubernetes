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

package tokens

import (
	go_context "context"

	"github.com/pkg/errors"

	"github.com/apache/dubbo-kubernetes/pkg/api-server/authn"
	config_access "github.com/apache/dubbo-kubernetes/pkg/config/access"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	core_tokens "github.com/apache/dubbo-kubernetes/pkg/core/tokens"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/authn/api-server/tokens/access"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/authn/api-server/tokens/issuer"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/authn/api-server/tokens/ws/server"
)

const PluginName = "tokens"

type plugin struct {
	// TODO: properly run AfterBootstrap - https://github.com/apache/dubbo-kubernetes/issues/6607
	isInitialised bool
}

var (
	_ plugins.AuthnAPIServerPlugin = &plugin{}
	_ plugins.BootstrapPlugin      = &plugin{}
)

// We declare AccessStrategies and not into Runtime because it's a plugin.
var AccessStrategies = map[string]func(*plugins.MutablePluginContext) access.GenerateUserTokenAccess{
	config_access.StaticType: func(context *plugins.MutablePluginContext) access.GenerateUserTokenAccess {
		return access.NewStaticGenerateUserTokenAccess(context.Config().Access.Static.GenerateUserToken)
	},
}

var NewUserTokenIssuer = func(signingKeyManager core_tokens.SigningKeyManager) issuer.UserTokenIssuer {
	return issuer.NewUserTokenIssuer(core_tokens.NewTokenIssuer(signingKeyManager))
}

func init() {
	plugins.Register(PluginName, &plugin{})
}

func (c *plugin) NewAuthenticator(context plugins.PluginContext) (authn.Authenticator, error) {
	publicKeys, err := core_tokens.PublicKeyFromConfig(context.Config().ApiServer.Authn.Tokens.Validator.PublicKeys)
	if err != nil {
		return nil, err
	}
	staticSigningKeyAccessor, err := core_tokens.NewStaticSigningKeyAccessor(publicKeys)
	if err != nil {
		return nil, err
	}
	accessors := []core_tokens.SigningKeyAccessor{staticSigningKeyAccessor}
	if context.Config().ApiServer.Authn.Tokens.Validator.UseSecrets {
		accessors = append(accessors, core_tokens.NewSigningKeyAccessor(context.ResourceManager(), issuer.UserTokenSigningKeyPrefix))
	}
	validator := issuer.NewUserTokenValidator(
		core_tokens.NewValidator(
			core.Log.WithName("tokens"),
			accessors,
			core_tokens.NewRevocations(context.ResourceManager(), issuer.UserTokenRevocationsGlobalSecretKey),
			context.Config().Store.Type,
		),
	)
	c.isInitialised = true
	return UserTokenAuthenticator(validator), nil
}

func (c *plugin) BeforeBootstrap(*plugins.MutablePluginContext, plugins.PluginConfig) error {
	return nil
}

func (c *plugin) AfterBootstrap(context *plugins.MutablePluginContext, config plugins.PluginConfig) error {
	if !c.isInitialised {
		return nil
	}
	ctx := go_context.Background()
	signingKeyManager := core_tokens.NewSigningKeyManager(context.ResourceManager(), issuer.UserTokenSigningKeyPrefix)
	component := core_tokens.NewDefaultSigningKeyComponent(ctx, signingKeyManager, log, context.Extensions())
	if err := context.ComponentManager().Add(component); err != nil {
		return err
	}
	accessFn, ok := AccessStrategies[context.Config().Access.Type]
	if !ok {
		return errors.Errorf("no Access strategy for type %q", context.Config().Access.Type)
	}
	tokenIssuer := NewUserTokenIssuer(signingKeyManager)
	if !context.Config().ApiServer.Authn.Tokens.EnableIssuer {
		tokenIssuer = issuer.DisabledIssuer{}
	}
	if context.Config().ApiServer.Authn.Tokens.BootstrapAdminToken {
		if err := context.ComponentManager().Add(NewAdminTokenBootstrap(tokenIssuer, context.ResourceManager(), context.Config())); err != nil {
			return err
		}
	}
	webService := server.NewWebService(tokenIssuer, accessFn(context))
	context.APIManager().Add(webService)
	return nil
}

func (c *plugin) Name() plugins.PluginName {
	return PluginName
}

func (c *plugin) Order() int {
	return plugins.EnvironmentPreparedOrder + 1
}