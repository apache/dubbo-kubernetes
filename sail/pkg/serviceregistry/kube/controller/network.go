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

package controller

import (
	"sync"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/network"
	v1 "k8s.io/api/core/v1"
)

type networkManager struct {
	sync.RWMutex
	clusterID             cluster.ID
	meshNetworksWatcher   mesh.NetworksWatcher
	network               network.ID
	networkFromMeshConfig network.ID
}

func initNetworkManager(c *Controller, options Options) *networkManager {
	n := &networkManager{
		clusterID:             options.ClusterID,
		meshNetworksWatcher:   options.MeshNetworksWatcher,
		network:               "",
		networkFromMeshConfig: "",
	}
	return n
}

func (n *networkManager) networkFromSystemNamespace() network.ID {
	n.RLock()
	defer n.RUnlock()
	return n.network
}

func (n *networkManager) networkFromMeshNetworks(endpointIP string) network.ID {
	n.RLock()
	defer n.RUnlock()
	if n.networkFromMeshConfig != "" {
		return n.networkFromMeshConfig
	}

	return ""
}

func (n *networkManager) HasSynced() bool {
	return true
}

func (n *networkManager) setNetworkFromNamespace(ns *v1.Namespace) bool {
	nw := ns.Labels["topology.dubbo.io/network"]
	n.Lock()
	defer n.Unlock()
	oldDefaultNetwork := n.network
	n.network = network.ID(nw)
	return oldDefaultNetwork != n.network
}
