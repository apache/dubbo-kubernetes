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

// ServiceName is a convenience type alias to clarify the meaning of string value.
type ServiceName = string

type MeshName = string

// Proxy contains required data for generating XDS config that is specific to a data plane proxy.
// The data that is specific for the whole mesh should go into MeshContext.
type Proxy struct {
	Id         ProxyId
	APIVersion APIVersion
	Zone       string
}

type ExternalService struct {
	ServerName string
}

type Locality struct {
	Zone     string
	SubZone  string
	Priority uint32
	Weight   uint32
}

type Endpoint struct {
	Target          string
	UnixDomainPath  string
	Port            uint32
	Tags            map[string]string
	Weight          uint32
	Locality        *Locality
	ExternalService *ExternalService
}

func (e Endpoint) HasLocality() bool {
	return e.Locality != nil
}

func (e Endpoint) LocalityString() string {
	if e.Locality == nil {
		return ""
	}
	return fmt.Sprintf("%s:%s", e.Locality.Zone, e.Locality.SubZone)
}

func (e Endpoint) Address() string {
	return fmt.Sprintf("%s:%d", e.Target, e.Port)
}

type EndpointList []Endpoint

type EndpointMap map[ServiceName][]Endpoint

func FromResourceKey(key core_model.ResourceKey) ProxyId {
	return ProxyId{
		mesh: key.Mesh,
		name: key.Name,
	}
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
