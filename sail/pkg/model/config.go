package model

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
)

type ConfigStore interface {
	Schemas() collection.Schemas
	Get(typ config.GroupVersionKind, name, namespace string) *config.Config
	List(typ config.GroupVersionKind, namespace string) []config.Config
	Create(config config.Config) (revision string, err error)
	Update(config config.Config) (newRevision string, err error)
	UpdateStatus(config config.Config) (newRevision string, err error)
	Patch(orig config.Config, patchFn config.PatchFunc) (string, error)
	Delete(typ config.GroupVersionKind, name, namespace string, resourceVersion *string) error
}

type EventHandler = func(config.Config, config.Config, Event)

type ConfigStoreController interface {
	ConfigStore
	RegisterEventHandler(kind config.GroupVersionKind, handler EventHandler)
	Run(stop <-chan struct{})
	HasSynced() bool
}
