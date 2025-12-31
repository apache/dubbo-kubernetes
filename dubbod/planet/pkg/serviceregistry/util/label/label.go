//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package label

import (
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	"github.com/apache/dubbo-kubernetes/pkg/network"
)

const (
	LabelTopologyCluster = "topology.dubbo.apache.org/cluster"
	LabelTopologyNetwork = "topology.dubbo.apache.org/network"
)

func AugmentLabels(in labels.Instance, clusterID cluster.ID, locality, k8sNode string, networkID network.ID) labels.Instance {
	// Copy the original labels to a new map.
	out := make(labels.Instance, len(in)+2)
	for k, v := range in {
		out[k] = v
	}

	// In proxyless mesh, locality is not used, so we skip region/zone/subzone labels
	if len(clusterID) > 0 {
		out[LabelTopologyCluster] = clusterID.String()
	}
	// In c.Network(), we already set the network label in priority order pod labels > namespace label
	// We won't let proxy.Metadata.Network override the above.
	if len(networkID) > 0 && out[LabelTopologyNetwork] == "" {
		out[LabelTopologyNetwork] = networkID.String()
	}
	return out
}
