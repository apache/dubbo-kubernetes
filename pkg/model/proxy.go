package model

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	networkutil "github.com/apache/dubbo-kubernetes/sail/pkg/util/network"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"strconv"
)

type NodeType string

type NodeMetaProxyConfig meshconfig.ProxyConfig

type NodeMetadata struct {
	Generator          string               `json:"GENERATOR,omitempty"`
	ClusterID          cluster.ID           `json:"CLUSTER_ID,omitempty"`
	Namespace          string               `json:"NAMESPACE,omitempty"`
	StsPort            string               `json:"STS_PORT,omitempty"`
	MetadataDiscovery  *StringBool          `json:"METADATA_DISCOVERY,omitempty"`
	ProxyConfig        *NodeMetaProxyConfig `json:"PROXY_CONFIG,omitempty"`
	SailSubjectAltName []string             `json:"SAIL_SAN,omitempty"`
	XDSRootCert        string               `json:"-"`
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
	default:
		return true
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
