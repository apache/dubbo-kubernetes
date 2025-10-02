package model

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"k8s.io/klog/v2"
	"strings"
	"sync"
)

var (
	defaultClusterLocalNamespaces = []string{"kube-system"}
	defaultClusterLocalServices   = []string{"kubernetes.default.svc"}
)

type ClusterLocalHosts struct {
	specific map[host.Name]bool
	wildcard map[host.Name]bool
}

type ClusterLocalProvider interface {
	GetClusterLocalHosts() ClusterLocalHosts
}

type clusterLocalProvider struct {
	mutex sync.RWMutex
	hosts ClusterLocalHosts
}

func NewClusterLocalProvider(e *Environment) ClusterLocalProvider {
	c := &clusterLocalProvider{}

	// Register a handler to update the environment when the mesh config is updated.
	e.AddMeshHandler(func() {
		c.onMeshUpdated(e)
	})

	// Update the cluster-local hosts now.
	c.onMeshUpdated(e)
	return c
}

func (c *clusterLocalProvider) GetClusterLocalHosts() ClusterLocalHosts {
	c.mutex.RLock()
	out := c.hosts
	c.mutex.RUnlock()
	return out
}

func (c *clusterLocalProvider) onMeshUpdated(e *Environment) {
	// Create the default list of cluster-local hosts.
	domainSuffix := e.DomainSuffix
	defaultClusterLocalHosts := make([]host.Name, 0)
	for _, n := range defaultClusterLocalNamespaces {
		defaultClusterLocalHosts = append(defaultClusterLocalHosts, host.Name("*."+n+".svc."+domainSuffix))
	}
	for _, s := range defaultClusterLocalServices {
		defaultClusterLocalHosts = append(defaultClusterLocalHosts, host.Name(s+"."+domainSuffix))
	}

	if discoveryHost, _, err := e.GetDiscoveryAddress(); err != nil {
		klog.Errorf("failed to make discoveryAddress cluster-local: %v", err)
	} else {
		if !strings.HasSuffix(string(discoveryHost), domainSuffix) {
			discoveryHost += host.Name("." + domainSuffix)
		}
		defaultClusterLocalHosts = append(defaultClusterLocalHosts, discoveryHost)
	}

	// Collect the cluster-local hosts.
	hosts := ClusterLocalHosts{
		specific: make(map[host.Name]bool),
		wildcard: make(map[host.Name]bool),
	}

	for _, serviceSettings := range e.Mesh().ServiceSettings {
		isClusterLocal := serviceSettings.GetSettings().GetClusterLocal()
		for _, h := range serviceSettings.GetHosts() {
			// If clusterLocal false, check to see if we should remove a default clusterLocal host.
			if !isClusterLocal {
				for i, defaultClusterLocalHost := range defaultClusterLocalHosts {
					if len(defaultClusterLocalHost) > 0 {
						if h == string(defaultClusterLocalHost) ||
							(defaultClusterLocalHost.IsWildCarded() &&
								strings.HasSuffix(h, string(defaultClusterLocalHost[1:]))) {
							// This default was explicitly overridden, so remove it.
							defaultClusterLocalHosts[i] = ""
						}
					}
				}
			}
		}

		// Add hosts with their clusterLocal setting to sets.
		for _, h := range serviceSettings.GetHosts() {
			hostname := host.Name(h)
			if hostname.IsWildCarded() {
				hosts.wildcard[hostname] = isClusterLocal
			} else {
				hosts.specific[hostname] = isClusterLocal
			}
		}
	}

	// Add any remaining defaults to the end of the list.
	for _, defaultClusterLocalHost := range defaultClusterLocalHosts {
		if len(defaultClusterLocalHost) > 0 {
			if defaultClusterLocalHost.IsWildCarded() {
				hosts.wildcard[defaultClusterLocalHost] = true
			} else {
				hosts.specific[defaultClusterLocalHost] = true
			}
		}
	}

	c.mutex.Lock()
	c.hosts = hosts
	c.mutex.Unlock()
}
