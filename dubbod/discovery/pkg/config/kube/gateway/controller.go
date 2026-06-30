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

package gateway

import (
	"fmt"

	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kubetypes"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/features"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/serviceregistry/kube/controller"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/status"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	"go.uber.org/atomic"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gateway "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

var log = dubbolog.RegisterScope("gateway", "gateway-api controller")

var errUnsupportedOp = fmt.Errorf("unsupported operation: the gateway config store is a read-only view")

type Controller struct {
	client         kube.Client
	cluster        cluster.ID
	revision       string
	waitForCRD     func(class schema.GroupVersionResource, stop <-chan struct{}) bool
	gatewayContext krt.RecomputeProtected[*atomic.Pointer[Context]]
	stop           chan struct{}
	xdsUpdater     model.XDSUpdater
	outputs        Outputs
	data           map[config.GroupVersionKind]kindStore
	domainSuffix   string
	status         *status.StatusCollections
	inputs         Inputs
}

type Outputs struct {
	Gateways        krt.Collection[config.Config]
	HTTPRoutes      krt.Collection[config.Config]
	ReferenceGrants referenceGrantStore
}

type kindStore struct {
	collection krt.Collection[config.Config]
	index      krt.Index[string, config.Config]
	handlers   []krt.HandlerRegistration
}

type Inputs struct {
	Services   krt.Collection[*corev1.Service]
	Namespaces krt.Collection[*corev1.Namespace]

	GatewayClasses  krt.Collection[*gateway.GatewayClass]
	Gateways        krt.Collection[*gateway.Gateway]
	HTTPRoutes      krt.Collection[*gateway.HTTPRoute]
	ReferenceGrants krt.Collection[*gatewayv1beta1.ReferenceGrant]
}

func NewController(kc kube.Client, waitForCRD func(class schema.GroupVersionResource, stop <-chan struct{}) bool, options controller.Options, xdsUpdater model.XDSUpdater) *Controller {
	stop := make(chan struct{})
	opts := krt.NewOptionsBuilder(stop, "gateway", options.KrtDebugger)

	c := &Controller{
		client:         kc,
		cluster:        options.ClusterID,
		revision:       options.Revision,
		waitForCRD:     waitForCRD,
		gatewayContext: krt.NewRecomputeProtected(atomic.NewPointer[Context](nil), false, opts.WithName("gatewayContext")...),
		stop:           stop,
		xdsUpdater:     xdsUpdater,
		domainSuffix:   options.DomainSuffix,
		status:         &status.StatusCollections{},
	}

	svcClient := kclient.NewFiltered[*corev1.Service](kc, kubetypes.Filter{ObjectFilter: kc.ObjectFilter()})

	inputs := Inputs{
		Services:       krt.WrapClient(svcClient, opts.WithName("informer/Services")...),
		Namespaces:     krt.NewInformer[*corev1.Namespace](kc, opts.WithName("informer/Namespaces")...),
		GatewayClasses: buildClient[*gateway.GatewayClass](c, kc, gvr.GatewayClass, opts, "informer/GatewayClasses"),
		Gateways:       buildClient[*gateway.Gateway](c, kc, gvr.KubernetesGateway, opts, "informer/Gateways"),
		HTTPRoutes:     buildClient[*gateway.HTTPRoute](c, kc, gvr.HTTPRoute, opts, "informer/HTTPRoutes"),
		ReferenceGrants: buildClient[*gatewayv1beta1.ReferenceGrant](
			c,
			kc,
			gvr.ReferenceGrant,
			opts,
			"informer/ReferenceGrants",
		),
	}

	GatewayClassStatus, GatewayClasses := GatewayClassesCollection(inputs.GatewayClasses, opts)
	status.RegisterStatus(c.status, GatewayClassStatus, GetStatus)

	GatewaysStatus, Gateways := GatewaysCollection(
		inputs.Gateways,
		GatewayClasses,
		opts,
	)

	GatewayFinalStatus := FinalGatewayStatusCollection(GatewaysStatus, opts)
	status.RegisterStatus(c.status, GatewayFinalStatus, GetStatus)

	ReferenceGrants := newReferenceGrantStore(ReferenceGrantsCollection(inputs.ReferenceGrants, opts))

	HTTPRoutes := krt.NewCollection(inputs.HTTPRoutes, func(ctx krt.HandlerContext, hr *gateway.HTTPRoute) *config.Config {
		cfg := convertHTTPRouteToConfig(hr)
		return &cfg
	}, opts.WithName("HTTPRoutes")...)

	outputs := Outputs{
		Gateways:        Gateways,
		HTTPRoutes:      HTTPRoutes,
		ReferenceGrants: ReferenceGrants,
	}
	data := map[config.GroupVersionKind]kindStore{
		gvk.KubernetesGateway: newKindStore(Gateways),
		gvk.HTTPRoute:         newKindStore(HTTPRoutes),
	}

	c.outputs = outputs
	c.data = data
	c.inputs = inputs

	return c
}

func newKindStore(collection krt.Collection[config.Config]) kindStore {
	return kindStore{
		collection: collection,
		index:      krt.NewNamespaceIndex(collection),
	}
}

func (c *Controller) Reconcile(ps *model.PushContext) {
	ctx := NewGatewayContext(ps, c.cluster)
	c.gatewayContext.Modify(func(i **atomic.Pointer[Context]) {
		(*i).Store(&ctx)
	})
	c.gatewayContext.MarkSynced()
}

func (c *Controller) SetStatusWrite(enabled bool, statusManager *status.Manager) {
	if enabled && features.EnableGatewayAPIStatus && statusManager != nil {
		var q status.Queue = statusManager.CreateGenericController(func(status status.Manipulator, context any) {
			status.SetInner(context)
		})
		c.status.SetQueue(q)
	} else {
		c.status.UnsetQueue()
	}
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
		if namespace == "" {
			return data.collection.List()
		}
		return data.index.Lookup(namespace)
	}
	return nil
}

func convertHTTPRouteToConfig(hr *gateway.HTTPRoute) config.Config {
	return config.Config{
		Meta: config.Meta{
			GroupVersionKind:  gvk.HTTPRoute,
			Name:              hr.Name,
			Namespace:         hr.Namespace,
			Labels:            hr.Labels,
			Annotations:       hr.Annotations,
			ResourceVersion:   hr.ResourceVersion,
			CreationTimestamp: hr.CreationTimestamp.Time,
			OwnerReferences:   hr.OwnerReferences,
			UID:               string(hr.UID),
			Generation:        hr.Generation,
		},
		Spec:   hr.Spec.DeepCopy(),
		Status: hr.Status.DeepCopy(),
	}
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

func (c *Controller) HasSynced() bool {
	if !collectionHasSynced(c.inputs.Services) ||
		!collectionHasSynced(c.inputs.Namespaces) ||
		!collectionHasSynced(c.inputs.GatewayClasses) ||
		!collectionHasSynced(c.inputs.Gateways) ||
		!collectionHasSynced(c.inputs.HTTPRoutes) ||
		!collectionHasSynced(c.inputs.ReferenceGrants) {
		return false
	}

	if c.outputs.ReferenceGrants.Collection != nil && !c.outputs.ReferenceGrants.Collection.HasSynced() {
		return false
	}

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

func collectionHasSynced[T any](collection krt.Collection[T]) bool {
	return collection == nil || collection.HasSynced()
}

func (c *Controller) RegisterEventHandler(typ config.GroupVersionKind, handler model.EventHandler) {
	if data, ok := c.data[typ]; ok {
		data.handlers = append(data.handlers, data.collection.RegisterBatch(func(evs []krt.Event[config.Config]) {
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
		}, false))
		c.data[typ] = data
	}
}

func (c *Controller) Schemas() collection.Schemas {
	return collection.SchemasFor(
		collections.KubernetesGateway,
		collections.HTTPRoute,
	)
}

func (c *Controller) Run(stop <-chan struct{}) {
	if features.EnableGatewayAPIGatewayClassController {
		go func() {
			if c.waitForCRD(gvr.GatewayClass, stop) {
				gcc := NewClassController(c.client)
				c.client.RunAndWait(stop)
				gcc.Run(stop)
			}
		}()
	}
	<-stop
	close(c.stop)
}

func (c *Controller) SecretAllowed(ourKind config.GroupVersionKind, resourceName string, namespace string) bool {
	return c.outputs.ReferenceGrants.SecretAllowed(nil, ourKind, resourceName, namespace)
}

func buildClient[I controllers.ComparableObject](
	c *Controller,
	kc kube.Client,
	res schema.GroupVersionResource,
	opts krt.OptionsBuilder,
	name string,
) krt.Collection[I] {
	filter := kclient.Filter{
		ObjectFilter: kubetypes.ComposeFilters(kc.ObjectFilter(), c.inRevision),
	}

	// Gateway API resources select revisions through their parent/attachment semantics.
	if res == gvr.KubernetesGateway || res == gvr.HTTPRoute || res == gvr.ReferenceGrant {
		filter.ObjectFilter = kc.ObjectFilter()
	}

	cc := kclient.NewDelayedInformer[I](kc, res, kubetypes.StandardInformer, filter)
	return krt.WrapClient[I](cc, opts.WithName(name)...)
}

func (c *Controller) inRevision(obj any) bool {
	object := controllers.ExtractObject(obj)
	if object == nil {
		return false
	}
	return config.LabelsInRevision(object.GetLabels(), c.revision)
}
