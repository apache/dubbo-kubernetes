package registry

import (
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type ApplicationContext struct {
	// InterfaceName Urls
	serviceUrls map[string][]*common.URL
	// Revision Metadata
	revisionToMetadata map[string]*common.MetadataInfo
	// AppName Instances
	allInstances map[string][]registry.ServiceInstance
	mu           sync.RWMutex
}

func NewApplicationContext() *ApplicationContext {
	return &ApplicationContext{
		serviceUrls:        make(map[string][]*common.URL),
		revisionToMetadata: make(map[string]*common.MetadataInfo),
		allInstances:       make(map[string][]registry.ServiceInstance),
	}
}

// GetServiceUrls returns the reference to the serviceUrls map with read lock
func (ac *ApplicationContext) GetServiceUrls() map[string][]*common.URL {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.serviceUrls
}

// SetServiceUrl sets a value in the serviceUrls map with write lock
func (ac *ApplicationContext) SetServiceUrl(interfaceKey string, value []*common.URL) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.serviceUrls[interfaceKey] = value
}

func (ac *ApplicationContext) NewServiceUrls(newServiceUrls map[string][]*common.URL) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.serviceUrls = newServiceUrls
}

// ModifyServiceUrls safely modifies the serviceUrls map
func (ac *ApplicationContext) ModifyServiceUrls(modifyFunc func(map[string][]*common.URL)) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	modifyFunc(ac.serviceUrls)
}

// GetRevisionToMetadata returns the reference to the revisionToMetadata map with read lock
func (ac *ApplicationContext) GetRevisionToMetadata(revision string) *common.MetadataInfo {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.revisionToMetadata[revision]
}

// UpdateRevisionToMetadata sets a value in the revisionToMetadata map with write lock
func (ac *ApplicationContext) UpdateRevisionToMetadata(key string, newKey string, value *common.MetadataInfo) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	delete(ac.revisionToMetadata, key)
	ac.revisionToMetadata[newKey] = value
}

func (ac *ApplicationContext) NewRevisionToMetadata(newRevisionToMetadata map[string]*common.MetadataInfo) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.revisionToMetadata = newRevisionToMetadata
}

// ModifyRevisionToMetadata safely modifies the revisionToMetadata map
func (ac *ApplicationContext) ModifyRevisionToMetadata(modifyFunc func(map[string]*common.MetadataInfo)) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	modifyFunc(ac.revisionToMetadata)
}

// GetAllInstances returns the reference to the allInstances map with read lock
func (ac *ApplicationContext) GetAllInstances() map[string][]registry.ServiceInstance {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.allInstances
}

// SetAllInstances sets a value in the allInstances map with write lock
func (ac *ApplicationContext) SetAllInstances(key string, value []registry.ServiceInstance) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.allInstances[key] = value
}

// ModifyAllInstances safely modifies the allInstances map
func (ac *ApplicationContext) ModifyAllInstances(modifyFunc func(map[string][]registry.ServiceInstance)) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	modifyFunc(ac.allInstances)
}
