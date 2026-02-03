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

package model

import (
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	networkutil "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/util/network"

	meshv1alpha1 "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/features"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/maps"
	pm "github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	serviceNodeSeparator = "~"
)

const (
	IPv4 = pm.IPv4
	IPv6 = pm.IPv6
	Dual = pm.Dual
)

type XdsResourceGenerator interface {
	Generate(proxy *Proxy, w *WatchedResource, req *PushRequest) (Resources, XdsLogDetails, error)
}

type XdsDeltaResourceGenerator interface {
	XdsResourceGenerator
	GenerateDeltas(proxy *Proxy, req *PushRequest, w *WatchedResource) (Resources, DeletedResources, XdsLogDetails, bool, error)
}

type (
	Node                  = pm.Node
	NodeMetadata          = pm.NodeMetadata
	BootstrapNodeMetadata = pm.BootstrapNodeMetadata
	NodeType              = pm.NodeType
	IPMode                = pm.IPMode
)

const (
	Proxyless = pm.Proxyless
	Router    = pm.Router
)

type Watcher = meshwatcher.WatcherCollection

type WatchedResource = xds.WatchedResource

type Environment struct {
	ServiceDiscovery
	Watcher
	ConfigStore
	mutex                sync.RWMutex
	pushContext          *PushContext
	clusterLocalServices ClusterLocalProvider
	DomainSuffix         string
	EndpointIndex        *EndpointIndex
	Cache                XdsCache
	GatewayAPIController GatewayController
}

type GatewayController interface {
	ConfigStoreController
	Reconcile(ctx *PushContext)
}

func NewEnvironment() *Environment {
	var cache XdsCache
	if features.EnableXDSCaching {
		cache = NewXdsCache()
	} else {
		cache = DisabledCache{}
	}
	return &Environment{
		pushContext:   NewPushContext(),
		Cache:         cache,
		EndpointIndex: NewEndpointIndex(cache),
	}
}

func NewEndpointIndex(cache XdsCache) *EndpointIndex {
	return &EndpointIndex{
		shardsBySvc: make(map[string]map[string]*EndpointShards),
		cache:       cache,
	}
}

type XdsLogDetails struct {
	Incremental    bool
	AdditionalInfo string
}

var DefaultXdsLogDetails = XdsLogDetails{}

type Resources = []*discovery.Resource

type DeletedResources = []string

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

func (e *Environment) Mesh() *meshv1alpha1.MeshGlobalConfig {
	if e != nil && e.Watcher != nil {
		return e.Watcher.Mesh()
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

func (e *Environment) GetProxyConfigOrDefault(ns string, labels, annotations map[string]string, meshGlobalConfig *meshv1alpha1.MeshGlobalConfig) *meshv1alpha1.ProxyConfig {
	return mesh.DefaultProxyConfig()
}

func (e *Environment) ClusterLocal() ClusterLocalProvider {
	return e.clusterLocalServices
}

func (e *Environment) Init() {
	if len(e.DomainSuffix) == 0 {
		e.DomainSuffix = constants.DefaultClusterLocalDomain
	}

	e.clusterLocalServices = NewClusterLocalProvider(e)
}

type Proxy struct {
	sync.RWMutex
	XdsResourceGenerator XdsResourceGenerator
	LastPushContext      *PushContext
	LastPushTime         time.Time
	Type                 NodeType
	WatchedResources     map[string]*WatchedResource
	ID                   string
	DNSDomain            string
	Metadata             *NodeMetadata
	IPAddresses          []string
	XdsNode              *core.Node
	ConfigNamespace      string
	ServiceTargets       []ServiceTarget
	ipMode               IPMode
	GlobalUnicastIP      string
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

func (node *Proxy) IsRouter() bool {
	return node != nil && node.Type == Router
}

func (node *Proxy) IsProxyless() bool {
	return node != nil && node.Type == Proxyless
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

func (node *Proxy) DiscoverIPMode() {
	node.ipMode = pm.DiscoverIPMode(node.IPAddresses)
	node.GlobalUnicastIP = networkutil.GlobalUnicastIP(node.IPAddresses)
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

func (node *Proxy) ShouldUpdateServiceTargets(updates sets.Set[ConfigKey]) bool {
	for config := range updates {
		// if config.Kind == kind.ServiceEntry || config.Namespace == node.Metadata.Namespace {
		// 	return true
		// }
		if config.Namespace == node.Metadata.Namespace {
			return true
		}
	}

	return false
}

func (node *Proxy) SetServiceTargets(serviceDiscovery ServiceDiscovery) {
	instances := serviceDiscovery.GetProxyServiceTargets(node)

	sort.SliceStable(instances, func(i, j int) bool {
		if instances[i].Service != nil && instances[j].Service != nil {
			if !instances[i].Service.CreationTime.Equal(instances[j].Service.CreationTime) {
				return instances[i].Service.CreationTime.Before(instances[j].Service.CreationTime)
			}
			// Additionally, sort by hostname just in case services created automatically at the same second.
			return instances[i].Service.Hostname < instances[j].Service.Hostname
		}
		return true
	})

	node.ServiceTargets = instances
}

func (node *Proxy) ShallowCloneWatchedResources() map[string]*WatchedResource {
	node.RLock()
	defer node.RUnlock()
	return maps.Clone(node.WatchedResources)
}

func ParseMetadata(metadata *structpb.Struct) (*NodeMetadata, error) {
	if metadata == nil {
		return &NodeMetadata{}, nil
	}

	bootstrapNodeMeta, err := ParseBootstrapNodeMetadata(metadata)
	if err != nil {
		return nil, err
	}
	return &bootstrapNodeMeta.NodeMetadata, nil
}

func ParseBootstrapNodeMetadata(metadata *structpb.Struct) (*BootstrapNodeMetadata, error) {
	if metadata == nil {
		return &BootstrapNodeMetadata{}, nil
	}

	b, err := protomarshal.MarshalProtoNames(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to read node metadata %v: %v", metadata, err)
	}
	meta := &BootstrapNodeMetadata{}
	if err := json.Unmarshal(b, meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node metadata (%v): %v", string(b), err)
	}
	return meta, nil
}

func ParseServiceNodeWithMetadata(nodeID string, metadata *NodeMetadata) (*Proxy, error) {
	parts := strings.Split(nodeID, serviceNodeSeparator)
	out := &Proxy{
		Metadata: metadata,
	}
	if len(parts) != 4 {
		return out, fmt.Errorf("missing parts in the service node %q (expected 4 parts, got %d)", nodeID, len(parts))
	}

	// Validate node type
	if !pm.IsApplicationNodeType(NodeType(parts[0])) {
		return out, fmt.Errorf("invalid node type %q in the service node %q", parts[0], nodeID)
	}
	out.Type = NodeType(parts[0])

	// Extract IP address from parts[1] (format: type~ip~id~domain)
	// Validate and set IP address
	if len(parts[1]) > 0 {
		ip := net.ParseIP(parts[1])
		if ip != nil {
			out.IPAddresses = []string{parts[1]}
		}
	}

	if len(out.IPAddresses) == 0 {
		out.IPAddresses = []string{}
	}

	out.ID = parts[2]
	out.DNSDomain = parts[3]

	// Validate that ID is not empty - this is critical for proxyless gRPC
	if len(out.ID) == 0 {
		return out, fmt.Errorf("node ID is empty in service node %q (parts[2] is empty)", nodeID)
	}

	return out, nil
}

func GetProxyConfigNamespace(proxy *Proxy) string {
	if proxy == nil {
		return ""
	}

	if len(proxy.Metadata.Namespace) > 0 {
		return proxy.Metadata.Namespace
	}

	parts := strings.Split(proxy.DNSDomain, ".")
	if len(parts) > 1 { // k8s will have namespace.<domain>
		return parts[0]
	}

	return ""
}
