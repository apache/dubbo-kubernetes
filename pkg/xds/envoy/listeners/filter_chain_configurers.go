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

package listeners

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	v3 "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/listeners/v3"
	envoy_routes "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/routes"
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tags"
	envoy_config_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"

	envoy_extensions_compression_gzip_compressor_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/compression/gzip/compressor/v3"
	envoy_extensions_filters_http_compressor_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/compressor/v3"
	envoy_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

func GrpcStats() FilterChainBuilderOpt {
	return AddFilterChainConfigurer(&v3.GrpcStatsConfigurer{})
}

func Kafka(statsName string) FilterChainBuilderOpt {
	return AddFilterChainConfigurer(&v3.KafkaConfigurer{
		StatsName: statsName,
	})
}

func StaticEndpoints(virtualHostName string, paths []*envoy_common.StaticEndpointPath) FilterChainBuilderOpt {
	return AddFilterChainConfigurer(&v3.StaticEndpointsConfigurer{
		VirtualHostName: virtualHostName,
		Paths:           paths,
	})
}

func DirectResponse(virtualHostName string, endpoints []v3.DirectResponseEndpoints) FilterChainBuilderOpt {
	return AddFilterChainConfigurer(&v3.DirectResponseConfigurer{
		VirtualHostName: virtualHostName,
		Endpoints:       endpoints,
	})
}

func HttpConnectionManager(statsName string, forwardClientCertDetails bool) FilterChainBuilderOpt {
	return AddFilterChainConfigurer(&v3.HttpConnectionManagerConfigurer{
		StatsName:                statsName,
		ForwardClientCertDetails: forwardClientCertDetails,
	})
}

type splitAdapter struct {
	clusterName string
	weight      uint32
	lbMetadata  tags.Tags

	hasExternalService bool
}

func (s *splitAdapter) ClusterName() string      { return s.clusterName }
func (s *splitAdapter) Weight() uint32           { return s.weight }
func (s *splitAdapter) LBMetadata() tags.Tags    { return s.lbMetadata }
func (s *splitAdapter) HasExternalService() bool { return s.hasExternalService }

func TcpProxyDeprecated(statsName string, clusters ...envoy_common.Cluster) FilterChainBuilderOpt {
	var splits []envoy_common.Split
	for _, cluster := range clusters {
		cluster := cluster.(*envoy_common.ClusterImpl)
		splits = append(splits, &splitAdapter{
			clusterName:        cluster.Name(),
			weight:             cluster.Weight(),
			lbMetadata:         cluster.Tags(),
			hasExternalService: cluster.IsExternalService(),
		})
	}
	return AddFilterChainConfigurer(&v3.TcpProxyConfigurer{
		StatsName:   statsName,
		Splits:      splits,
		UseMetadata: false,
	})
}

func TcpProxyDeprecatedWithMetadata(statsName string, clusters ...envoy_common.Cluster) FilterChainBuilderOpt {
	var splits []envoy_common.Split
	for _, cluster := range clusters {
		cluster := cluster.(*envoy_common.ClusterImpl)
		splits = append(splits, &splitAdapter{
			clusterName:        cluster.Name(),
			weight:             cluster.Weight(),
			lbMetadata:         cluster.Tags(),
			hasExternalService: cluster.IsExternalService(),
		})
	}
	return AddFilterChainConfigurer(&v3.TcpProxyConfigurer{
		StatsName:   statsName,
		Splits:      splits,
		UseMetadata: true,
	})
}

func TCPProxy(statsName string, splits ...envoy_common.Split) FilterChainBuilderOpt {
	return AddFilterChainConfigurer(&v3.TcpProxyConfigurer{
		StatsName:   statsName,
		Splits:      splits,
		UseMetadata: true,
	})
}

func HttpStaticRoute(builder *envoy_routes.RouteConfigurationBuilder) FilterChainBuilderOpt {
	return AddFilterChainConfigurer(&v3.HttpStaticRouteConfigurer{
		Builder: builder,
	})
}

// HttpDynamicRoute configures the listener filter chain to dynamically request
// the named RouteConfiguration.
func HttpDynamicRoute(name string) FilterChainBuilderOpt {
	return AddFilterChainConfigurer(&v3.HttpDynamicRouteConfigurer{
		RouteName: name,
	})
}

func HttpInboundRoutes(service string, routes envoy_common.Routes) FilterChainBuilderOpt {
	return AddFilterChainConfigurer(&v3.HttpInboundRouteConfigurer{
		Service: service,
		Routes:  routes,
	})
}

func HttpOutboundRoute(service string, routes envoy_common.Routes, dpTags mesh_proto.MultiValueTagSet) FilterChainBuilderOpt {
	return AddFilterChainConfigurer(&v3.HttpOutboundRouteConfigurer{
		Service: service,
		Routes:  routes,
		DpTags:  dpTags,
	})
}

// ServerHeader sets the value that the HttpConnectionManager will write
// to the "Server" header in HTTP responses.
func ServerHeader(name string) FilterChainBuilderOpt {
	return AddFilterChainConfigurer(
		v3.HttpConnectionManagerMustConfigureFunc(func(hcm *envoy_hcm.HttpConnectionManager) {
			hcm.ServerName = name
		}),
	)
}

// EnablePathNormalization enables HTTP request path normalization.
func EnablePathNormalization() FilterChainBuilderOpt {
	return AddFilterChainConfigurer(
		v3.HttpConnectionManagerMustConfigureFunc(func(hcm *envoy_hcm.HttpConnectionManager) {
			hcm.NormalizePath = util_proto.Bool(true)
			hcm.MergeSlashes = true
			hcm.PathWithEscapedSlashesAction = envoy_hcm.HttpConnectionManager_UNESCAPE_AND_REDIRECT
		}),
	)
}

// StripHostPort strips the port component before matching the HTTP host
// header (authority) to the available virtual hosts.
func StripHostPort() FilterChainBuilderOpt {
	return AddFilterChainConfigurer(
		v3.HttpConnectionManagerMustConfigureFunc(func(hcm *envoy_hcm.HttpConnectionManager) {
			hcm.StripPortMode = &envoy_hcm.HttpConnectionManager_StripAnyHostPort{
				StripAnyHostPort: true,
			}
		}),
	)
}

// DefaultCompressorFilter adds a gzip compressor filter in its default configuration.
func DefaultCompressorFilter() FilterChainBuilderOpt {
	return AddFilterChainConfigurer(
		v3.HttpConnectionManagerMustConfigureFunc(func(hcm *envoy_hcm.HttpConnectionManager) {
			c := envoy_extensions_filters_http_compressor_v3.Compressor{
				CompressorLibrary: &envoy_config_core.TypedExtensionConfig{
					Name:        "gzip",
					TypedConfig: util_proto.MustMarshalAny(&envoy_extensions_compression_gzip_compressor_v3.Gzip{}),
				},
				ResponseDirectionConfig: &envoy_extensions_filters_http_compressor_v3.Compressor_ResponseDirectionConfig{
					DisableOnEtagHeader: true,
				},
			}

			gzip := &envoy_hcm.HttpFilter{
				Name: "gzip-compress",
				ConfigType: &envoy_hcm.HttpFilter_TypedConfig{
					TypedConfig: util_proto.MustMarshalAny(&c),
				},
			}

			hcm.HttpFilters = append(hcm.HttpFilters, gzip)
		}),
	)
}
