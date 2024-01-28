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

package v3

import (
	"github.com/apache/dubbo-kubernetes/pkg/util/proto"
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_extensions_filters_listener_tls_inspector_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/tls_inspector/v3"
)

type TLSInspectorConfigurer struct{}

var _ ListenerConfigurer = &TLSInspectorConfigurer{}

func (c *TLSInspectorConfigurer) Configure(l *envoy_listener.Listener) error {
	any, err := proto.MarshalAnyDeterministic(&envoy_extensions_filters_listener_tls_inspector_v3.TlsInspector{})
	if err != nil {
		return err
	}
	l.ListenerFilters = append(l.ListenerFilters, &envoy_listener.ListenerFilter{
		Name: "envoy.filters.listener.tls_inspector",
		ConfigType: &envoy_listener.ListenerFilter_TypedConfig{
			TypedConfig: any,
		},
	})
	return nil
}
