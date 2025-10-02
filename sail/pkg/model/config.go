package model

import (
	"cmp"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	"github.com/apache/dubbo-kubernetes/pkg/util/hash"
	"sort"
)

type ConfigHash uint64

const (
	NamespaceAll = ""
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

func sortConfigByCreationTime(configs []config.Config) []config.Config {
	sort.Slice(configs, func(i, j int) bool {
		if r := configs[i].CreationTimestamp.Compare(configs[j].CreationTimestamp); r != 0 {
			return r == -1 // -1 means i is less than j, so return true
		}
		if r := cmp.Compare(configs[i].Name, configs[j].Name); r != 0 {
			return r == -1
		}
		return cmp.Compare(configs[i].Namespace, configs[j].Namespace) == -1
	})
	return configs
}

func (key ConfigKey) String() string {
	return key.Kind.String() + "/" + key.Namespace + "/" + key.Name
}

func (key ConfigKey) HashCode() ConfigHash {
	h := hash.New()
	h.Write([]byte{byte(key.Kind)})
	// Add separator / to avoid collision.
	h.WriteString("/")
	h.WriteString(key.Namespace)
	h.WriteString("/")
	h.WriteString(key.Name)
	return ConfigHash(h.Sum64())
}
