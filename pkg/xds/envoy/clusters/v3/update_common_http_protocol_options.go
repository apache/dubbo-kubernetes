package clusters

import (
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
	envoy_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_upstream_http "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

func UpdateCommonHttpProtocolOptions(cluster *envoy_cluster.Cluster, fn func(*envoy_upstream_http.HttpProtocolOptions)) error {
	if cluster.TypedExtensionProtocolOptions == nil {
		cluster.TypedExtensionProtocolOptions = map[string]*anypb.Any{}
	}
	options := &envoy_upstream_http.HttpProtocolOptions{}
	if any := cluster.TypedExtensionProtocolOptions["envoy.extensions.upstreams.http.v3.HttpProtocolOptions"]; any != nil {
		if err := util_proto.UnmarshalAnyTo(any, options); err != nil {
			return err
		}
	}

	fn(options)

	pbst, err := util_proto.MarshalAnyDeterministic(options)
	if err != nil {
		return err
	}
	cluster.TypedExtensionProtocolOptions["envoy.extensions.upstreams.http.v3.HttpProtocolOptions"] = pbst
	return nil
}
