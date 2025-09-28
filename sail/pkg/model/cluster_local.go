package model

import "github.com/apache/dubbo-kubernetes/pkg/config/host"

type ClusterLocalHosts struct {
	specific map[host.Name]bool
	wildcard map[host.Name]bool
}

type ClusterLocalProvider interface {
	// GetClusterLocalHosts returns the list of cluster-local hosts, sorted in
	// ascending order. The caller must not modify the returned list.
	GetClusterLocalHosts() ClusterLocalHosts
}
