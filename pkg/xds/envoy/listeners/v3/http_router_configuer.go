package v3

import (
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_router "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	envoy_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

// HTTPRouterStartChildSpanRouter configures the router to start child spans.
type HTTPRouterStartChildSpanRouter struct{}

var _ FilterChainConfigurer = &HTTPRouterStartChildSpanRouter{}

func (c *HTTPRouterStartChildSpanRouter) Configure(filterChain *envoy_listener.FilterChain) error {
	return UpdateHTTPConnectionManager(filterChain, func(hcm *envoy_hcm.HttpConnectionManager) error {
		typedConfig, err := util_proto.MarshalAnyDeterministic(&envoy_router.Router{
			StartChildSpan: true,
		})
		if err != nil {
			return err
		}
		router := &envoy_hcm.HttpFilter{
			Name: "envoy.filters.http.router",
			ConfigType: &envoy_hcm.HttpFilter_TypedConfig{
				TypedConfig: typedConfig,
			},
		}
		hcm.HttpFilters = append(hcm.HttpFilters, router)
		return nil
	})
}
