package v3

import (
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tags"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

type ResetTagsHeaderConfigurer struct{}

func (r *ResetTagsHeaderConfigurer) Configure(rc *envoy_config_route_v3.RouteConfiguration) error {
	rc.RequestHeadersToRemove = append(rc.RequestHeadersToRemove, tags.TagsHeaderName)
	return nil
}
