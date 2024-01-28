package xds

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tags"
	"github.com/apache/dubbo-kubernetes/pkg/xds/generator"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

type EndpointMap map[xds.ServiceName][]*endpointv3.ClusterLoadAssignment

func GatherOutboundEndpoints(rs *xds.ResourceSet) EndpointMap {
	return gatherEndpoints(rs, generator.OriginOutbound)
}

func gatherEndpoints(rs *xds.ResourceSet, origin string) EndpointMap {
	em := EndpointMap{}
	for _, res := range rs.Resources(envoy_resource.EndpointType) {
		if res.Origin != origin {
			continue
		}

		cla := res.Resource.(*endpointv3.ClusterLoadAssignment)
		serviceName := tags.ServiceFromClusterName(cla.ClusterName)
		em[serviceName] = append(em[serviceName], cla)
	}
	for _, res := range rs.Resources(envoy_resource.ClusterType) {
		if res.Origin != origin {
			continue
		}

		cluster := res.Resource.(*clusterv3.Cluster)
		serviceName := tags.ServiceFromClusterName(cluster.Name)
		if cluster.LoadAssignment != nil {
			em[serviceName] = append(em[serviceName], cluster.LoadAssignment)
		}
	}
	return em
}
