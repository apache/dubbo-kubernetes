package registry

import (
	"encoding/json"
	"strconv"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	gxset "github.com/dubbogo/gost/container/set"
)

// Endpoint nolint
type Endpoint struct {
	Port     int    `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

type InterfaceNotifyListener struct {
	allUrls        gxset.HashSet
	appToInstances *sync.Map
	NotifyListener
	registry dubboRegistry.Registry
	mutex    sync.Mutex
}

func NewInterfaceNotifyListener(listener NotifyListener, registry dubboRegistry.Registry) *InterfaceNotifyListener {
	return &InterfaceNotifyListener{
		allUrls:        *gxset.NewSet(),
		appToInstances: &sync.Map{},
		registry:       registry,
		NotifyListener: listener,
	}
}

func (ilstn *InterfaceNotifyListener) Notify(event *dubboRegistry.ServiceEvent) {
	url := event.Service
	urlStr := url.String()

	ilstn.mutex.Lock()
	defer ilstn.mutex.Unlock()

	if ilstn.allUrls.Contains(urlStr) {
		return
	}
	ilstn.allUrls.Add(urlStr)

	serviceInfo := common.NewServiceInfoWithURL(url)

	appName := url.Service()
	instances, ok := ilstn.appToInstances.Load(appName)
	if !ok {
		instances = make(map[string]dubboRegistry.ServiceInstance)
		ilstn.appToInstances.Store(appName, instances)
	}
	instanceIDMap := instances.(map[string]dubboRegistry.DefaultServiceInstance)
	instance, exists := instanceIDMap[url.Address()]

	if !exists {
		serviceMetadata := common.NewMetadataInfo(appName, "", make(map[string]*common.ServiceInfo))
		serviceMetadata.AddService(serviceInfo)
		port, _ := strconv.Atoi(url.Port)
		metadata := map[string]string{
			constant.ExportedServicesRevisionPropertyName: serviceMetadata.CalAndGetRevision(),
			constant.MetadataStorageTypePropertyName:      constant.DefaultMetadataStorageType,
			constant.TimestampKey:                         url.GetParam(constant.TimestampKey, ""),
			constant.ServiceInstanceEndpoints:             getEndpointsStr(url.Protocol, port),
			constant.MetadataServiceURLParamsPropertyName: getURLParams(serviceInfo),
		}
		instance = dubboRegistry.DefaultServiceInstance{
			ID:              url.Address(),
			ServiceName:     appName,
			Host:            url.Ip,
			Port:            port,
			Enable:          true,
			Healthy:         true,
			Metadata:        metadata,
			ServiceMetadata: serviceMetadata,
			Address:         url.Address(),
			GroupName:       serviceInfo.Group,
			Tag:             "",
		}
		instanceIDMap[instance.ID] = instance
	} else {
		serviceMetadata := instance.ServiceMetadata
		serviceMetadata.AddService(serviceInfo)
		instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName] = serviceMetadata.CalAndGetRevision()
	}

}

func (ilstn *InterfaceNotifyListener) NotifyAll(events []*dubboRegistry.ServiceEvent, f func()) {
	for _, event := range events {
		ilstn.Notify(event)
	}
}

// endpointsStr convert the map to json like [{"protocol": "dubbo", "port": 123}]
func getEndpointsStr(protocol string, port int) string {
	protocolMap := make(map[string]int, 4)
	protocolMap[protocol] = port
	if len(protocolMap) == 0 {
		return ""
	}

	endpoints := make([]Endpoint, 0, len(protocolMap))
	for k, v := range protocolMap {
		endpoints = append(endpoints, Endpoint{
			Port:     v,
			Protocol: k,
		})
	}

	str, err := json.Marshal(endpoints)
	if err != nil {
		logger.Errorf("could not convert the endpoints to json")
		return ""
	}
	return string(str)
}

func getURLParams(serviceInfo *common.ServiceInfo) string {
	urlParams, err := json.Marshal(serviceInfo.Params)
	if err != nil {
		logger.Error("could not convert the url params to json")
		return ""
	}
	return string(urlParams)
}
