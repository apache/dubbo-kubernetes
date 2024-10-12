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

package generator

import (
	"context"
	"fmt"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"

	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	util_maps "github.com/apache/dubbo-kubernetes/pkg/util/maps"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	envoy_clusters "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/clusters"
	envoy_listeners "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/listeners"
	envoy_names "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/names"
)

// OriginAdmin is a marker to indicate by which ProxyGenerator resources were generated.
const OriginAdmin = "admin"

var staticEndpointPaths = []*envoy_common.StaticEndpointPath{
	{
		Path:        "/ready",
		RewritePath: "/ready",
	},
}

var staticTlsEndpointPaths = []*envoy_common.StaticEndpointPath{
	{
		Path:        "/",
		RewritePath: "/",
	},
}

// AdminProxyGenerator generates resources to expose some endpoints of Admin API on public interface.
// By default, Admin API is exposed only on loopback interface because of security reasons.
type AdminProxyGenerator struct{}

var adminAddressAllowedValues = map[string]struct{}{
	"127.0.0.1": {},
	"0.0.0.0":   {},
	"::1":       {},
	"::":        {},
	"":          {},
}

func (g AdminProxyGenerator) Generator(ctx context.Context, _ *core_xds.ResourceSet, xdsCtx xds_context.Context, proxy *core_xds.Proxy) (*core_xds.ResourceSet, error) {
	if proxy.Metadata.GetAdminPort() == 0 {
		// It's not possible to export Admin endpoints if Envoy Admin API has not been enabled on that dataplane.
		return nil, nil
	}

	adminPort := proxy.Metadata.GetAdminPort()
	// We assume that Admin API must be available on a loopback interface (while users
	// can override the default value `127.0.0.1` in the Bootstrap Server section of `dubbo-cp` config,
	// the only reasonable alternatives are `::1`, `0.0.0.0` or `::`).
	// In contrast to `AdminPort`, we shouldn't trust `AdminAddress` from the Envoy node metadata
	// since it would allow a malicious user to manipulate that value and use Prometheus endpoint
	// as a gateway to another host.
	envoyAdminClusterName := envoy_names.GetEnvoyAdminClusterName()
	adminAddress := proxy.Metadata.GetAdminAddress()
	if _, ok := adminAddressAllowedValues[adminAddress]; !ok {
		var allowedAddresses []string
		for _, address := range util_maps.SortedKeys(adminAddressAllowedValues) {
			allowedAddresses = append(allowedAddresses, fmt.Sprintf(`"%s"`, address))
		}
		return nil, errors.Errorf("envoy admin cluster is not allowed to have addresses other than %s", strings.Join(allowedAddresses, ", "))
	}
	switch adminAddress {
	case "", "0.0.0.0":
		adminAddress = "127.0.0.1"
	case "::":
		adminAddress = "::1"
	}
	cluster, err := envoy_clusters.NewClusterBuilder(proxy.APIVersion, envoyAdminClusterName).
		Configure(envoy_clusters.ProvidedEndpointCluster(
			govalidator.IsIPv6(adminAddress),
			core_xds.Endpoint{Target: adminAddress, Port: adminPort})).
		Configure(envoy_clusters.DefaultTimeout()).
		Build()
	if err != nil {
		return nil, err
	}

	resources := core_xds.NewResourceSet()

	for _, se := range staticEndpointPaths {
		se.ClusterName = envoyAdminClusterName
	}

	// We bind admin to 127.0.0.1 by default, creating another listener with same address and port will result in error.
	if g.getAddress(proxy) != adminAddress {
		filterChains := []envoy_listeners.ListenerBuilderOpt{
			envoy_listeners.FilterChain(envoy_listeners.NewFilterChainBuilder(proxy.APIVersion, envoy_common.AnonymousResource).
				Configure(envoy_listeners.StaticEndpoints(envoy_names.GetAdminListenerName(), staticEndpointPaths)),
			),
		}
		for _, se := range staticTlsEndpointPaths {
			se.ClusterName = envoyAdminClusterName
		}
		filterChains = append(filterChains, envoy_listeners.FilterChain(envoy_listeners.NewFilterChainBuilder(proxy.APIVersion, envoy_common.AnonymousResource).
			Configure(envoy_listeners.MatchTransportProtocol("tls")).
			Configure(envoy_listeners.StaticEndpoints(envoy_names.GetAdminListenerName(), staticTlsEndpointPaths)).
			Configure(envoy_listeners.ServerSideStaticMTLS(proxy.EnvoyAdminMTLSCerts)),
		))

		listener, err := envoy_listeners.NewInboundListenerBuilder(proxy.APIVersion, g.getAddress(proxy), adminPort, core_xds.SocketAddressProtocolTCP).
			WithOverwriteName(envoy_names.GetAdminListenerName()).
			Configure(envoy_listeners.TLSInspector()).
			Configure(filterChains...).
			Build()
		if err != nil {
			return nil, err
		}
		resources.Add(&core_xds.Resource{
			Name:     listener.GetName(),
			Origin:   OriginAdmin,
			Resource: listener,
		})
	}

	resources.Add(&core_xds.Resource{
		Name:     cluster.GetName(),
		Origin:   OriginAdmin,
		Resource: cluster,
	})
	return resources, nil
}

func (g AdminProxyGenerator) getAddress(proxy *core_xds.Proxy) string {
	if proxy.Dataplane != nil {
		return proxy.Dataplane.Spec.GetNetworking().Address
	}

	//TODO: ZoneEgressProxy
	//if proxy.ZoneEgressProxy != nil {
	//	return proxy.ZoneEgressProxy.ZoneEgressResource.Spec.GetNetworking().GetAddress()
	//}

	if proxy.ZoneIngressProxy != nil {
		return proxy.ZoneIngressProxy.ZoneIngressResource.Spec.GetNetworking().GetAddress()
	}

	return ""
}
