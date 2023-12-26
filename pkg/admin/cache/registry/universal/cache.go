package universal

import (
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache"
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache/selector"
	"github.com/apache/dubbo-kubernetes/pkg/admin/constant"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/admin/util"
)

var UniversalCacheInstance *UniversalCache

type UniversalCache struct {
	providers sync.Map
	consumers sync.Map
	idCache   sync.Map
}

func NewUniversalCache() *UniversalCache {
	return &UniversalCache{}
}

func (uc *UniversalCache) GetApplications(namespace string) ([]*cache.ApplicationModel, error) {
	names := map[string]struct{}{}
	uc.providers.Range(func(key, value interface{}) bool {
		if innerMap, ok := value.(map[string]*DubboModel); ok {
			for _, v := range innerMap {
				names[v.Application] = struct{}{}
			}
		}
		return true
	})

	uc.consumers.Range(func(key, value interface{}) bool {
		if innerMap, ok := value.(map[string]*DubboModel); ok {
			for _, v := range innerMap {
				names[v.Application] = struct{}{}
			}
		}
		return true
	})

	applications := make([]*cache.ApplicationModel, 0, len(names))
	for name := range names {
		applications = append(applications, &cache.ApplicationModel{Name: name})
	}

	return applications, nil
}

func (uc *UniversalCache) GetWorkloads(namespace string) ([]*cache.WorkloadModel, error) {
	// TODO implement me
	return []*cache.WorkloadModel{}, nil
}

func (uc *UniversalCache) GetWorkloadsWithSelector(namespace string, selector selector.Selector) ([]*cache.WorkloadModel, error) {
	// TODO implement me
	return []*cache.WorkloadModel{}, nil
}

func (uc *UniversalCache) GetInstances(namespace string) ([]*cache.InstanceModel, error) {
	// TODO implement me
	return []*cache.InstanceModel{}, nil
}

func (uc *UniversalCache) GetInstancesWithSelector(namespace string, selector selector.Selector) ([]*cache.InstanceModel, error) {
	// TODO implement me
	return []*cache.InstanceModel{}, nil
}

func (uc *UniversalCache) GetServices(namespace string) ([]*cache.ServiceModel, error) {
	// TODO implement me
	panic("implement me")
}

func (uc *UniversalCache) GetServicesWithSelector(namespace string, selector selector.Selector) ([]*cache.ServiceModel, error) {
	// TODO implement me
	panic("implement me")
}

func (uc *UniversalCache) getId(key string) string {
	id, _ := uc.idCache.LoadOrStore(key, util.Md5_16bit(key))
	return id.(string)
}

func serviceKey(url *common.URL) string {
	return url.GetParam(constant.ApplicationKey, "") + url.ServiceKey()
}

func (uc *UniversalCache) store(url *common.URL) {
	if url == nil {
		return
	}
	category := url.GetParam(constant.CategoryKey, "")
	serviceKey := serviceKey(url)
	id := uc.getId(url.Key())

	var targetCache *sync.Map
	dubboModel := &DubboModel{}
	switch category {
	case constant.ProvidersCategory:
		provider := &model.Provider{}
		provider.InitByUrl(id, url)
		dubboModel.InitByProvider(provider, url)
		targetCache = &uc.providers
	case constant.ConsumersCategory:
		consumer := &model.Consumer{}
		consumer.InitByUrl(id, url)
		dubboModel.InitByConsumer(consumer, url)
		targetCache = &uc.consumers
	default:
		return
	}

	actual, _ := targetCache.LoadOrStore(serviceKey, map[string]*DubboModel{})
	if prev, ok := actual.(map[string]*DubboModel)[id]; ok {
		// exist in cache, update registry type
		if prev.RegistryType == constant.RegistryAll || prev.RegistryType != dubboModel.RegistryType {
			dubboModel.RegistryType = constant.RegistryAll
		}
	} else {
		// not exist in cache, add to cache
		actual.(map[string]*DubboModel)[id] = dubboModel
	}
}

func (uc *UniversalCache) delete(url *common.URL) {
	if url == nil {
		return
	}
	category := url.GetParam(constant.CategoryKey, "")
	serviceKey := serviceKey(url)
	id := uc.getId(url.Key())

	var targetCache *sync.Map
	switch category {
	case constant.ProvidersCategory:
		targetCache = &uc.providers
	case constant.ConsumersCategory:
		targetCache = &uc.consumers
	}

	// used to determine whether to delete or change the registry type
	isDeleteOrChangeRegisterType := func(prev *DubboModel, deleteRegistryType string) bool {
		if prev.RegistryType == deleteRegistryType {
			return true
		}
		if prev.RegistryType == constant.RegistryAll {
			prev.ToggleRegistryType(deleteRegistryType)
		}
		return false
	}

	registryType := url.GetParam(constant.RegistryType, constant.RegistryInstance)
	group := url.Group()
	version := url.Version()
	if group != constant.AnyValue && version != constant.AnyValue {
		// delete by serviceKey and id
		if innerMap, ok := targetCache.Load(serviceKey); ok {
			if prev, ok := innerMap.(map[string]*DubboModel)[id]; ok {
				if isDeleteOrChangeRegisterType(prev, registryType) {
					delete(innerMap.(map[string]*DubboModel), id)
				}
			}
		}
	} else {
		// support delete by wildcard search
		targetCache.Range(func(serviceKey, innerMap interface{}) bool {
			curServiceKey := serviceKey.(string)
			if util.GetInterface(curServiceKey) == url.Service() &&
				(group == constant.AnyValue || group == util.GetGroup(curServiceKey)) &&
				(version == constant.AnyValue || version == util.GetVersion(curServiceKey)) {
				deleteIds := make([]string, 0)
				for id, m := range innerMap.(map[string]*DubboModel) {
					if isDeleteOrChangeRegisterType(m, registryType) {
						deleteIds = append(deleteIds, id)
					}
				}
				for _, id := range deleteIds {
					delete(innerMap.(map[string]*DubboModel), id)
				}
			}
			return true
		})
	}
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
