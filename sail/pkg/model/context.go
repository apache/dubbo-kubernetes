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

package model

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	pm "github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"net"
	"strconv"
	"sync"
	"time"
)

type (
	NodeMetadata = pm.NodeMetadata
)
type Watcher = meshwatcher.WatcherCollection

type WatchedResource = xds.WatchedResource

type Environment struct {
	ServiceDiscovery
	Watcher
	ConfigStore
	mutex                sync.RWMutex
	pushContext          *PushContext
	NetworksWatcher      mesh.NetworksWatcher
	NetworkManager       *NetworkManager
	clusterLocalServices ClusterLocalProvider
	DomainSuffix         string
	EndpointIndex        *EndpointIndex
}

func NewEnvironment() *Environment {
	return &Environment{
		pushContext:   NewPushContext(),
		EndpointIndex: NewEndpointIndex(),
	}
}

func NewEndpointIndex() *EndpointIndex {
	return &EndpointIndex{
		shardsBySvc: make(map[string]map[string]*EndpointShards),
	}
}

var _ mesh.Holder = &Environment{}

func (e *Environment) PushContext() *PushContext {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.pushContext
}

func (e *Environment) SetPushContext(pc *PushContext) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.pushContext = pc
}

func (e *Environment) Mesh() *meshconfig.MeshConfig {
	if e != nil && e.Watcher != nil {
		return e.Watcher.Mesh()
	}
	return nil
}

func (e *Environment) MeshNetworks() *meshconfig.MeshNetworks {
	if e != nil && e.NetworksWatcher != nil {
		return e.NetworksWatcher.Networks()
	}
	return nil
}

func (e *Environment) AddMeshHandler(h func()) {
	if e != nil && e.Watcher != nil {
		e.Watcher.AddMeshHandler(h)
	}
}

func (e *Environment) GetDiscoveryAddress() (host.Name, string, error) {
	proxyConfig := mesh.DefaultProxyConfig()
	if e.Mesh().DefaultConfig != nil {
		proxyConfig = e.Mesh().DefaultConfig
	}
	hostname, port, err := net.SplitHostPort(proxyConfig.DiscoveryAddress)
	if err != nil {
		return "", "", fmt.Errorf("invalid Dubbod Address: %s, %v", proxyConfig.DiscoveryAddress, err)
	}
	if _, err := strconv.Atoi(port); err != nil {
		return "", "", fmt.Errorf("invalid Dubbod Port: %s, %s, %v", port, proxyConfig.DiscoveryAddress, err)
	}
	return host.Name(hostname), port, nil
}

func (e *Environment) GetProxyConfigOrDefault(ns string, labels, annotations map[string]string, meshConfig *meshconfig.MeshConfig) *meshconfig.ProxyConfig {
	return mesh.DefaultProxyConfig()
}

func (e *Environment) ClusterLocal() ClusterLocalProvider {
	return e.clusterLocalServices
}

func (e *Environment) Init() {
	// Use a default DomainSuffix, if none was provided.
	if len(e.DomainSuffix) == 0 {
		e.DomainSuffix = constants.DefaultClusterLocalDomain
	}

	e.clusterLocalServices = NewClusterLocalProvider(e)
}

func (e *Environment) InitNetworksManager(updater XDSUpdater) (err error) {
	e.NetworkManager, err = NewNetworkManager(e, updater)
	return
}

type Proxy struct {
	sync.RWMutex
	XdsResourceGenerator XdsResourceGenerator
	LastPushContext      *PushContext
	LastPushTime         time.Time
	WatchedResources     map[string]*WatchedResource
	ID                   string
	Metadata             *NodeMetadata
	IPAddresses          []string
}

func (node *Proxy) GetWatchedResource(typeURL string) *WatchedResource {
	node.RLock()
	defer node.RUnlock()

	return node.WatchedResources[typeURL]
}

func (node *Proxy) DeleteWatchedResource(typeURL string) {
	node.Lock()
	defer node.Unlock()

	delete(node.WatchedResources, typeURL)
}

func (node *Proxy) NewWatchedResource(typeURL string, names []string) {
	node.Lock()
	defer node.Unlock()

	node.WatchedResources[typeURL] = &WatchedResource{TypeUrl: typeURL, ResourceNames: sets.New(names...)}
}

func (node *Proxy) GetID() string {
	if node == nil {
		return ""
	}
	return node.ID
}

func (node *Proxy) UpdateWatchedResource(typeURL string, updateFn func(*WatchedResource) *WatchedResource) {
	node.Lock()
	defer node.Unlock()
	r := node.WatchedResources[typeURL]
	r = updateFn(r)
	if r != nil {
		node.WatchedResources[typeURL] = r
	} else {
		delete(node.WatchedResources, typeURL)
	}
}

func (node *Proxy) IsProxylessGrpc() bool {
	return node.Metadata != nil && node.Metadata.Generator == "grpc"
}

type XdsLogDetails struct {
	Incremental    bool
	AdditionalInfo string
}

type Resources = []*discovery.Resource

type XdsResourceGenerator interface {
	// Generate generates the Sotw resources for Xds.
	Generate(proxy *Proxy, w *WatchedResource, req *PushRequest) (Resources, XdsLogDetails, error)
}
