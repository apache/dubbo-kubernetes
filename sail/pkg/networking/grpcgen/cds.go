package grpcgen

import (
	"fmt"

	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/networking/util"
	"github.com/apache/dubbo-kubernetes/sail/pkg/util/protoconv"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"k8s.io/klog/v2"
)

func (g *GrpcConfigGenerator) BuildClusters(node *model.Proxy, push *model.PushContext, names []string) model.Resources {
	filter := newClusterFilter(names)
	clusters := make([]*cluster.Cluster, 0, len(names))
	for defaultClusterName, subsetFilter := range filter {
		builder, err := newClusterBuilder(node, push, defaultClusterName, subsetFilter)
		if err != nil {
			klog.Warning(err)
			continue
		}
		clusters = append(clusters, builder.build()...)
	}

	resp := make(model.Resources, 0, len(clusters))
	for _, c := range clusters {
		resp = append(resp, &discovery.Resource{
			Name:     c.Name,
			Resource: protoconv.MessageToAny(c),
		})
	}
	if len(resp) == 0 && len(names) == 0 {
		klog.Warningf("did not generate any cds for %s; no names provided", node.ID)
	}
	return resp
}

func newClusterFilter(names []string) map[string]sets.String {
	filter := map[string]sets.String{}
	for _, name := range names {
		dir, _, hn, p := model.ParseSubsetKey(name)
		defaultKey := model.BuildSubsetKey(dir, "", hn, p)
		sets.InsertOrNew(filter, defaultKey, name)
	}
	return filter
}

func newClusterBuilder(node *model.Proxy, push *model.PushContext, defaultClusterName string, filter sets.String) (*clusterBuilder, error) {
	_, _, hostname, portNum := model.ParseSubsetKey(defaultClusterName)
	if hostname == "" || portNum == 0 {
		return nil, fmt.Errorf("failed parsing subset key: %s", defaultClusterName)
	}

	// try to resolve the service and port
	var port *model.Port
	svc := push.ServiceForHostname(node, hostname)
	if svc == nil {
		return nil, fmt.Errorf("cds gen for %s: did not find service for cluster %s", node.ID, defaultClusterName)
	}

	port, ok := svc.Ports.GetByPort(portNum)
	if !ok {
		return nil, fmt.Errorf("cds gen for %s: did not find port %d in service for cluster %s", node.ID, portNum, defaultClusterName)
	}

	return &clusterBuilder{
		node: node,
		push: push,

		defaultClusterName: defaultClusterName,
		hostname:           hostname,
		portNum:            portNum,
		filter:             filter,

		svc:  svc,
		port: port,
	}, nil
}

type clusterBuilder struct {
	push *model.PushContext
	node *model.Proxy

	defaultClusterName string
	hostname           host.Name
	portNum            int

	// may not be set
	svc    *model.Service
	port   *model.Port
	filter sets.String
}

func (b *clusterBuilder) build() []*cluster.Cluster {
	var defaultCluster *cluster.Cluster
	if b.filter.Contains(b.defaultClusterName) {
		defaultCluster = b.edsCluster(b.defaultClusterName)
		// CRITICAL: For gRPC proxyless, we need to set CommonLbConfig to handle endpoint health status
		// This ensures that the cluster can use healthy endpoints for load balancing
		if defaultCluster.CommonLbConfig == nil {
			defaultCluster.CommonLbConfig = &cluster.Cluster_CommonLbConfig{}
		}
		if b.svc.SupportsDrainingEndpoints() {
			// see core/v1alpha3/cluster.go
			defaultCluster.CommonLbConfig.OverrideHostStatus = &core.HealthStatusSet{
				Statuses: []core.HealthStatus{
					core.HealthStatus_HEALTHY,
					core.HealthStatus_DRAINING, core.HealthStatus_UNKNOWN, core.HealthStatus_DEGRADED,
				},
			}
		} else {
			// For gRPC proxyless, only use HEALTHY endpoints by default
			defaultCluster.CommonLbConfig.OverrideHostStatus = &core.HealthStatusSet{
				Statuses: []core.HealthStatus{
					core.HealthStatus_HEALTHY,
				},
			}
		}
	}

	subsetClusters := b.applyDestinationRule(defaultCluster)
	out := make([]*cluster.Cluster, 0, 1+len(subsetClusters))
	if defaultCluster != nil {
		out = append(out, defaultCluster)
	}
	return append(out, subsetClusters...)
}

func (b *clusterBuilder) edsCluster(name string) *cluster.Cluster {
	return &cluster.Cluster{
		Name:                 name,
		AltStatName:          util.DelimitedStatsPrefix(name),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
		EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
			ServiceName: name,
			EdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
		},
		// CRITICAL: For gRPC proxyless, we need to set LbPolicy to ROUND_ROBIN
		// This is the default load balancing policy for gRPC xDS clients
		LbPolicy: cluster.Cluster_ROUND_ROBIN,
	}
}

func (b *clusterBuilder) applyDestinationRule(defaultCluster *cluster.Cluster) (subsetClusters []*cluster.Cluster) {
	if b.svc == nil || b.port == nil {
		return nil
	}
	// TODO
	return
}
