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

package xds

import (
	"fmt"
	"strings"
)

import (
	"github.com/pkg/errors"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

type APIVersion string

// StreamID represents a stream opened by XDS
type StreamID = int64

type ProxyId struct {
	mesh string
	name string
}

func (id *ProxyId) String() string {
	return fmt.Sprintf("%s.%s", id.mesh, id.name)
}

func (id *ProxyId) ToResourceKey() core_model.ResourceKey {
	return core_model.ResourceKey{
		Name: id.name,
		Mesh: id.mesh,
	}
}

// ServiceName is a convenience type alias to clarify the meaning of string value.
type ServiceName = string

type MeshName = string

// TagSelectorSet is a set of unique TagSelectors.
type TagSelectorSet []mesh_proto.TagSelector

// DestinationMap holds a set of selectors for all reachable Dataplanes grouped by service name.
// DestinationMap is based on ServiceName and not on the OutboundInterface because TrafficRoute can introduce new service destinations that were not included in a outbound section.
// Policies that match on outbound connections also match by service destination name and not outbound interface for the same reason.
type DestinationMap map[ServiceName]TagSelectorSet

type ExternalService struct {
	TLSEnabled               bool
	CaCert                   []byte
	ClientCert               []byte
	ClientKey                []byte
	AllowRenegotiation       bool
	SkipHostnameVerification bool
	ServerName               string
}

type Locality struct {
	Zone     string
	SubZone  string
	Priority uint32
	Weight   uint32
}

// Endpoint holds routing-related information about a single endpoint.
// It is abstracted from mesh_proto.DataplaneRecourse.Spec.Networking.Inbound
type Endpoint struct {
	Target          string
	UnixDomainPath  string
	Port            uint32
	Tags            map[string]string // clone from inbound.GetTags
	Weight          uint32
	Locality        *Locality
	ExternalService *ExternalService
}

func (e Endpoint) Address() string {
	return fmt.Sprintf("%s:%d", e.Target, e.Port)
}

// EndpointList is a list of Endpoints with convenience methods.
type EndpointList []Endpoint

// EndpointMap holds routing-related information about a set of endpoints grouped by service name.
// here key is dubbo-serviceName
type EndpointMap map[ServiceName][]Endpoint

// SocketAddressProtocol is the L4 protocol the listener should bind to
type SocketAddressProtocol int32

const (
	SocketAddressProtocolTCP SocketAddressProtocol = 0
	SocketAddressProtocolUDP SocketAddressProtocol = 1
)

// Proxy contains required data for generating XDS config that is specific to a data plane proxy.
// The data that is specific for the whole mesh should go into MeshContext.
type Proxy struct {
	Id         ProxyId
	APIVersion APIVersion
	Dataplane  *core_mesh.DataplaneResource
	Metadata   *DataplaneMetadata
	Routing    Routing
	Policies   MatchedPolicies

	// SecretsTracker allows us to track when a generator references a secret so
	// we can be sure to include only those secrets later on.
	SecretsTracker SecretsTracker

	// ZoneIngressProxy is available only when XDS is generated for ZoneIngress data plane proxy.
	ZoneIngressProxy *ZoneIngressProxy
	// RuntimeExtensions a set of extensions to add for custom extensions
	RuntimeExtensions map[string]interface{}
	// Zone the zone the proxy is in
	Zone string
}

type ServerSideTLSCertPaths struct {
	CertPath string
	KeyPath  string
}

type IdentityCertRequest interface {
	Name() string
}

type CaRequest interface {
	MeshName() []string
	Name() string
}

// SecretsTracker provides a way to ask for a secret and keeps track of which are
// used, so that they can later be generated and included in the resources.
type SecretsTracker interface {
	RequestIdentityCert() IdentityCertRequest
	RequestCa(mesh string) CaRequest
	RequestAllInOneCa() CaRequest

	UsedIdentity() bool
	UsedCas() map[string]struct{}
	UsedAllInOne() bool
}

type ExternalServiceDynamicPolicies map[ServiceName]PluginOriginatedPolicies

type MeshIngressResources struct {
	Mesh        *core_mesh.MeshResource
	EndpointMap EndpointMap
	Resources   map[core_model.ResourceType]core_model.ResourceList
}

type ZoneIngressProxy struct {
	ZoneIngressResource *core_mesh.ZoneIngressResource
	MeshResourceList    []*MeshIngressResources
}

type ClusterSelectorList struct {
	MatchInfo    mesh_proto.TrafficRoute_Http_Match
	ModifyInfo   mesh_proto.TrafficRoute_Http_Modify
	EndSelectors []ClusterSelector
}

type ServiceSelectorMap map[ServiceName][]ClusterSelectorList

type ClusterSelector struct {
	ConfigInfo TrafficRouteConfig
	TagSelect  mesh_proto.TagSelector
}

func (c *ClusterSelector) Select(l EndpointList) EndpointList {
	res := EndpointList{}
	for _, endpoint := range l {
		if c.TagSelect.Matches(endpoint.Tags) {
			res = append(res, endpoint)
		}
	}
	return res
}

func (e *ClusterSelectorList) GetMatchInfo() *mesh_proto.TrafficRoute_Http_Match {
	return &e.MatchInfo
}
func (e *ClusterSelectorList) GetModifyInfo() *mesh_proto.TrafficRoute_Http_Modify {
	return &e.ModifyInfo
}

type Routing struct {
	OutboundSelector ServiceSelectorMap
	OutboundTargets  EndpointMap
	// ExternalServiceOutboundTargets contains endpoint map for direct access of external services (without egress)
	// Since we take into account TrafficPermission to exclude external services from the map,
	// it is specific for each data plane proxy.
	ExternalServiceOutboundTargets EndpointMap
}

func (s TagSelectorSet) Add(new mesh_proto.TagSelector) TagSelectorSet {
	for _, old := range s {
		if new.Equal(old) {
			return s
		}
	}
	return append(s, new)
}

func (s TagSelectorSet) Matches(tags map[string]string) bool {
	for _, selector := range s {
		if selector.Matches(tags) {
			return true
		}
	}
	return false
}

func (e Endpoint) IsExternalService() bool {
	return e.ExternalService != nil
}

func (e Endpoint) LocalityString() string {
	if e.Locality == nil {
		return ""
	}
	return fmt.Sprintf("%s:%s", e.Locality.Zone, e.Locality.SubZone)
}

func (e Endpoint) HasLocality() bool {
	return e.Locality != nil
}

// ContainsTags returns 'true' if for every key presented both in 'tags' and 'Endpoint#Tags'
// values are equal
func (e Endpoint) ContainsTags(tags map[string]string) bool {
	for otherKey, otherValue := range tags {
		endpointValue, ok := e.Tags[otherKey]
		if !ok || otherValue != endpointValue {
			return false
		}
	}
	return true
}

func (l EndpointList) Filter(selector mesh_proto.TagSelector) EndpointList {
	var endpoints EndpointList
	for _, endpoint := range l {
		if selector.Matches(endpoint.Tags) {
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

// if false endpoint should be accessed through zoneIngress of other zone
func (e Endpoint) IsReachableFromZone(localZone string) bool {
	return e.Locality == nil || e.Locality.Zone == "" || e.Locality.Zone == localZone
}

func BuildProxyId(mesh, name string) *ProxyId {
	return &ProxyId{
		name: name,
		mesh: mesh,
	}
}

func ParseProxyIdFromString(id string) (*ProxyId, error) {
	if id == "" {
		return nil, errors.Errorf("Envoy ID must not be nil")
	}
	parts := strings.SplitN(id, ".", 2)
	mesh := parts[0]
	// when proxy is an ingress mesh is empty
	if len(parts) < 2 {
		return nil, errors.New("the name should be provided after the dot")
	}
	name := parts[1]
	if name == "" {
		return nil, errors.New("name must not be empty")
	}
	return &ProxyId{
		mesh: mesh,
		name: name,
	}, nil
}

func FromResourceKey(key core_model.ResourceKey) ProxyId {
	return ProxyId{
		mesh: key.Mesh,
		name: key.Name,
	}
}
