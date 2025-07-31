package model

import (
	"github.com/apache/dubbo-kubernetes/navigator/pkg/features"
)

type XdsCache interface{}

type DisabledCache struct{}

func NewXdsCache() XdsCache {
	cache := XdsCacheImpl{
		eds: newTypedXdsCache[uint64](),
	}
	if features.EnableCDSCaching {
		cache.cds = newTypedXdsCache[uint64]()
	} else {
		cache.cds = disabledCache[uint64]{}
	}
	if features.EnableRDSCaching {
		cache.rds = newTypedXdsCache[uint64]()
	} else {
		cache.rds = disabledCache[uint64]{}
	}

	cache.sds = newTypedXdsCache[string]()

	return cache
}
