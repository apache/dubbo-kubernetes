package registry

import (
	"encoding/json"
	"strconv"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
)

type InterfaceServiceChangedNotifyListener struct {
	mutex          sync.Mutex
	appToInstances *sync.Map
	registry       registry.Registry
}

func NewInterfaceServiceChangedNotifyListener(
	listener *GeneralInterfaceNotifyListener,
) *InterfaceServiceChangedNotifyListener {
	return &InterfaceServiceChangedNotifyListener{
		mutex: sync.Mutex{},
	}
}

// Notify OnEvent on ServiceInstancesChangedEvent the service instances change event
func (iscnl *InterfaceServiceChangedNotifyListener) Notify(event *registry.ServiceEvent) {

}

// NotifyAll the events are complete Service Event List.
// The argument of events []*ServiceEvent is equal to urls []*URL, The Action of serviceEvent should be EventTypeUpdate.
// If your registry center can only get all urls but can't get individual event, you should use this one.
// After notify the address, the callback func will be invoked.
func (iscnl *InterfaceServiceChangedNotifyListener) NotifyAll(events []*registry.ServiceEvent, f func()) {
	for _, event := range events {
		iscnl.Notify(event)
	}
}

func (iscnl *InterfaceServiceChangedNotifyListener) RegisterUrl(url *common.URL) {
	serviceInfo := common.NewServiceInfoWithURL(url)

	appName := url.Service()
	instances, ok := iscnl.appToInstances.Load(appName)
	if !ok {
		instances = make(map[string]registry.ServiceInstance)
		iscnl.appToInstances.Store(appName, instances)
	}
	instanceIDMap := instances.(map[string]registry.DefaultServiceInstance)
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
		instance = registry.DefaultServiceInstance{
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

	go func() {
		err := iscnl.registry.Subscribe(buildServiceSubscribeUrl(serviceInfo), iscnl)
		if err != nil {
			logger.Error("Failed to subscribe to registry, might not be able to show services of the cluster!")
		}
	}()
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

func buildServiceSubscribeUrl(serviceInfo *common.ServiceInfo) *common.URL {
	subscribeUrl, _ := common.NewURL(common.GetLocalIp()+":0",
		common.WithProtocol(consts.AdminProtocol),
		common.WithParams(serviceInfo.GetParams()))
	return subscribeUrl
}
