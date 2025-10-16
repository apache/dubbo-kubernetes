package controller

import (
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/network"
	v1 "k8s.io/api/core/v1"
	"sync"
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
