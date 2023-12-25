package cache

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache/selector"
	"github.com/apache/dubbo-kubernetes/pkg/admin/constant"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
)

type Cache interface {
	GetApplications(namespace string) ([]*ApplicationModel, error)
	GetWorkloads(namespace string) ([]*WorkloadModel, error)
	GetWorkloadsWithSelector(namespace string, selector selector.Selector) ([]*WorkloadModel, error)
	GetInstances(namespace string) ([]*InstanceModel, error)
	GetInstancesWithSelector(namespace string, selector selector.Selector) ([]*InstanceModel, error)
	GetServices(namespace string) ([]*ServiceModel, error)
	GetServicesWithSelector(namespace string, selector selector.Selector) ([]*ServiceModel, error)

	// TODO: add support for other resources
	// Telemetry/Metrics
	// ConditionRule
	// TagRule
	// DynamicConfigurationRule
}

// ApplicationModel is an application in dubbo or kubernetes
type ApplicationModel struct {
	Name string
}

// WorkloadModel is a dubbo process or a deployment in kubernetes
type WorkloadModel struct {
	Application *ApplicationModel
	Name        string
	Type        string
	Image       string
	Labels      map[string]string
}

type InstanceModel struct {
	Application *ApplicationModel
	Workload    *WorkloadModel
	Name        string
	Ip          string
	Status      string
	Node        string
	Labels      map[string]string
}

type ServiceModel struct {
	Application *ApplicationModel
	Name        string
	Labels      map[string]string
	ServiceKey  string
	Group       string
	Version     string
}

// DubboModel is a dubbo provider or consumer in registry
type DubboModel struct {
	Application  string
	Category     string // provider or consumer
	ServiceKey   string // service key
	Group        string
	Version      string
	Protocol     string
	Ip           string
	Port         string
	RegistryType string

	Provider *model.Provider
	Consumer *model.Consumer
}

func (m *DubboModel) InitByProvider(provider *model.Provider, url *common.URL) {
	m.Provider = provider

	m.Category = constant.ProviderSide
	m.ServiceKey = provider.Service
	m.Group = url.Group()
	m.Version = url.Version()
	m.Application = provider.Application
	m.Protocol = url.Protocol
	m.Ip = url.Ip
	m.Port = url.Port
	m.RegistryType = url.GetParam(constant.RegistryType, constant.RegistryInstance)
}

func (m *DubboModel) InitByConsumer(consumer *model.Consumer, url *common.URL) {
	m.Consumer = consumer

	m.Category = constant.ConsumerSide
	m.ServiceKey = consumer.Service
	m.Group = url.Group()
	m.Version = url.Version()
	m.Application = consumer.Application
	m.Protocol = url.Protocol
	m.Ip = url.Ip
	m.Port = url.Port
	m.RegistryType = url.GetParam(constant.RegistryType, constant.RegistryInstance)
}

func (m *DubboModel) ToggleRegistryType(deleteType string) {
	if m.RegistryType != constant.RegistryAll {
		return
	}
	if deleteType == constant.RegistryInstance {
		m.RegistryType = constant.RegistryInterface
	} else {
		m.RegistryType = constant.RegistryInstance
	}
}
