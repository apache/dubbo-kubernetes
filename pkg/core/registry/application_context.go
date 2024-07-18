package registry

import (
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type GlobalRegistryContext struct {
	context sync.Map
}

func (grc *GlobalRegistryContext) GetApplicationContext(appName string) *ApplicationContext {
	value, _ := grc.context.LoadOrStore(appName, NewApplicationContext())
	return value.(*ApplicationContext)
}

// NewGlobalRegistryContext initializes a new GlobalRegistryContext
func NewGlobalRegistryContext() *GlobalRegistryContext {
	return &GlobalRegistryContext{}
}

// DeleteApplicationContext deletes an ApplicationContext for the given appName
func (grc *GlobalRegistryContext) DeleteApplicationContext(appName string) {
	grc.context.Delete(appName)
}

// ListApplicationNames lists all application names in the registry context
func (grc *GlobalRegistryContext) ListApplicationNames() []string {
	var appNames []string
	grc.context.Range(func(key, value interface{}) bool {
		appNames = append(appNames, key.(string))
		return true
	})
	return appNames
}

type ApplicationContext struct {
	serviceUrls        map[string][]*common.URL
	revisionToMetadata map[string]*common.MetadataInfo
	allInstances       map[string][]registry.ServiceInstance
	mu                 sync.RWMutex
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

// ModifyServiceUrls safely modifies the serviceUrls map
func (ac *ApplicationContext) ModifyServiceUrls(modifyFunc func(map[string][]*common.URL)) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	modifyFunc(ac.serviceUrls)
}

// GetRevisionToMetadata returns the reference to the revisionToMetadata map with read lock
func (ac *ApplicationContext) GetRevisionToMetadata() map[string]*common.MetadataInfo {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.revisionToMetadata
}

// SetRevisionToMetadata sets a value in the revisionToMetadata map with write lock
func (ac *ApplicationContext) SetRevisionToMetadata(key string, value *common.MetadataInfo) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.revisionToMetadata[key] = value
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
