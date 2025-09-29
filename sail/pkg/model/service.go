package model

import (
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/pkg/config/visibility"
	"github.com/apache/dubbo-kubernetes/pkg/maps"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	corev1 "k8s.io/api/core/v1"
)

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

	// k8s specific fields
	KubernetesService   *corev1.Service   `json:"kubernetesService,omitempty"`
	KubernetesNamespace string            `json:"kubernetesNamespace,omitempty"`
	DubboAnnotations    map[string]string `json:"dubboAnnotations,omitempty"`

	// istio specific fields
	IstioDestinationRule map[string]interface{} `json:"istioDestinationRule,omitempty"`
	IstioVirtualService  map[string]interface{} `json:"istioVirtualService,omitempty"`
	IstioWorkloadEntry   map[string]interface{} `json:"istioWorkloadEntry,omitempty"`
	IstioServiceEntry    map[string]interface{} `json:"istioServiceEntry,omitempty"`
	IstioGateway         map[string]interface{} `json:"istioGateway,omitempty"`
	IstioSidecar         map[string]interface{} `json:"istioSidecar,omitempty"`
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
	if s == nil {
		return nil
	}
	out := &Service{
		Attributes:      s.Attributes.DeepCopy(),
		Hostname:        s.Hostname,
		CreationTime:    s.CreationTime,
	}
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
	if s.ClusterVIPs.Addresses != nil {
		out.ClusterVIPs = *s.ClusterVIPs.DeepCopy()
	} else {
		out.ClusterVIPs = AddressMap{}
	}
	return out
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

	// Clone k8s specific fields
	if s.KubernetesService != nil {
		out.KubernetesService = s.KubernetesService.DeepCopy()
	}
	out.DubboAnnotations = maps.Clone(s.DubboAnnotations)

	// Clone istio specific fields
	if s.IstioDestinationRule != nil {
		out.IstioDestinationRule = maps.Clone(s.IstioDestinationRule)
	}
	if s.IstioVirtualService != nil {
		out.IstioVirtualService = maps.Clone(s.IstioVirtualService)
	}
	if s.IstioWorkloadEntry != nil {
		out.IstioWorkloadEntry = maps.Clone(s.IstioWorkloadEntry)
	}
	if s.IstioServiceEntry != nil {
		out.IstioServiceEntry = maps.Clone(s.IstioServiceEntry)
	}
	if s.IstioGateway != nil {
		out.IstioGateway = maps.Clone(s.IstioGateway)
	}
	if s.IstioSidecar != nil {
		out.IstioSidecar = maps.Clone(s.IstioSidecar)
	}

	// AddressMap contains a mutex, which is safe to return a copy in this case.
	// nolint: govet
	return out
}
