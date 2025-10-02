package crdclient

import (
	"encoding/json"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/resource"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/maps"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	jsonmerge "github.com/evanphx/json-patch/v5"
	"go.uber.org/atomic"
	"gomodules.xyz/jsonpatch/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sync"
	"time"
)

type Client struct {
	started          *atomic.Bool
	stop             chan struct{}
	kinds            map[config.GroupVersionKind]nsStore
	kindsMu          sync.RWMutex
	domainSuffix     string
	schemasByCRDName map[string]resource.Schema
	schemas          collection.Schemas
	client           kube.Client
	filtersByGVK     map[config.GroupVersionKind]kubetypes.Filter
}

var _ model.ConfigStoreController = &Client{}

type nsStore struct {
	collection krt.Collection[config.Config]
	index      krt.Index[string, config.Config]
	handlers   []krt.HandlerRegistration
}

type Option struct {
	DomainSuffix string
	Identifier   string
	FiltersByGVK map[config.GroupVersionKind]kubetypes.Filter
	KrtDebugger  *krt.DebugHandler
}

func NewForSchemas(client kube.Client, opts Option, schemas collection.Schemas) *Client {
	schemasByCRDName := map[string]resource.Schema{}
	for _, s := range schemas.All() {
		// From the spec: "Its name MUST be in the format <.spec.name>.<.spec.group>."
		name := fmt.Sprintf("%s.%s", s.Plural(), s.Group())
		schemasByCRDName[name] = s
	}

	stop := make(chan struct{})

	out := &Client{
		domainSuffix:     opts.DomainSuffix,
		schemas:          schemas,
		schemasByCRDName: schemasByCRDName,
		started:          atomic.NewBool(false),
		kinds:            map[config.GroupVersionKind]nsStore{},
		client:           client,
		filtersByGVK:     opts.FiltersByGVK,
		stop:             stop,
	}

	kopts := krt.NewOptionsBuilder(stop, "crdclient", opts.KrtDebugger)
	for _, s := range out.schemas.All() {
		// From the spec: "Its name MUST be in the format <.spec.name>.<.spec.group>."
		name := fmt.Sprintf("%s.%s", s.Plural(), s.Group())
		out.addCRD(name, kopts)
	}

	return out
}

func (cl *Client) Run(stop <-chan struct{}) {
	if cl.started.Swap(true) {
		// was already started by other thread
		return
	}

	t0 := time.Now()
	klog.Info("Starting Sail Kubernetes CRD controller")
	if !kube.WaitForCacheSync("crdclient", stop, cl.informerSynced) {
		klog.Errorf("Failed to sync Sail Kubernetes CRD controller cache")
	} else {
		klog.Infof("Sail Kubernetes CRD controller synced in %v", time.Since(t0))
	}
	<-stop
	close(cl.stop)
	klog.Info("controller terminated")
}

func (cl *Client) HasSynced() bool {
	for _, ctl := range cl.allKinds() {
		if !ctl.collection.HasSynced() {
			return false
		}

		for _, h := range ctl.handlers {
			if !h.HasSynced() {
				return false
			}
		}
	}

	return true
}

func (cl *Client) informerSynced() bool {
	for gk, ctl := range cl.allKinds() {
		if !ctl.collection.HasSynced() {
			klog.Infof("controller %q is syncing...", gk)
			return false
		}
	}
	return true
}

func (cl *Client) allKinds() map[config.GroupVersionKind]nsStore {
	cl.kindsMu.RLock()
	defer cl.kindsMu.RUnlock()
	return maps.Clone(cl.kinds)
}

func getObjectMetadata(config config.Config) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:            config.Name,
		Namespace:       config.Namespace,
		Labels:          config.Labels,
		Annotations:     config.Annotations,
		ResourceVersion: config.ResourceVersion,
		OwnerReferences: config.OwnerReferences,
		UID:             types.UID(config.UID),
	}
}

func genPatchBytes(oldRes, modRes runtime.Object, patchType types.PatchType) ([]byte, error) {
	oldJSON, err := json.Marshal(oldRes)
	if err != nil {
		return nil, fmt.Errorf("failed marhsalling original resource: %v", err)
	}
	newJSON, err := json.Marshal(modRes)
	if err != nil {
		return nil, fmt.Errorf("failed marhsalling modified resource: %v", err)
	}
	switch patchType {
	case types.JSONPatchType:
		ops, err := jsonpatch.CreatePatch(oldJSON, newJSON)
		if err != nil {
			return nil, err
		}
		return json.Marshal(ops)
	case types.MergePatchType:
		return jsonmerge.CreateMergePatch(oldJSON, newJSON)
	default:
		return nil, fmt.Errorf("unsupported patch type: %v. must be one of JSONPatchType or MergePatchType", patchType)
	}
}

func (cl *Client) addCRD(name string, opts krt.OptionsBuilder) {
	klog.V(2).Infof("adding CRD %q", name)
	s, f := cl.schemasByCRDName[name]
	if !f {
		klog.V(2).Infof("added resource that we are not watching: %v", name)
		return
	}
	resourceGVK := s.GroupVersionKind()
	gvr := s.GroupVersionResource()

	cl.kindsMu.Lock()
	defer cl.kindsMu.Unlock()
	if _, f := cl.kinds[resourceGVK]; f {
		klog.V(2).Infof("added resource that already exists: %v", resourceGVK)
		return
	}
	translateFunc, f := translationMap[resourceGVK]
	if !f {
		klog.Errorf("translation function for %v not found", resourceGVK)
		return
	}

	var extraFilter func(obj any) bool
	var transform func(obj any) (any, error)
	var fieldSelector string
	if of, f := cl.filtersByGVK[resourceGVK]; f {
		if of.ObjectFilter != nil {
			extraFilter = of.ObjectFilter.Filter
		}
		if of.ObjectTransform != nil {
			transform = of.ObjectTransform
		}
		fieldSelector = of.FieldSelector
	}

	var namespaceFilter kubetypes.DynamicObjectFilter
	if !s.IsClusterScoped() {
		namespaceFilter = cl.client.ObjectFilter()
	}

	filter := kubetypes.Filter{
		ObjectFilter:    kubetypes.ComposeFilters(namespaceFilter, extraFilter),
		ObjectTransform: transform,
		FieldSelector:   fieldSelector,
	}

	var kc kclient.Untyped
	if s.IsBuiltin() {
		kc = kclient.NewUntypedInformer(cl.client, gvr, filter)
	} else {
		kc = kclient.NewDelayedInformer[controllers.Object](
			cl.client,
			gvr,
			kubetypes.StandardInformer,
			filter,
		)
	}

	wrappedClient := krt.WrapClient(kc, opts.WithName("informer/"+resourceGVK.Kind)...)
	collection := krt.MapCollection(wrappedClient, func(obj controllers.Object) config.Config {
		cfg := translateFunc(obj)
		cfg.Domain = cl.domainSuffix
		return cfg
	}, opts.WithName("collection/"+resourceGVK.Kind)...)
	index := krt.NewNamespaceIndex(collection)
	cl.kinds[resourceGVK] = nsStore{
		collection: collection,
		index:      index,
		handlers: []krt.HandlerRegistration{
			collection.RegisterBatch(func(o []krt.Event[config.Config]) {
			}, false),
		},
	}
}

func (cl *Client) kind(r config.GroupVersionKind) (nsStore, bool) {
	cl.kindsMu.RLock()
	defer cl.kindsMu.RUnlock()
	ch, ok := cl.kinds[r]
	return ch, ok
}

func (cl *Client) Schemas() collection.Schemas {
	return cl.schemas
}

func (cl *Client) Get(typ config.GroupVersionKind, name, namespace string) *config.Config {
	h, f := cl.kind(typ)
	if !f {
		klog.Warningf("unknown type: %s", typ)
		return nil
	}

	var key string
	if namespace == "" {
		key = name
	} else {
		key = namespace + "/" + name
	}

	obj := h.collection.GetKey(key)
	if obj == nil {
		klog.V(2).Infof("couldn't find %s/%s in informer index", namespace, name)
		return nil
	}

	return obj
}

func (cl *Client) Create(cfg config.Config) (string, error) {
	if cfg.Spec == nil {
		return "", fmt.Errorf("nil spec for %v/%v", cfg.Name, cfg.Namespace)
	}

	meta, err := create(cl.client, cfg, getObjectMetadata(cfg))
	if err != nil {
		return "", err
	}
	return meta.GetResourceVersion(), nil
}

func (cl *Client) Update(cfg config.Config) (string, error) {
	if cfg.Spec == nil {
		return "", fmt.Errorf("nil spec for %v/%v", cfg.Name, cfg.Namespace)
	}

	meta, err := update(cl.client, cfg, getObjectMetadata(cfg))
	if err != nil {
		return "", err
	}
	return meta.GetResourceVersion(), nil
}

func (cl *Client) UpdateStatus(cfg config.Config) (string, error) {
	if cfg.Status == nil {
		return "", fmt.Errorf("nil status for %v/%v on updateStatus()", cfg.Name, cfg.Namespace)
	}

	meta, err := updateStatus(cl.client, cfg, getObjectMetadata(cfg))
	if err != nil {
		return "", err
	}
	return meta.GetResourceVersion(), nil
}

func (cl *Client) Patch(orig config.Config, patchFn config.PatchFunc) (string, error) {
	modified, patchType := patchFn(orig.DeepCopy())

	meta, err := patch(cl.client, orig, getObjectMetadata(orig), modified, getObjectMetadata(modified), patchType)
	if err != nil {
		return "", err
	}
	return meta.GetResourceVersion(), nil
}

func (cl *Client) Delete(typ config.GroupVersionKind, name, namespace string, resourceVersion *string) error {
	return delete(cl.client, typ, name, namespace, resourceVersion)
}

// List implements store interface
func (cl *Client) List(kind config.GroupVersionKind, namespace string) []config.Config {
	h, f := cl.kind(kind)
	if !f {
		return nil
	}

	if namespace == metav1.NamespaceAll {
		return h.collection.List()
	}

	return h.index.Lookup(namespace)
}

func (cl *Client) RegisterEventHandler(kind config.GroupVersionKind, handler model.EventHandler) {
	if c, ok := cl.kind(kind); ok {
		c.handlers = append(c.handlers, c.collection.RegisterBatch(func(o []krt.Event[config.Config]) {
			for _, event := range o {
				switch event.Event {
				case controllers.EventAdd:
					handler(config.Config{}, *event.New, model.Event(event.Event))
				case controllers.EventUpdate:
					handler(*event.Old, *event.New, model.Event(event.Event))
				case controllers.EventDelete:
					handler(config.Config{}, *event.Old, model.Event(event.Event))
				}
			}
		}, false))
		return
	}

	klog.Warningf("unknown type: %s", kind)
}
