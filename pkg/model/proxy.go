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
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	networkutil "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/util/network"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	meshconfig "istio.io/api/mesh/v1alpha1"
)

type NodeType string

const (
	Proxyless NodeType = "proxyless"
)

type NodeMetaProxyConfig meshconfig.ProxyConfig

// MarshalJSON customizes JSON serialization to handle oneof ClusterName field
func (n *NodeMetaProxyConfig) MarshalJSON() ([]byte, error) {
	// Convert to base type
	base := (*meshconfig.ProxyConfig)(n)

	// Use protomarshal to handle protobuf message serialization correctly
	return protomarshal.Marshal(base)
}

// UnmarshalJSON customizes JSON deserialization to handle oneof ClusterName field
func (n *NodeMetaProxyConfig) UnmarshalJSON(data []byte) error {
	if n == nil {
		return fmt.Errorf("cannot unmarshal into nil NodeMetaProxyConfig")
	}

	// Parse JSON into a map to handle ClusterName conversion
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Convert ClusterName from string to oneof object if needed
	if clusterNameStr, ok := raw["ClusterName"].(string); ok {
		// Convert string "dubbo-proxy" to oneof object {"ServiceCluster": "dubbo-proxy"}
		raw["ClusterName"] = map[string]any{"ServiceCluster": clusterNameStr}
	}

	// Re-marshal to JSON and then use protomarshal to unmarshal into protobuf message
	convertedData, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	// Use protomarshal to handle protobuf message deserialization correctly
	// Convert to base type pointer for unmarshaling
	base := (*meshconfig.ProxyConfig)(n)
	if err := protomarshal.Unmarshal(convertedData, base); err != nil {
		return err
	}

	return nil
}

type NodeMetadata struct {
	Generator            string               `json:"GENERATOR,omitempty"`
	ClusterID            cluster.ID           `json:"CLUSTER_ID,omitempty"`
	Namespace            string               `json:"NAMESPACE,omitempty"`
	StsPort              string               `json:"STS_PORT,omitempty"`
	MetadataDiscovery    *StringBool          `json:"METADATA_DISCOVERY,omitempty"`
	ProxyConfig          *NodeMetaProxyConfig `json:"PROXY_CONFIG,omitempty"`
	PlanetSubjectAltName []string             `json:"PLANET_SAN,omitempty"`
	XDSRootCert          string               `json:"-"`
}
type Node struct {
	// ID of the Envoy node
	ID string
	// Metadata is the typed node metadata
	Metadata *BootstrapNodeMetadata
	// RawMetadata is the untyped node metadata
	RawMetadata map[string]any
	// Locality from Envoy bootstrap
	Locality *core.Locality
}
type BootstrapNodeMetadata struct {
	NodeMetadata
}

func IsApplicationNodeType(nType NodeType) bool {
	switch nType {
	case Proxyless:
		return true
	default:
		return false
	}
}

type IPMode int

const (
	IPv4 IPMode = iota + 1
	IPv6
	Dual
)

func DiscoverIPMode(addrs []string) IPMode {
	if networkutil.AllIPv4(addrs) {
		return IPv4
	} else if networkutil.AllIPv6(addrs) {
		return IPv6
	}
	return Dual
}

type StringBool bool

func (s StringBool) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%t"`, s)), nil
}

func (s *StringBool) UnmarshalJSON(data []byte) error {
	pls, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	b, err := strconv.ParseBool(pls)
	if err != nil {
		return err
	}
	*s = StringBool(b)
	return nil
}
