package universal

import (
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache"
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

func (uc *UniversalCache) GetProviders(namespace string, selector cache.Selector) ([]*cache.ServiceModel, error) {
	// TODO: implement
	return nil, nil
}

func (uc *UniversalCache) getId(key string) string {
	id, _ := uc.idCache.LoadOrStore(key, util.Md5_16bit(key))
	return id.(string)
}

func (uc *UniversalCache) store(url *common.URL) {
	if url == nil {
		return
	}
	category := url.GetParam(constant.CategoryKey, "")
	serviceKey := url.ServiceKey()
	id := uc.getId(url.Key())

	var targetCache *sync.Map
	serviceModel := &cache.ServiceModel{}
	switch category {
	case constant.ProvidersCategory:
		provider := &model.Provider{}
		provider.InitByUrl(id, url)
		serviceModel.InitByProvider(provider, url)
		targetCache = &uc.providers
	case constant.ConsumersCategory:
		consumer := &model.Consumer{}
		consumer.InitByUrl(id, url)
		serviceModel.InitByConsumer(consumer, url)
		targetCache = &uc.consumers
	default:
		return
	}

	actual, _ := targetCache.LoadOrStore(serviceKey, map[string]*cache.ServiceModel{})
	if prev, ok := actual.(map[string]*cache.ServiceModel)[id]; ok {
		// exist in cache, update registry type
		if prev.RegistryType == constant.RegistryAll || prev.RegistryType != serviceModel.RegistryType {
			serviceModel.RegistryType = constant.RegistryAll
		}
	} else {
		// not exist in cache, add to cache
		actual.(map[string]*cache.ServiceModel)[id] = serviceModel
	}
}

func (uc *UniversalCache) delete(url *common.URL) {
	if url == nil {
		return
	}
	category := url.GetParam(constant.CategoryKey, "")
	serviceKey := url.ServiceKey()
	id := uc.getId(url.Key())

	var targetCache *sync.Map
	switch category {
	case constant.ProvidersCategory:
		targetCache = &uc.providers
	case constant.ConsumersCategory:
		targetCache = &uc.consumers
	}

	// used to determine whether to delete or change the registry type
	isDeleteOrChangeRegisterType := func(prev *cache.ServiceModel, deleteRegistryType string) bool {
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
			if prev, ok := innerMap.(map[string]*cache.ServiceModel)[id]; ok {
				if isDeleteOrChangeRegisterType(prev, registryType) {
					delete(innerMap.(map[string]*cache.ServiceModel), id)
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
				for id, m := range innerMap.(map[string]*cache.ServiceModel) {
					if isDeleteOrChangeRegisterType(m, registryType) {
						deleteIds = append(deleteIds, id)
					}
				}
				for _, id := range deleteIds {
					delete(innerMap.(map[string]*cache.ServiceModel), id)
				}
			}
			return true
		})
	}
}
