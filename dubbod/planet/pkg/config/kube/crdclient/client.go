//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package crdclient

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/maps"
	jsonmerge "github.com/evanphx/json-patch/v5"
	"go.uber.org/atomic"
	"gomodules.xyz/jsonpatch/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type Client struct {
	started          *atomic.Bool
	stop             chan struct{}
	kinds            map[config.GroupVersionKind]nsStore
	kindsMu          sync.RWMutex
	domainSuffix     string
	schemasByCRDName map[string]collection.Schema
	schemas          collection.Schemas
	client           kube.Client
	filtersByGVK     map[config.GroupVersionKind]kubetypes.Filter
	logger           *log.Logger
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
	schemasByCRDName := map[string]collection.Schema{}
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
		logger:           log.RegisterScope("crdclient", "Planet Kubernetes CRD controller"),
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
	cl.logger.Infof("Starting Planet Kubernetes CRD controller")
	if !kube.WaitForCacheSync("crdclient", stop, cl.informerSynced) {
		cl.logger.Infof("Failed to sync Planet Kubernetes CRD controller cache")
	} else {
		cl.logger.Infof("Planet Kubernetes CRD controller synced in %v", time.Since(t0))
	}
	<-stop
	close(cl.stop)
	cl.logger.Infof("controller terminated")
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
			cl.logger.Infof("controller %q is syncing...", gk)
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

func (cl *Client) addCRD(name string, opts krt.OptionsBuilder) {
	cl.logger.Debugf("addCRD: adding CRD %q", name)
	s, f := cl.schemasByCRDName[name]
	if !f {
		cl.logger.Debugf("addCRD: added resource that we are not watching: %v", name)
		return
	}
	resourceGVK := s.GroupVersionKind()
	gvr := s.GroupVersionResource()
	cl.logger.Debugf("addCRD: CRD %q maps to GVK %v, GVR %v", name, resourceGVK, gvr)

	cl.kindsMu.Lock()
	defer cl.kindsMu.Unlock()
	if _, f := cl.kinds[resourceGVK]; f {
		cl.logger.Warnf("addCRD: added resource that already exists: %v", resourceGVK)
		return
	}
	translateFunc, f := translationMap[resourceGVK]
	if !f {
		cl.logger.Errorf("translation function for %v not found", resourceGVK)
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
		cl.logger.Debugf("addCRD: using namespace filter for %v (not cluster-scoped)", resourceGVK)
	} else {
		cl.logger.Debugf("addCRD: no namespace filter for %v (cluster-scoped)", resourceGVK)
	}

	filter := kubetypes.Filter{
		ObjectFilter:    kubetypes.ComposeFilters(namespaceFilter, extraFilter),
		ObjectTransform: transform,
		FieldSelector:   fieldSelector,
	}
	cl.logger.Debugf("addCRD: created filter for %v (namespaceFilter=%v, extraFilter=%v, fieldSelector=%v)", resourceGVK, namespaceFilter != nil, extraFilter != nil, fieldSelector)

	var kc kclient.Untyped
	if s.IsBuiltin() {
		kc = kclient.NewUntypedInformer(cl.client, gvr, filter)
	} else {
		// For DestinationRule and VirtualService, we use Dynamic client which returns unstructured objects
		// So we need to use DynamicInformer type to ensure the informer expects unstructured objects
		informerType := kubetypes.StandardInformer
		if resourceGVK == gvk.DestinationRule || resourceGVK == gvk.VirtualService || resourceGVK == gvk.PeerAuthentication {
			informerType = kubetypes.DynamicInformer
			cl.logger.Debugf("addCRD: using DynamicInformer for %v (uses Dynamic client)", resourceGVK)
		}
		kc = kclient.NewDelayedInformer[controllers.Object](
			cl.client,
			gvr,
			informerType,
			filter,
		)
	}

	wrappedClient := krt.WrapClient(kc, opts.WithName("informer/"+resourceGVK.Kind)...)
	collection := krt.MapCollection(wrappedClient, func(obj controllers.Object) config.Config {
		cfg := translateFunc(obj)
		cfg.Domain = cl.domainSuffix
		// Only log at Debug level to avoid spam, but keep it available for diagnosis
		cl.logger.Debugf("addCRD: MapCollection translating object %s/%s to config for %v", obj.GetNamespace(), obj.GetName(), resourceGVK)
		return cfg
	}, opts.WithName("collection/"+resourceGVK.Kind)...)
	index := krt.NewNamespaceIndex(collection)
	// Register a debug handler to track all events from the wrappedClient (before MapCollection)
	// This helps diagnose if events are being filtered before reaching the collection
	wrappedClientDebugHandler := wrappedClient.RegisterBatch(func(o []krt.Event[controllers.Object]) {
		if len(o) > 0 {
			cl.logger.Debugf("addCRD: wrappedClient event detected for %v: %d events", resourceGVK, len(o))
			for i, event := range o {
				var nameStr, nsStr string
				if event.New != nil {
					obj := *event.New
					nameStr = obj.GetName()
					nsStr = obj.GetNamespace()
				} else if event.Old != nil {
					obj := *event.Old
					nameStr = obj.GetName()
					nsStr = obj.GetNamespace()
				}
				cl.logger.Debugf("addCRD: wrappedClient event[%d] %s for %v (name=%s/%s)",
					i, event.Event, resourceGVK, nsStr, nameStr)
			}
		}
	}, false)
	// Register a debug handler to track all events from the collection
	// This helps diagnose why new config changes might not trigger events
	// Use false to match Dubbo's implementation - only process future events, not initial sync
	debugHandler := collection.RegisterBatch(func(o []krt.Event[config.Config]) {
		if len(o) > 0 {
			cl.logger.Debugf("addCRD: collection event detected for %v: %d events", resourceGVK, len(o))
			for i, event := range o {
				var nameStr, nsStr string
				if event.New != nil {
					nameStr = event.New.Name
					nsStr = event.New.Namespace
				} else if event.Old != nil {
					nameStr = event.Old.Name
					nsStr = event.Old.Namespace
				}
				cl.logger.Debugf("addCRD: collection event[%d] %s for %v (name=%s/%s)",
					i, event.Event, resourceGVK, nsStr, nameStr)
			}
		}
	}, false)
	cl.kinds[resourceGVK] = nsStore{
		collection: collection,
		index:      index,
		handlers: []krt.HandlerRegistration{
			wrappedClientDebugHandler,
			debugHandler,
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

func (cl *Client) RegisterEventHandler(kind config.GroupVersionKind, handler model.EventHandler) {
	cl.kindsMu.Lock()
	defer cl.kindsMu.Unlock()

	c, ok := cl.kinds[kind]
	if !ok {
		cl.logger.Warnf("unknown type: %s", kind)
		return
	}

	cl.logger.Debugf("RegisterEventHandler: registering handler for %v", kind)
	// Match Dubbo's implementation: RegisterBatch returns a HandlerRegistration that is already
	// registered with the collection, so we just need to append it to handlers to keep a reference
	// The handler will be called by the collection when events occur, regardless of whether we
	// update cl.kinds[kind] or not. However, we update it to keep the handlers slice in sync.
	handlerReg := c.collection.RegisterBatch(func(o []krt.Event[config.Config]) {
		cl.logger.Debugf("RegisterEventHandler: batch handler triggered for %v with %d events", kind, len(o))
		for i, event := range o {
			var nameStr, nsStr string
			if event.New != nil {
				nameStr = event.New.Name
				nsStr = event.New.Namespace
			} else if event.Old != nil {
				nameStr = event.Old.Name
				nsStr = event.Old.Namespace
			}
			cl.logger.Debugf("RegisterEventHandler: processing event[%d] %s for %v (name=%s/%s)",
				i, event.Event, kind, nsStr, nameStr)
			switch event.Event {
			case controllers.EventAdd:
				if event.New != nil {
					handler(config.Config{}, *event.New, model.Event(event.Event))
				} else {
					cl.logger.Warnf("RegisterEventHandler: EventAdd but event.New is nil, skipping")
				}
			case controllers.EventUpdate:
				if event.Old != nil && event.New != nil {
					handler(*event.Old, *event.New, model.Event(event.Event))
				} else {
					cl.logger.Warnf("RegisterEventHandler: EventUpdate but event.Old or event.New is nil, skipping")
				}
			case controllers.EventDelete:
				if event.Old != nil {
					handler(config.Config{}, *event.Old, model.Event(event.Event))
				} else {
					cl.logger.Warnf("RegisterEventHandler: EventDelete but event.Old is nil, skipping")
				}
			}
		}
	}, false)
	// Update handlers slice to keep reference (though not strictly necessary for functionality)
	c.handlers = append(c.handlers, handlerReg)
	cl.kinds[kind] = c
	cl.logger.Debugf("RegisterEventHandler: successfully registered handler for %v", kind)
}

func (cl *Client) Get(typ config.GroupVersionKind, name, namespace string) *config.Config {
	h, f := cl.kind(typ)
	if !f {
		cl.logger.Warnf("unknown type: %s", typ)
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
		cl.logger.Debugf("couldn't find %s/%s in informer index", namespace, name)
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

func (cl *Client) List(kind config.GroupVersionKind, namespace string) []config.Config {
	h, f := cl.kind(kind)
	if !f {
		cl.logger.Warnf("List: unknown kind %v", kind)
		return nil
	}

	// Check if collection is synced
	if !h.collection.HasSynced() {
		cl.logger.Warnf("List: collection for %v is not synced yet", kind)
	}

	var configs []config.Config
	if namespace == metav1.NamespaceAll {
		// Get all configs from collection
		configs = h.collection.List()
		cl.logger.Debugf("List: found %d configs for %v (namespace=all, synced=%v)",
			len(configs), kind, h.collection.HasSynced())
		if len(configs) > 0 {
			for i, cfg := range configs {
				cl.logger.Debugf("List: config[%d] %s/%s for %v", i, cfg.Namespace, cfg.Name, kind)
			}
		} else {
			cl.logger.Debugf("List: collection returned 0 configs for %v (synced=%v), this may indicate informer is not watching correctly or resources are being filtered", kind, h.collection.HasSynced())
		}
		// Log collection type for diagnosis
		cl.logger.Debugf("List: collection type is %T, HasSynced=%v", h.collection, h.collection.HasSynced())
	} else {
		configs = h.index.Lookup(namespace)
		cl.logger.Debugf("List: found %d configs for %v in namespace %s (synced=%v)", len(configs), kind, namespace, h.collection.HasSynced())
		if len(configs) > 0 {
			for i, cfg := range configs {
				cl.logger.Debugf("List: config[%d] %s/%s for %v", i, cfg.Namespace, cfg.Name, kind)
			}
		} else {
			cl.logger.Debugf("List: found 0 configs for %v in namespace %s (synced=%v), checking if resources exist in cluster", kind, namespace, h.collection.HasSynced())
		}
	}

	return configs
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
