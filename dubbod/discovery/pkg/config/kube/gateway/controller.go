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
	handlers       []krt.HandlerRegistration
	outputs        Outputs
	domainSuffix   string
	status         *status.StatusCollections
	inputs         Inputs
}

type Outputs struct {
	Gateways krt.Collection[Gateway]
}

type Inputs struct {
	Services   krt.Collection[*corev1.Service]
	Namespaces krt.Collection[*corev1.Namespace]

	GatewayClasses krt.Collection[*gateway.GatewayClass]
	Gateways       krt.Collection[*gateway.Gateway]
	HTTPRoutes     krt.Collection[*gateway.HTTPRoute]
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

	handlers := []krt.HandlerRegistration{}
	outputs := Outputs{
		Gateways: Gateways,
	}

	c.outputs = outputs
	c.handlers = handlers
	c.inputs = inputs

	return c
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
	return nil
}

func (c *Controller) List(typ config.GroupVersionKind, namespace string) []config.Config {
	switch typ {
	case gvk.HTTPRoute:
		// Convert HTTPRoute collection to config.Config list
		httpRoutes := c.inputs.HTTPRoutes.List()
		result := make([]config.Config, 0, len(httpRoutes))
		for _, hr := range httpRoutes {
			cfg := convertHTTPRouteToConfig(hr)
			if namespace == "" || cfg.Namespace == namespace {
				result = append(result, cfg)
			}
		}
		return result
	default:
		return nil
	}
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
		Spec:   &hr.Spec,
		Status: &hr.Status,
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
	// Check if all input collections are synced
	if !c.inputs.Services.HasSynced() {
		return false
	}
	if !c.inputs.Namespaces.HasSynced() {
		return false
	}
	if !c.inputs.GatewayClasses.HasSynced() {
		return false
	}
	if !c.inputs.Gateways.HasSynced() {
		return false
	}
	if !c.inputs.HTTPRoutes.HasSynced() {
		return false
	}
	for _, h := range c.handlers {
		if !h.HasSynced() {
			return false
		}
	}
	return true
}

func (c *Controller) RegisterEventHandler(typ config.GroupVersionKind, handler model.EventHandler) {
	// We do not do event handler registration this way, and instead directly call the XDS Updated.
}

func (c *Controller) Schemas() collection.Schemas {
	return collection.SchemasFor(
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
	return true // TODO
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

	// Gateway and HTTPRoute are not filtered by revision, they need to select tags as well
	if res == gvr.KubernetesGateway || res == gvr.HTTPRoute {
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
