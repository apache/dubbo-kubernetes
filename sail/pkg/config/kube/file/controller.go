package file

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/sail/pkg/config/kube/crd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	krtfiles "github.com/apache/dubbo-kubernetes/pkg/kube/krt/files"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	kubecontroller "github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/kube/controller"
)

var errUnsupportedOp = fmt.Errorf("unsupported operation: the file config store is a read-only view")

type kindStore struct {
	collection krt.Collection[config.Config]
	index      krt.Index[string, config.Config]
	handlers   []krt.HandlerRegistration
}

type Controller struct {
	data    map[config.GroupVersionKind]kindStore
	schemas collection.Schemas
	stop    chan struct{}
}

type ConfigKind struct {
	*config.Config
}

func (c ConfigKind) ResourceName() string {
	if c.Namespace == "" {
		return c.GroupVersionKind.String() + "/" + c.Name
	}

	return c.GroupVersionKind.String() + "/" + c.Namespace + c.Name
}

func (c ConfigKind) Equals(other ConfigKind) bool {
	return c.Config.Equals(other.Config)
}

func NewController(fileDir string, domainSuffix string, schemas collection.Schemas, options kubecontroller.Options) (*Controller, error) {
	stop := make(chan struct{})
	opts := krt.NewOptionsBuilder(stop, "file-monitor", options.KrtDebugger)
	watch, err := krtfiles.NewFolderWatch(fileDir, func(b []byte) ([]*config.Config, error) {
		return parseInputs(b, domainSuffix)
	}, stop)
	if err != nil {
		return nil, err
	}
	mainCollection := krtfiles.NewFileCollection(watch, func(c *config.Config) *ConfigKind {
		return &ConfigKind{c}
	}, opts.WithName("main")...)

	data := make(map[config.GroupVersionKind]kindStore)
	for _, s := range schemas.All() {
		gvk := s.GroupVersionKind()
		if _, ok := collections.Sail.FindByGroupVersionKind(gvk); ok {
			collection := krt.NewCollection(mainCollection, func(ctx krt.HandlerContext, c ConfigKind) *config.Config {
				if c.GroupVersionKind == gvk {
					return c.Config
				}

				return nil
			}, opts.WithName(gvk.Kind)...)

			data[gvk] = kindStore{
				collection: collection,
				index:      krt.NewNamespaceIndex(collection),
			}
		}
	}

	return &Controller{
		schemas: schemas,
		stop:    stop,
		data:    data,
	}, nil
}

func (c *Controller) Schemas() collection.Schemas {
	return c.schemas
}

func (c *Controller) Get(typ config.GroupVersionKind, name, namespace string) *config.Config {
	if data, ok := c.data[typ]; ok {
		if namespace == "" {
			return data.collection.GetKey(name)
		}

		return data.collection.GetKey(namespace + "/" + name)
	}

	return nil
}

func (c *Controller) List(typ config.GroupVersionKind, namespace string) []config.Config {
	if data, ok := c.data[typ]; ok {
		if namespace == metav1.NamespaceAll {
			return data.collection.List()
		}

		return data.index.Lookup(namespace)
	}

	return nil
}

func (c *Controller) Create(config config.Config) (revision string, err error) {
	return "", errUnsupportedOp
}

func (c *Controller) Update(config config.Config) (newRevision string, err error) {
	return "", errUnsupportedOp
}

func (c *Controller) UpdateStatus(config config.Config) (newRevision string, err error) {
	return "", errUnsupportedOp
}

func (c *Controller) Patch(orig config.Config, patchFn config.PatchFunc) (string, error) {
	return "", errUnsupportedOp
}

func (c *Controller) Delete(typ config.GroupVersionKind, name, namespace string, _ *string) error {
	return errUnsupportedOp
}

func (c *Controller) RegisterEventHandler(typ config.GroupVersionKind, handler model.EventHandler) {
	if data, ok := c.data[typ]; ok {
		data.handlers = append(
			data.handlers,
			data.collection.RegisterBatch(func(evs []krt.Event[config.Config]) {
				for _, event := range evs {
					switch event.Event {
					case controllers.EventAdd:
						handler(config.Config{}, *event.New, model.EventAdd)
					case controllers.EventUpdate:
						handler(*event.Old, *event.New, model.EventUpdate)
					case controllers.EventDelete:
						handler(config.Config{}, *event.Old, model.EventDelete)
					}
				}
			}, false),
		)
	}
}

func (c *Controller) Run(stop <-chan struct{}) {
	<-stop
	close(c.stop)
}

func (c *Controller) HasSynced() bool {
	for _, data := range c.data {
		if !data.collection.HasSynced() {
			return false
		}

		for _, handler := range data.handlers {
			if !handler.HasSynced() {
				return false
			}
		}
	}

	return true
}

// parseInputs is identical to crd.ParseInputs, except that it returns an array of config pointers.
func parseInputs(data []byte, domainSuffix string) ([]*config.Config, error) {
	configs, _, err := crd.ParseInputs(string(data))

	// Convert to an array of pointers.
	refs := make([]*config.Config, len(configs))
	for i := range configs {
		refs[i] = &configs[i]
		refs[i].Domain = domainSuffix
	}
	return refs, err
}
