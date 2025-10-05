package model

import (
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/pkg/config/visibility"
	"github.com/apache/dubbo-kubernetes/pkg/maps"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	"github.com/google/go-cmp/cmp"
	"sync"
	"time"
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
	// Healthy indicates an endpoint is ready to accept traffic
	Healthy HealthStatus = 1
	// UnHealthy indicates an endpoint is not ready to accept traffic
	UnHealthy HealthStatus = 2
	// Draining is a special case, which is used only when persistent sessions are enabled. This indicates an endpoint
	// was previously healthy, but is now shutting down.
	// Without persistent sessions, an endpoint that is shutting down will be marked as Terminating.
	Draining HealthStatus = 3
	// Terminating marks an endpoint as shutting down. Similar to "unhealthy", this means we should not send it traffic.
	// But unlike "unhealthy", this means we do not consider it when calculating failover.
	Terminating HealthStatus = 4
)

type DubboEndpoint struct {
	ServiceAccount         string
	Addresses              []string
	WorkloadName           string
	ServicePortName        string
	Labels                 labels.Instance
	HealthStatus           HealthStatus
	SendUnhealthyEndpoints bool
	DiscoverabilityPolicy  EndpointDiscoverabilityPolicy `json:"-"`
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
	Attributes      ServiceAttributes
	Hostname        host.Name  `json:"hostname"`
	Ports           PortList   `json:"ports,omitempty"`
	ServiceAccounts []string   `json:"serviceAccounts,omitempty"`
	ClusterVIPs     AddressMap `json:"clusterVIPs,omitempty"`
	CreationTime    time.Time  `json:"creationTime,omitempty"`
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
