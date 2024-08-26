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

package certs

import (
	"github.com/apache/dubbo-kubernetes/pkg/api-server/authn"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/plugins"
)

const PluginName = "adminClientCerts"

var log = core.Log.WithName("plugins").WithName("authn").WithName("api-server").WithName("certs")

type plugin struct{}

func init() {
	plugins.Register(PluginName, &plugin{})
}

var _ plugins.AuthnAPIServerPlugin = plugin{}

func (c plugin) NewAuthenticator(_ plugins.PluginContext) (authn.Authenticator, error) {
	log.Info("WARNING: admin client certificates are deprecated. Please migrate to user token as API Server authentication mechanism.")
	return ClientCertAuthenticator, nil
}
