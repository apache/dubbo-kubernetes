package context

import (
	ctx "context"

	"github.com/apache/dubbo-kubernetes/pkg/config/app"
	"github.com/apache/dubbo-kubernetes/pkg/core/manager"
	coreruntime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/core/store"
)

type Context interface {

	ResourceManager() manager.ResourceManager

	ResourceStore() store.ResourceStore

	Config()	app.AdminConfig

	AppContext() ctx.Context
}

var _ Context = &context{}


func NewConsoleContext(coreRt coreruntime.Runtime) Context {
	return &context{
		coreRt: coreRt,
	}
}

type context struct {
	coreRt          coreruntime.Runtime
}

func (c *context) AppContext() ctx.Context {
	return c.coreRt.AppContext()
}

func (c *context) Config() app.AdminConfig {
	return c.coreRt.Config()
}

func (c *context) ResourceManager() manager.ResourceManager {
	rmc, _ := c.coreRt.GetComponent(coreruntime.ResourceManager)
	return rmc.(manager.ResourceManagerComponent).ResourceManager()
}

func (c *context) ResourceStore() store.ResourceStore {
	rsc, _ := c.coreRt.GetComponent(coreruntime.ResourceStore)
	return rsc.(store.BaseResourceStoreComponent).ResourceStore()
}

