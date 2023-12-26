package cache

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache/selector"
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

type ApplicationModel struct {
	Name string
}

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
	Port        string
	Status      string
	Node        string
	Labels      map[string]string
}

type ServiceModel struct {
	Application *ApplicationModel
	Category    string
	Name        string
	Labels      map[string]string
	ServiceKey  string
	Group       string
	Version     string
}
