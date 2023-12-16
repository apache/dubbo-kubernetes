package cache

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/apache/dubbo-kubernetes/pkg/admin/constant"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
)

type Cache interface {
	GetProviders(namespace string, selector Selector) ([]*ServiceModel, error)
	// TODO: add more methods
}

// ServiceModel is a struct that contains the service information in the cache
type ServiceModel struct {
	Category     string // provider or consumer
	ServiceKey   string // service key
	Group        string
	Version      string
	Application  string
	Protocol     string
	Ip           string
	Port         string
	RegistryType string

	Provider *model.Provider
	Consumer *model.Consumer
}

func (m *ServiceModel) InitByProvider(provider *model.Provider, url *common.URL) {
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

func (m *ServiceModel) InitByConsumer(consumer *model.Consumer, url *common.URL) {
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

func (m *ServiceModel) ToggleRegistryType(deleteType string) {
	if m.RegistryType != constant.RegistryAll {
		return
	}
	if deleteType == constant.RegistryInstance {
		m.RegistryType = constant.RegistryInterface
	} else {
		m.RegistryType = constant.RegistryInstance
	}
}
