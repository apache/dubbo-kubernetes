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

package bootstrap

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	dp_server "github.com/apache/dubbo-kubernetes/pkg/config/dp-server"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func RegisterBootstrap(rt core_runtime.RuntimeContext) error {
	generator, err := NewDefaultBootstrapGenerator(
		rt.ResourceManager(),
		rt.Config().BootstrapServer,
		rt.Config().Proxy,
		rt.Config().DpServer.TlsCertFile,
		map[string]bool{
			string(mesh_proto.DataplaneProxyType): rt.Config().DpServer.Authn.DpProxy.Type != dp_server.DpServerAuthNone,
			string(mesh_proto.IngressProxyType):   rt.Config().DpServer.Authn.ZoneProxy.Type != dp_server.DpServerAuthNone,
			string(mesh_proto.EgressProxyType):    rt.Config().DpServer.Authn.ZoneProxy.Type != dp_server.DpServerAuthNone,
		},
		rt.Config().DpServer.Authn.EnableReloadableTokens,
		rt.Config().DpServer.Hds.Enabled,
		rt.Config().GetEnvoyAdminPort(),
	)
	if err != nil {
		return err
	}
	bootstrapHandler := BootstrapHandler{
		Generator: generator,
	}
	log.Info("registering Bootstrap in Dataplane Server")
	rt.DpServer().HTTPMux().HandleFunc("/bootstrap", bootstrapHandler.Handle)
	return nil
}
