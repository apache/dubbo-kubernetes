package model

import (
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/pkg/config/visibility"
	"github.com/apache/dubbo-kubernetes/pkg/maps"
	"github.com/apache/dubbo-kubernetes/pkg/network"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	"github.com/google/go-cmp/cmp"
	"istio.io/api/annotation"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"strings"
	"sync"
	"time"
)

type Resolution int

const (
	// ClientSideLB implies that the proxy will decide the endpoint from its local lb pool
	ClientSideLB Resolution = iota
	// DNSLB implies that the proxy will resolve a DNS address and forward to the resolved address
	DNSLB
	// Passthrough implies that the proxy should forward traffic to the destination IP requested by the caller
	Passthrough
	// DNSRoundRobinLB implies that the proxy will resolve a DNS address and forward to the resolved address
	DNSRoundRobinLB
	// Alias defines a Service that is an alias for another.
	Alias
)

type EndpointDiscoverabilityPolicy interface {
	String() string
}

type endpointDiscoverabilityPolicyImpl struct {
	name string
	f    func(*DubboEndpoint, *Proxy) bool
}

func (p *endpointDiscoverabilityPolicyImpl) String() string {
	return p.name
}

var endpointDiscoverabilityPolicyImplCmpOpt = cmp.Comparer(func(x, y endpointDiscoverabilityPolicyImpl) bool {
	return x.String() == y.String()
})

func (p *endpointDiscoverabilityPolicyImpl) CmpOpts() []cmp.Option {
	return []cmp.Option{endpointDiscoverabilityPolicyImplCmpOpt}
}

type HealthStatus int32

const (
	Healthy     HealthStatus = 1
	UnHealthy   HealthStatus = 2
	Draining    HealthStatus = 3
	Terminating HealthStatus = 4
)

type DubboEndpoint struct {
	ServiceAccount         string
	Addresses              []string
	ServicePortName        string
	Labels                 labels.Instance
	HealthStatus           HealthStatus
	SendUnhealthyEndpoints bool
	DiscoverabilityPolicy  EndpointDiscoverabilityPolicy `json:"-"`
	LegacyClusterPortKey   int
	EndpointPort           uint32
	WorkloadName           string
	Network                network.ID
	Namespace              string
	// Specifies the hostname of the Pod, empty for vm workload.
	HostName string
	// If specified, the fully qualified Pod hostname will be "<hostname>.<subdomain>.<pod namespace>.svc.<cluster domain>".
	SubDomain string
	NodeName  string
}

func (ep *DubboEndpoint) FirstAddressOrNil() string {
	if ep == nil || len(ep.Addresses) == 0 {
		return ""
	}
	return ep.Addresses[0]
}

func (ep *DubboEndpoint) Key() string {
	return ep.FirstAddressOrNil() + "/" + ep.WorkloadName + "/" + ep.ServicePortName
}

func (ep *DubboEndpoint) Equals(other *DubboEndpoint) bool {
	if ep == nil {
		return other == nil
	}
	if other == nil {
		return ep == nil
	}

	// Check things we can directly compare...
	eq := ep.ServicePortName == other.ServicePortName &&
		ep.ServiceAccount == other.ServiceAccount &&
		ep.WorkloadName == other.WorkloadName
	if !eq {
		return false
	}

	// check everything else
	if !slices.EqualUnordered(ep.Addresses, other.Addresses) {
		return false
	}
	if !maps.Equal(ep.Labels, other.Labels) {
		return false
	}

	// Compare discoverability by name
	var epp string
	if ep.DiscoverabilityPolicy != nil {
		epp = ep.DiscoverabilityPolicy.String()
	}
	var op string
	if other.DiscoverabilityPolicy != nil {
		op = other.DiscoverabilityPolicy.String()
	}
	if epp != op {
		return false
	}

	return true
}

type NamespacedHostname struct {
	Hostname  host.Name
	Namespace string
}

type ServiceAttributes struct {
	Labels                   map[string]string
	LabelSelectors           map[string]string
	ExportTo                 sets.Set[visibility.Instance]
	ClusterExternalAddresses *AddressMap
	ClusterExternalPorts     map[cluster.ID]map[uint32]uint32
	Aliases                  []NamespacedHostname
	PassthroughTargetPorts   map[uint32]uint32
	// Name is "destination.service.name" attribute
	Name string
	// Namespace is "destination.service.namespace" attribute
	Namespace       string
	ServiceRegistry provider.ID
	K8sAttributes
}

type K8sAttributes struct {
	// Type holds the value of the corev1.Type of the Kubernetes service
	// spec.Type
	Type string

	// spec.ExternalName
	ExternalName string

	// NodeLocal means the proxy will only forward traffic to node local endpoints
	// spec.InternalTrafficPolicy == Local
	NodeLocal bool

	// TrafficDistribution determines the service-level traffic distribution.
	// This may be overridden by locality load balancing settings.
	TrafficDistribution TrafficDistribution

	// ObjectName is the object name of the underlying object. This may differ from the Service.Attributes.Name for legacy semantics.
	ObjectName string

	// spec.PublishNotReadyAddresses
	PublishNotReadyAddresses bool
}

type TrafficDistribution int

const (
	// TrafficDistributionAny allows any destination
	TrafficDistributionAny TrafficDistribution = iota
	// TrafficDistributionPreferPreferSameZone prefers traffic in same zone, failing over to same region and then network.
	TrafficDistributionPreferSameZone
	// TrafficDistributionPreferNode prefers traffic in same node, failing over to same subzone, then zone, region, and network.
	TrafficDistributionPreferSameNode
)

func GetTrafficDistribution(specValue *string, annotations map[string]string) TrafficDistribution {
	if specValue != nil {
		switch *specValue {
		case corev1.ServiceTrafficDistributionPreferSameZone, corev1.ServiceTrafficDistributionPreferClose:
			return TrafficDistributionPreferSameZone
		case corev1.ServiceTrafficDistributionPreferSameNode:
			return TrafficDistributionPreferSameNode
		}
	}
	// The TrafficDistribution field is quite new, so we allow a legacy annotation option as well
	// This also has some custom types
	trafficDistributionAnnotationValue := strings.ToLower(annotations[annotation.NetworkingTrafficDistribution.Name])
	switch trafficDistributionAnnotationValue {
	case strings.ToLower(corev1.ServiceTrafficDistributionPreferClose), strings.ToLower(corev1.ServiceTrafficDistributionPreferSameZone):
		return TrafficDistributionPreferSameZone
	case strings.ToLower(corev1.ServiceTrafficDistributionPreferSameNode):
		return TrafficDistributionPreferSameNode
	default:
		if trafficDistributionAnnotationValue != "" {
			klog.Warningf("Unknown traffic distribution annotation, defaulting to any")
		}
		return TrafficDistributionAny
	}
}

type AddressMap struct {
	Addresses map[cluster.ID][]string

	// NOTE: The copystructure library is not able to copy unexported fields, so the mutex will not be copied.
	mutex sync.RWMutex
}

func (m *AddressMap) DeepCopy() *AddressMap {
	if m == nil {
		return nil
	}
	return &AddressMap{
		Addresses: m.GetAddresses(),
	}
}

type Service struct {
	Attributes               ServiceAttributes
	Hostname                 host.Name  `json:"hostname"`
	Ports                    PortList   `json:"ports,omitempty"`
	ServiceAccounts          []string   `json:"serviceAccounts,omitempty"`
	ClusterVIPs              AddressMap `json:"clusterVIPs,omitempty"`
	CreationTime             time.Time  `json:"creationTime,omitempty"`
	DefaultAddress           string     `json:"defaultAddress,omitempty"`
	ResourceVersion          string
	Resolution               Resolution
	AutoAllocatedIPv4Address string `json:"autoAllocatedIPv4Address,omitempty"`
	AutoAllocatedIPv6Address string `json:"autoAllocatedIPv6Address,omitempty"`
	MeshExternal             bool
}

func (s *Service) DeepCopy() *Service {
	// nolint: govet
	out := *s
	out.Attributes = s.Attributes.DeepCopy()
	if s.Ports != nil {
		out.Ports = make(PortList, len(s.Ports))
		for i, port := range s.Ports {
			if port != nil {
				out.Ports[i] = &Port{
					Name:     port.Name,
					Port:     port.Port,
					Protocol: port.Protocol,
				}
			} else {
				out.Ports[i] = nil
			}
		}
	}

	out.ServiceAccounts = slices.Clone(s.ServiceAccounts)
	out.ClusterVIPs = *s.ClusterVIPs.DeepCopy()
	return &out
}

func (s *Service) Key() string {
	if s == nil {
		return ""
	}

	return s.Attributes.Namespace + "/" + string(s.Hostname)
}

type Port struct {
	Name     string            `json:"name,omitempty"`
	Port     int               `json:"port"`
	Protocol protocol.Instance `json:"protocol,omitempty"`
}

type PortList []*Port

func (p *Port) Equals(other *Port) bool {
	if p == nil {
		return other == nil
	}
	if other == nil {
		return p == nil
	}
	return p.Name == other.Name && p.Port == other.Port && p.Protocol == other.Protocol
}

func (ports PortList) Equals(other PortList) bool {
	return slices.EqualFunc(ports, other, func(a, b *Port) bool {
		return a.Equals(b)
	})
}

type ServiceDiscovery interface {
	Services() []*Service
	GetService(hostname host.Name) *Service
}

func (s *ServiceAttributes) DeepCopy() ServiceAttributes {
	// AddressMap contains a mutex, which is safe to copy in this case.
	// nolint: govet
	out := *s

	out.Labels = maps.Clone(s.Labels)
	if s.ExportTo != nil {
		out.ExportTo = s.ExportTo.Copy()
	}

	out.LabelSelectors = maps.Clone(s.LabelSelectors)
	out.ClusterExternalAddresses = s.ClusterExternalAddresses.DeepCopy()

	if s.ClusterExternalPorts != nil {
		out.ClusterExternalPorts = make(map[cluster.ID]map[uint32]uint32, len(s.ClusterExternalPorts))
		for k, m := range s.ClusterExternalPorts {
			out.ClusterExternalPorts[k] = maps.Clone(m)
		}
	}

	out.Aliases = slices.Clone(s.Aliases)
	out.PassthroughTargetPorts = maps.Clone(out.PassthroughTargetPorts)

	// AddressMap contains a mutex, which is safe to return a copy in this case.
	// nolint: govet
	return out
}

func (s *Service) Equals(other *Service) bool {
	if s == nil {
		return other == nil
	}
	if other == nil {
		return s == nil
	}

	if !s.Attributes.Equals(&other.Attributes) {
		return false
	}

	if !s.Ports.Equals(other.Ports) {
		return false
	}
	if !slices.Equal(s.ServiceAccounts, other.ServiceAccounts) {
		return false
	}

	if len(s.ClusterVIPs.Addresses) != len(other.ClusterVIPs.Addresses) {
		return false
	}
	for k, v1 := range s.ClusterVIPs.Addresses {
		if v2, ok := other.ClusterVIPs.Addresses[k]; !ok || !slices.Equal(v1, v2) {
			return false
		}
	}

	return s.DefaultAddress == other.DefaultAddress && s.AutoAllocatedIPv4Address == other.AutoAllocatedIPv4Address &&
		s.AutoAllocatedIPv6Address == other.AutoAllocatedIPv6Address && s.Hostname == other.Hostname &&
		s.Resolution == other.Resolution && s.MeshExternal == other.MeshExternal
}

func (s *ServiceAttributes) Equals(other *ServiceAttributes) bool {
	if s == nil {
		return other == nil
	}
	if other == nil {
		return s == nil
	}

	if !maps.Equal(s.Labels, other.Labels) {
		return false
	}

	if !maps.Equal(s.LabelSelectors, other.LabelSelectors) {
		return false
	}

	if !maps.Equal(s.ExportTo, other.ExportTo) {
		return false
	}

	if !slices.Equal(s.Aliases, other.Aliases) {
		return false
	}

	if s.ClusterExternalAddresses.Len() != other.ClusterExternalAddresses.Len() {
		return false
	}

	for k, v1 := range s.ClusterExternalAddresses.GetAddresses() {
		if v2, ok := other.ClusterExternalAddresses.Addresses[k]; !ok || !slices.Equal(v1, v2) {
			return false
		}
	}

	if len(s.ClusterExternalPorts) != len(other.ClusterExternalPorts) {
		return false
	}

	for k, v1 := range s.ClusterExternalPorts {
		if v2, ok := s.ClusterExternalPorts[k]; !ok || !maps.Equal(v1, v2) {
			return false
		}
	}
	return s.Name == other.Name && s.Namespace == other.Namespace &&
		s.ServiceRegistry == other.ServiceRegistry && s.K8sAttributes == other.K8sAttributes
}

func (s *Service) SupportsUnhealthyEndpoints() bool {
	return false
}
