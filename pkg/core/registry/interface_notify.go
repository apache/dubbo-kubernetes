package registry

import (
	"sync"

	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"
	gxset "github.com/dubbogo/gost/container/set"
)

type InterfaceNotifyListener struct {
	allUrls        gxset.HashSet
	appNames       gxset.HashSet
	appToInstances *sync.Map

	mutex sync.Mutex
}

func NewInterfaceNotifyListener(listener NotifyListener) *InterfaceNotifyListener {
	return &InterfaceNotifyListener{
		appNames:       *gxset.NewSet(),
		appToInstances: &sync.Map{},
		mutex:          sync.Mutex{},
	}
}

func (ilstn *InterfaceNotifyListener) Notify(event *dubboRegistry.ServiceEvent) {
	url := event.Service
	if ilstn.allUrls.Contains(url) {
		return
	}
	appName := url.Service()
	ilstn.mutex.Lock()
	defer ilstn.mutex.Unlock()

	_, ok := ilstn.appToInstances.Load(appName)
	if !ok {
		_ = dubboRegistry.DefaultServiceInstance{
			ID:              url.Ip + ":" + url.Port,
			ServiceName:     appName,
			Host:            "",
			Port:            0,
			Enable:          false,
			Healthy:         false,
			Metadata:        nil,
			ServiceMetadata: nil,
			Address:         "",
			GroupName:       "",
			Tag:             "",
		}

	}
}

func (ilstn *InterfaceNotifyListener) NotifyAll(events []*dubboRegistry.ServiceEvent, f func()) {
}
