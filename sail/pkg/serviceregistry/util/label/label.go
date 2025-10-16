package label

import (
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	"github.com/apache/dubbo-kubernetes/pkg/network"
	"istio.io/api/label"
	"strings"
)

const (
	LabelHostname = "kubernetes.io/hostname"

	LabelTopologyZone    = "topology.kubernetes.io/zone"
	LabelTopologySubzone = "topology.dubbo.io/subzone"
	LabelTopologyRegion  = "topology.kubernetes.io/region"
)

func AugmentLabels(in labels.Instance, clusterID cluster.ID, locality, k8sNode string, networkID network.ID) labels.Instance {
	// Copy the original labels to a new map.
	out := make(labels.Instance, len(in)+6)
	for k, v := range in {
		out[k] = v
	}

	region, zone, subzone := SplitLocalityLabel(locality)
	if len(region) > 0 {
		out[LabelTopologyRegion] = region
	}
	if len(zone) > 0 {
		out[LabelTopologyZone] = zone
	}
	if len(subzone) > 0 {
		out[label.TopologySubzone.Name] = subzone
	}
	if len(clusterID) > 0 {
		out[label.TopologyCluster.Name] = clusterID.String()
	}
	if len(k8sNode) > 0 {
		out[LabelHostname] = k8sNode
	}
	// In c.Network(), we already set the network label in priority order pod labels > namespace label > mesh Network
	// We won't let proxy.Metadata.Network override the above.
	if len(networkID) > 0 && out[label.TopologyNetwork.Name] == "" {
		out[label.TopologyNetwork.Name] = networkID.String()
	}
	return out
}

func SplitLocalityLabel(locality string) (region, zone, subzone string) {
	items := strings.Split(locality, "/")
	switch len(items) {
	case 1:
		return items[0], "", ""
	case 2:
		return items[0], items[1], ""
	default:
		return items[0], items[1], items[2]
	}
}
