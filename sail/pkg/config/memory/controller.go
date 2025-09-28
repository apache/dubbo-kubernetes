package memory

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"k8s.io/apimachinery/pkg/types"
)

// Controller is an implementation of ConfigStoreController.
type Controller struct {
	monitor     Monitor
	configStore model.ConfigStore
	hasSynced   func() bool

	// If meshConfig.DiscoverySelectors are specified, the namespacesFilter tracks the namespaces this controller watches.
	namespacesFilter func(obj interface{}) bool
}

// NewController return an implementation of ConfigStoreController
// This is a client-side monitor that dispatches events as the changes are being
// made on the client.
func NewController(cs model.ConfigStore) *Controller {
	out := &Controller{
		configStore: cs,
		// TODO monitor
	}
	return out
}

func (c *Controller) RegisterHasSyncedHandler(cb func() bool) {
	c.hasSynced = cb
}

func (c *Controller) RegisterEventHandler(kind config.GroupVersionKind, f model.EventHandler) {
	c.monitor.AppendEventHandler(kind, f)
}

// HasSynced return whether store has synced
// It can be controlled externally (such as by the data source),
// otherwise it'll always consider synced.
func (c *Controller) HasSynced() bool {
	if c.hasSynced != nil {
		return c.hasSynced()
	}
	return true
}

func (c *Controller) Run(stop <-chan struct{}) {
	c.monitor.Run(stop)
}

func (c *Controller) Schemas() collection.Schemas {
	return c.configStore.Schemas()
}

func (c *Controller) Get(kind config.GroupVersionKind, key, namespace string) *config.Config {
	if c.namespacesFilter != nil && !c.namespacesFilter(namespace) {
		return nil
	}
	return c.configStore.Get(kind, key, namespace)
}

func (c *Controller) Create(config config.Config) (revision string, err error) {
	if revision, err = c.configStore.Create(config); err == nil {
		c.monitor.ScheduleProcessEvent(ConfigEvent{
			config: config,
			event:  model.EventAdd,
		})
	}
	return
}

func (c *Controller) Update(config config.Config) (newRevision string, err error) {
	oldconfig := c.configStore.Get(config.GroupVersionKind, config.Name, config.Namespace)
	if newRevision, err = c.configStore.Update(config); err == nil {
		c.monitor.ScheduleProcessEvent(ConfigEvent{
			old:    *oldconfig,
			config: config,
			event:  model.EventUpdate,
		})
	}
	return
}

func (c *Controller) UpdateStatus(config config.Config) (newRevision string, err error) {
	oldconfig := c.configStore.Get(config.GroupVersionKind, config.Name, config.Namespace)
	if newRevision, err = c.configStore.UpdateStatus(config); err == nil {
		c.monitor.ScheduleProcessEvent(ConfigEvent{
			old:    *oldconfig,
			config: config,
			event:  model.EventUpdate,
		})
	}
	return
}

func (c *Controller) Patch(orig config.Config, patchFn config.PatchFunc) (newRevision string, err error) {
	cfg, typ := patchFn(orig.DeepCopy())
	switch typ {
	case types.MergePatchType:
	case types.JSONPatchType:
	default:
		return "", fmt.Errorf("unsupported merge type: %s", typ)
	}
	if newRevision, err = c.configStore.Patch(cfg, patchFn); err == nil {
		c.monitor.ScheduleProcessEvent(ConfigEvent{
			old:    orig,
			config: cfg,
			event:  model.EventUpdate,
		})
	}
	return
}

func (c *Controller) Delete(kind config.GroupVersionKind, key, namespace string, resourceVersion *string) error {
	if config := c.Get(kind, key, namespace); config != nil {
		if err := c.configStore.Delete(kind, key, namespace, resourceVersion); err != nil {
			return err
		}
		c.monitor.ScheduleProcessEvent(ConfigEvent{
			config: *config,
			event:  model.EventDelete,
		})
		return nil
	}
	return fmt.Errorf("delete: config %v/%v/%v does not exist", kind, namespace, key)
}

func (c *Controller) List(kind config.GroupVersionKind, namespace string) []config.Config {
	configs := c.configStore.List(kind, namespace)
	if c.namespacesFilter != nil {
		return slices.Filter(configs, func(config config.Config) bool {
			return c.namespacesFilter(config)
		})
	}
	return configs
}
