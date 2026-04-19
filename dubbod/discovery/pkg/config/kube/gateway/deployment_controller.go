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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	gateway "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/yaml"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/features"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
)

var logger = dubbolog.RegisterScope("gateway-deployment-controller", "gateway deployment controller debugging")

type classInfo struct {
	controller             string
	controllerLabel        string
	description            string
	templates              string
	defaultServiceType     corev1.ServiceType
	disableRouteGeneration bool
	supportsListenerSet    bool
	disableNameSuffix      bool
	addressType            gateway.AddressType
}

var builtinClasses = getBuiltinClasses()

var classInfos = getClassInfos()

func getBuiltinClasses() map[gateway.ObjectName]gateway.GatewayController {
	res := map[gateway.ObjectName]gateway.GatewayController{
		gateway.ObjectName(features.GatewayAPIDefaultGatewayClass): gateway.GatewayController(features.ManagedGatewayController),
	}
	return res
}

func getClassInfos() map[gateway.GatewayController]classInfo {
	m := map[gateway.GatewayController]classInfo{
		gateway.GatewayController(features.ManagedGatewayController): {
			controller:          features.ManagedGatewayController,
			description:         "The default Dubbo GatewayClass",
			templates:           "gateway",
			defaultServiceType:  corev1.ServiceTypeLoadBalancer,
			addressType:         gateway.HostnameAddressType,
			controllerLabel:     constants.ManagedGatewayControllerLabel,
			supportsListenerSet: true,
		},
	}
	return m
}

// DeploymentController manages Gateway deployments
type DeploymentController struct {
	client          kube.Client
	clusterID       cluster.ID
	env             *model.Environment
	queue           controllers.Queue
	patcher         patcher
	gateways        kclient.Client[*gateway.Gateway]
	gatewayClasses  kclient.Client[*gateway.GatewayClass]
	clients         map[schema.GroupVersionResource]getter
	injectConfig    func() inject.Config
	deployments     kclient.Client[*appsv1.Deployment]
	services        kclient.Client[*corev1.Service]
	serviceAccounts kclient.Client[*corev1.ServiceAccount]
	namespaces      kclient.Client[*corev1.Namespace]
	tagWatcher      TagWatcher
	revision        string
	systemNamespace string
}

type patcher func(gvr schema.GroupVersionResource, name string, namespace string, data []byte, subresources ...string) error

type getter interface {
	Get(name, namespace string) controllers.Object
}

// UntypedWrapper wraps a typed reader to an untyped one
type UntypedWrapper[T controllers.ComparableObject] struct {
	reader kclient.Reader[T]
}

func NewUntypedWrapper[T controllers.ComparableObject](c kclient.Client[T]) getter {
	return UntypedWrapper[T]{c}
}

func (u UntypedWrapper[T]) Get(name, namespace string) controllers.Object {
	res := u.reader.Get(name, namespace)
	if controllers.IsNil(res) {
		return nil
	}
	return res
}

// NewDeploymentController creates a new deployment controller
func NewDeploymentController(
	client kube.Client,
	clusterID cluster.ID,
	env *model.Environment,
	webhookConfig func() inject.Config,
	injectionHandler func(fn func()),
	tw TagWatcher,
	revision string,
	systemNamespace string,
) *DeploymentController {
	filter := kclient.Filter{ObjectFilter: client.ObjectFilter()}
	gateways := kclient.NewFiltered[*gateway.Gateway](client, filter)
	gatewayClasses := kclient.New[*gateway.GatewayClass](client)

	dc := &DeploymentController{
		client:    client,
		clusterID: clusterID,
		clients:   map[schema.GroupVersionResource]getter{},
		env:       env,
		patcher: func(gvr schema.GroupVersionResource, name string, namespace string, data []byte, subresources ...string) error {
			c := client.Dynamic().Resource(gvr).Namespace(namespace)
			t := true
			_, err := c.Patch(context.Background(), name, types.ApplyPatchType, data, metav1.PatchOptions{
				Force:        &t,
				FieldManager: features.ManagedGatewayController,
			}, subresources...)
			return err
		},
		gateways:        gateways,
		gatewayClasses:  gatewayClasses,
		injectConfig:    webhookConfig,
		tagWatcher:      tw,
		revision:        revision,
		systemNamespace: systemNamespace,
	}

	dc.queue = controllers.NewQueue("gateway deployment",
		controllers.WithReconciler(dc.Reconcile),
		controllers.WithMaxAttempts(5))

	// Set up parent handler
	parentHandler := controllers.ObjectHandler(func(o controllers.Object) {
		// Enqueue parent Gateway when child resources change
		if gwName, ok := o.GetLabels()["gateway.networking.k8s.io/gateway-name"]; ok {
			dc.queue.Add(types.NamespacedName{
				Name:      gwName,
				Namespace: o.GetNamespace(),
			})
		}
	})

	dc.services = kclient.NewFiltered[*corev1.Service](client, filter)
	dc.services.AddEventHandler(parentHandler)
	dc.clients[gvr.Service] = NewUntypedWrapper(dc.services)

	dc.deployments = kclient.NewFiltered[*appsv1.Deployment](client, filter)
	dc.deployments.AddEventHandler(parentHandler)
	dc.clients[gvr.Deployment] = NewUntypedWrapper(dc.deployments)

	dc.serviceAccounts = kclient.NewFiltered[*corev1.ServiceAccount](client, filter)
	dc.serviceAccounts.AddEventHandler(parentHandler)
	dc.clients[gvr.ServiceAccount] = NewUntypedWrapper(dc.serviceAccounts)

	// Namespace is a cluster-scoped resource, use New instead of NewFiltered
	dc.namespaces = kclient.New[*corev1.Namespace](client)
	dc.namespaces.AddEventHandler(controllers.ObjectHandler(func(o controllers.Object) {
		for _, gw := range dc.gateways.List(o.GetName(), klabels.Everything()) {
			dc.queue.AddObject(gw)
		}
	}))

	// Gateway event handlers
	gateways.AddEventHandler(controllers.ObjectHandler(func(o controllers.Object) {
		dc.queue.AddObject(o)
	}))

	gatewayClasses.AddEventHandler(controllers.ObjectHandler(func(o controllers.Object) {
		for _, g := range dc.gateways.List(metav1.NamespaceAll, klabels.Everything()) {
			if string(g.Spec.GatewayClassName) == o.GetName() {
				dc.queue.AddObject(g)
			}
		}
	}))

	// On injection template change, requeue all gateways
	if injectionHandler != nil {
		injectionHandler(func() {
			for _, gw := range dc.gateways.List(metav1.NamespaceAll, klabels.Everything()) {
				dc.queue.AddObject(gw)
			}
		})
	}

	if dc.tagWatcher != nil {
		dc.tagWatcher.AddHandler(func(tags any) {
			dc.HandleTagChange(tags)
		})
	}

	return dc
}

func (d *DeploymentController) Run(stop <-chan struct{}) {
	kube.WaitForCacheSync(
		"deployment controller",
		stop,
		d.namespaces.HasSynced,
		d.deployments.HasSynced,
		d.services.HasSynced,
		d.serviceAccounts.HasSynced,
		d.gateways.HasSynced,
		d.gatewayClasses.HasSynced,
	)
	if d.tagWatcher != nil {
		// Start tagWatcher in background if it exists
		go d.tagWatcher.Run(stop)
	}
	d.queue.Run(stop)
	controllers.ShutdownAll(
		d.namespaces,
		d.deployments,
		d.services,
		d.serviceAccounts,
		d.gateways,
		d.gatewayClasses,
	)
}

// Reconcile reconciles a Gateway
func (d *DeploymentController) Reconcile(req types.NamespacedName) error {
	log := logger.WithLabels("gateway", req)

	gw := d.gateways.Get(req.Name, req.Namespace)
	if gw == nil {
		log.Debugf("gateway no longer exists")
		return nil
	}

	var controller gateway.GatewayController
	if gc := d.gatewayClasses.Get(string(gw.Spec.GatewayClassName), ""); gc != nil {
		controller = gc.Spec.ControllerName
	} else {
		if builtin, f := builtinClasses[gw.Spec.GatewayClassName]; f {
			controller = builtin
		}
	}

	ci, f := classInfos[controller]
	if !f {
		log.Debugf("skipping unknown controller %q", controller)
		return nil
	}
	log.Infof("reconciling gateway with controller %s", ci.controller)

	// Check revision
	if d.tagWatcher != nil && !d.tagWatcher.IsMine(gw.ObjectMeta) {
		log.Debugf("gateway is not for this revision, skipping")
		return nil
	}

	// Reconcile gateway
	return d.configureGateway(log, *gw, ci)
}

func (d *DeploymentController) configureGateway(log *dubbolog.Logger, gw gateway.Gateway, gi classInfo) error {
	if gi.templates == "" {
		log.Debugf("skip gateway class without template")
		return nil
	}

	if !IsManaged(&gw.Spec) {
		log.Debugf("skip disabled gateway")
		return nil
	}

	log.Infof("reconciling")

	defaultName := getDefaultName(gw.Name, &gw.Spec, gi.disableNameSuffix)
	serviceType := gi.defaultServiceType

	// Extract service ports from Gateway listeners
	ports := extractServicePorts(gw)

	input := TemplateInput{
		Gateway:         &gw,
		GatewayClass:    string(gw.Spec.GatewayClassName),
		DeploymentName:  defaultName,
		ServiceAccount:  defaultName,
		Ports:           ports,
		ServiceType:     serviceType,
		Revision:        d.revision,
		ControllerLabel: gi.controllerLabel,
	}

	log.Debugf("rendering template %q for gateway %s/%s", gi.templates, gw.Namespace, gw.Name)
	rendered, err := d.render(gi.templates, input)
	if err != nil {
		log.Errorf("error rendering templates: %v", err)
		return nil
	}

	if len(rendered) == 0 {
		log.Warnf("no resources rendered from template %q", gi.templates)
		return nil
	}

	log.Debugf("rendered %d resources from template", len(rendered))
	for i, t := range rendered {
		log.Debugf("applying resource %d/%d", i+1, len(rendered))
		if err := d.apply(gi.controller, t); err != nil {
			log.Errorf("apply failed for resource %d/%d: %v", i+1, len(rendered), err)
			return fmt.Errorf("apply failed: %v", err)
		}
	}

	log.Infof("gateway updated successfully")
	return nil
}

type TemplateInput struct {
	*gateway.Gateway
	GatewayClass    string
	DeploymentName  string
	ServiceAccount  string
	Ports           []corev1.ServicePort
	ServiceType     corev1.ServiceType
	Revision        string
	ControllerLabel string
}

func (d *DeploymentController) render(templateName string, mi TemplateInput) ([]string, error) {
	cfg := d.injectConfig()

	if cfg.Templates == nil {
		logger.Warnf("templates map is nil, webhook config may not be initialized yet")
		return nil, fmt.Errorf("templates map is nil")
	}

	// Log available templates for debugging
	availableTemplates := make([]string, 0, len(cfg.Templates))
	for k := range cfg.Templates {
		availableTemplates = append(availableTemplates, k)
	}
	logger.Debugf("looking for template %q, available templates: %v", templateName, availableTemplates)

	tmpl := cfg.Templates[templateName]
	if tmpl == nil {
		return nil, fmt.Errorf("no %q template defined, available templates: %v", templateName, availableTemplates)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, mi); err != nil {
		return nil, fmt.Errorf("template execution failed: %v", err)
	}

	result := buf.String()
	if result == "" {
		return nil, fmt.Errorf("template %q rendered empty output", templateName)
	}

	return splitYAML(result), nil
}

func (d *DeploymentController) apply(controller string, yml string) error {
	data := map[string]any{}
	err := yaml.Unmarshal([]byte(yml), &data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %v", err)
	}
	us := unstructured.Unstructured{Object: data}

	// set managed-by label
	clabel := strings.ReplaceAll(controller, "/", "-")
	err = unstructured.SetNestedField(us.Object, clabel, "metadata", "labels", "gateway.dubbo.apache.org/managed")
	if err != nil {
		return fmt.Errorf("failed to set managed label: %v", err)
	}

	gvk := us.GroupVersionKind()
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(gvk.Kind) + "s",
	}

	name := us.GetName()
	namespace := us.GetNamespace()
	logger.Debugf("applying %v %s/%s", gvk, namespace, name)

	canManage, resourceVersion := d.canManage(gvr, name, namespace)
	if !canManage {
		logger.Debugf("skipping %v/%v/%v, already managed", gvr, name, namespace)
		return nil
	}
	us.SetResourceVersion(resourceVersion)

	j, err := json.Marshal(us.Object)
	if err != nil {
		return fmt.Errorf("failed to marshal object: %v", err)
	}
	logger.Debugf("applying %v %s/%s: %s", gvk, namespace, name, string(j))
	if err := d.patcher(gvr, name, namespace, j); err != nil {
		return fmt.Errorf("patch %v/%v/%v: %v", gvk, namespace, name, err)
	}
	logger.Infof("successfully applied %v %s/%s", gvk, namespace, name)
	return nil
}

func (d *DeploymentController) canManage(gvr schema.GroupVersionResource, name, namespace string) (bool, string) {
	store, f := d.clients[gvr]
	if !f {
		logger.Warnf("unknown GVR %v", gvr)
		return true, ""
	}
	obj := store.Get(name, namespace)
	if obj == nil {
		return true, ""
	}
	_, managed := obj.GetLabels()["gateway.dubbo.apache.org/managed"]
	return managed, obj.GetResourceVersion()
}

func (d *DeploymentController) HandleTagChange(newTags any) {
	for _, gw := range d.gateways.List(metav1.NamespaceAll, klabels.Everything()) {
		d.queue.AddObject(gw)
	}
}

// IsManaged checks if a gateway should be managed
func IsManaged(gw *gateway.GatewaySpec) bool {
	// For now, always manage gateways that don't have explicit addresses
	if len(gw.Addresses) == 0 {
		return true
	}
	if len(gw.Addresses) > 1 {
		return false
	}
	if t := gw.Addresses[0].Type; t == nil || *t == gateway.IPAddressType {
		return true
	}
	return false
}

func getDefaultName(name string, kgw *gateway.GatewaySpec, disableNameSuffix bool) string {
	if disableNameSuffix {
		return name
	}
	return fmt.Sprintf("%v-%v", name, kgw.GatewayClassName)
}

func extractServicePorts(gw gateway.Gateway) []corev1.ServicePort {
	svcPorts := make([]corev1.ServicePort, 0, len(gw.Spec.Listeners)+1)
	tcp := "tcp"
	svcPorts = append(svcPorts, corev1.ServicePort{
		Name:        "status-port",
		Port:        int32(15021),
		AppProtocol: &tcp,
	})

	for i, l := range gw.Spec.Listeners {
		name := string(l.Name)
		if name == "" {
			name = fmt.Sprintf("%s-%d", strings.ToLower(string(l.Protocol)), i)
		}
		appProtocol := strings.ToLower(string(l.Protocol))
		svcPorts = append(svcPorts, corev1.ServicePort{
			Name:        name,
			Port:        l.Port,
			AppProtocol: &appProtocol,
		})
	}
	return svcPorts
}

// splitYAML splits a YAML document into individual resources
func splitYAML(yamlText string) []string {
	out := make([]string, 0)
	parts := strings.Split(yamlText, "\n---\n")

	for _, part := range parts {
		part := strings.TrimSpace(part)
		if len(part) > 0 {
			out = append(out, part)
		}
	}
	return out
}
