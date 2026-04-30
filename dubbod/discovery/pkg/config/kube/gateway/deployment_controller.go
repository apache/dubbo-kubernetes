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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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

const defaultDxgateGatewayName = "dxgate-gateway"

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
	configMaps      kclient.Client[*corev1.ConfigMap]
	httpRoutes      kclient.Client[*gateway.HTTPRoute]
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

	dc.configMaps = kclient.NewFiltered[*corev1.ConfigMap](client, filter)
	dc.configMaps.AddEventHandler(parentHandler)
	dc.clients[gvr.ConfigMap] = NewUntypedWrapper(dc.configMaps)

	dc.httpRoutes = kclient.NewFiltered[*gateway.HTTPRoute](client, filter)
	dc.httpRoutes.AddEventHandler(controllers.ObjectHandler(func(o controllers.Object) {
		for _, gw := range dc.gateways.List(metav1.NamespaceAll, klabels.Everything()) {
			dc.queue.AddObject(gw)
		}
	}))

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
		d.configMaps.HasSynced,
		d.httpRoutes.HasSynced,
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
		d.configMaps,
		d.httpRoutes,
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
	legacyName := getLegacyDefaultName(gw.Name, &gw.Spec, gi.disableNameSuffix)
	serviceType := gi.defaultServiceType

	// Extract service ports from Gateway listeners
	ports := extractServicePorts(gw)
	if len(ports) == 0 {
		log.Infof("gateway has no supported HTTP listeners, skipping dxgate deployment")
		return nil
	}
	bootstrapConfig, bootstrapConfigHash, err := d.buildDxgateBootstrapConfig(gw.Namespace, defaultName, ports)
	if err != nil {
		log.Errorf("failed building dxgate bootstrap config: %v", err)
		return err
	}

	input := TemplateInput{
		Gateway:             &gw,
		GatewayClass:        string(gw.Spec.GatewayClassName),
		DeploymentName:      defaultName,
		ServiceAccount:      defaultName,
		Ports:               ports,
		ServiceType:         serviceType,
		Revision:            d.revision,
		ControllerLabel:     gi.controllerLabel,
		BootstrapConfig:     bootstrapConfig,
		BootstrapConfigHash: bootstrapConfigHash,
		DxgateImage:         features.DxgateImage,
		SystemNamespace:     d.systemNamespace,
		ClusterID:           string(d.clusterID),
		DomainSuffix:        d.domainSuffix(),
	}

	log.Infof("desired dxgate deployment=%s/%s gatewayClass=%s serviceType=%s ports=%s image=%s",
		gw.Namespace, defaultName, input.GatewayClass, serviceType, formatGatewayServicePorts(ports), input.DxgateImage)

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

	if err := d.cleanupLegacyGatewayResources(context.TODO(), log, gw.Namespace, legacyName, defaultName); err != nil {
		log.Warnf("failed cleaning up legacy dxgate resources %s/%s: %v", gw.Namespace, legacyName, err)
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

	BootstrapConfig     string
	BootstrapConfigHash string
	RuntimeConfig       string
	RuntimeConfigHash   string
	DxgateImage         string
	SystemNamespace     string
	ClusterID           string
	DomainSuffix        string
}

type dxgateBootstrapConfig struct {
	XDSAddress    string   `json:"xds_address" yaml:"xds_address"`
	ListenerNames []string `json:"listener_names" yaml:"listener_names"`
	ClusterID     string   `json:"cluster_id" yaml:"cluster_id"`
	DNSDomain     string   `json:"dns_domain" yaml:"dns_domain"`
}

type dxgateRuntimeConfig struct {
	Version   string           `json:"version" yaml:"version"`
	Listeners []dxgateListener `json:"listeners" yaml:"listeners"`
	Clusters  []dxgateCluster  `json:"clusters" yaml:"clusters"`
	Secrets   []dxgateSecret   `json:"secrets" yaml:"secrets"`
}

type dxgateListener struct {
	Name         string              `json:"name" yaml:"name"`
	Bind         string              `json:"bind" yaml:"bind"`
	Protocol     string              `json:"protocol" yaml:"protocol"`
	VirtualHosts []dxgateVirtualHost `json:"virtual_hosts" yaml:"virtual_hosts"`
	TLSSecret    *string             `json:"tls_secret,omitempty" yaml:"tls_secret,omitempty"`
}

type dxgateVirtualHost struct {
	Name    string        `json:"name" yaml:"name"`
	Domains []string      `json:"domains" yaml:"domains"`
	Routes  []dxgateRoute `json:"routes" yaml:"routes"`
}

type dxgateRoute struct {
	Name             string                  `json:"name" yaml:"name"`
	Matches          []dxgateRouteMatch      `json:"matches" yaml:"matches"`
	WeightedClusters []dxgateWeightedCluster `json:"weighted_clusters" yaml:"weighted_clusters"`
}

type dxgateRouteMatch struct {
	Path    dxgatePathMatch     `json:"path" yaml:"path"`
	Headers []dxgateHeaderMatch `json:"headers" yaml:"headers"`
}

type dxgatePathMatch struct {
	Type  string `json:"type" yaml:"type"`
	Value string `json:"value" yaml:"value"`
}

type dxgateHeaderMatch struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

type dxgateWeightedCluster struct {
	Name   string `json:"name" yaml:"name"`
	Weight uint32 `json:"weight" yaml:"weight"`
}

type dxgateCluster struct {
	Name      string           `json:"name" yaml:"name"`
	Endpoints []dxgateEndpoint `json:"endpoints" yaml:"endpoints"`
}

type dxgateEndpoint struct {
	Address  string  `json:"address" yaml:"address"`
	Port     uint16  `json:"port" yaml:"port"`
	Healthy  bool    `json:"healthy" yaml:"healthy"`
	NodeName *string `json:"node_name,omitempty" yaml:"node_name,omitempty"`
}

type dxgateSecret struct {
	Name                string `json:"name" yaml:"name"`
	CertificateChainPEM string `json:"certificate_chain_pem" yaml:"certificate_chain_pem"`
	PrivateKeyPEM       string `json:"private_key_pem" yaml:"private_key_pem"`
}

func (d *DeploymentController) domainSuffix() string {
	if d.env != nil && d.env.DomainSuffix != "" {
		return d.env.DomainSuffix
	}
	return constants.DefaultClusterLocalDomain
}

func (d *DeploymentController) buildDxgateBootstrapConfig(namespace, serviceName string, ports []corev1.ServicePort) (string, string, error) {
	systemNamespace := d.systemNamespace
	if systemNamespace == "" {
		systemNamespace = constants.DubboSystemNamespace
	}
	return buildDxgateBootstrapConfig(
		fmt.Sprintf("http://dubbod.%s.svc:26010", systemNamespace),
		dxgateListenerNames(namespace, serviceName, d.domainSuffix(), ports),
		string(d.clusterID),
		d.domainSuffix(),
	)
}

func buildDxgateBootstrapConfig(xdsAddress string, listenerNames []string, clusterID, dnsDomain string) (string, string, error) {
	if xdsAddress == "" {
		return "", "", fmt.Errorf("dxgate bootstrap xDS address is empty")
	}
	if clusterID == "" {
		clusterID = string(cluster.ID("Kubernetes"))
	}
	if dnsDomain == "" {
		dnsDomain = constants.DefaultClusterLocalDomain
	}
	cfg := dxgateBootstrapConfig{
		XDSAddress:    xdsAddress,
		ListenerNames: listenerNames,
		ClusterID:     clusterID,
		DNSDomain:     dnsDomain,
	}
	rendered, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("marshal dxgate bootstrap config: %v", err)
	}
	rendered = append(rendered, '\n')
	sum := sha256.Sum256(rendered)
	return string(rendered), hex.EncodeToString(sum[:]), nil
}

func dxgateListenerNames(namespace, serviceName, domainSuffix string, ports []corev1.ServicePort) []string {
	if domainSuffix == "" {
		domainSuffix = constants.DefaultClusterLocalDomain
	}
	out := make([]string, 0, len(ports))
	for _, port := range ports {
		out = append(out, fmt.Sprintf("%s.%s.svc.%s:%d", serviceName, namespace, domainSuffix, port.Port))
	}
	sort.Strings(out)
	return out
}

func formatGatewayServicePorts(ports []corev1.ServicePort) string {
	out := make([]string, 0, len(ports))
	for _, port := range ports {
		out = append(out, fmt.Sprintf("%s:%d->%s", port.Name, port.Port, port.TargetPort.String()))
	}
	return strings.Join(out, ",")
}

func (d *DeploymentController) buildDxgateRuntimeConfig(gw gateway.Gateway) (string, string, error) {
	routes := d.httpRoutes.List(metav1.NamespaceAll, klabels.Everything())
	return buildDxgateRuntimeConfig(gw, routes, d.domainSuffix())
}

func buildDxgateRuntimeConfig(gw gateway.Gateway, routes []*gateway.HTTPRoute, domainSuffix string) (string, string, error) {
	if domainSuffix == "" {
		domainSuffix = constants.DefaultClusterLocalDomain
	}

	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Namespace != routes[j].Namespace {
			return routes[i].Namespace < routes[j].Namespace
		}
		return routes[i].Name < routes[j].Name
	})

	cfg := dxgateRuntimeConfig{
		Version: dxgateRuntimeVersion(gw, routes),
		Listeners: []dxgateListener{
			{
				Name:         "http-80",
				Bind:         "0.0.0.0:80",
				Protocol:     "http",
				VirtualHosts: []dxgateVirtualHost{},
			},
		},
		Clusters: []dxgateCluster{},
		Secrets:  []dxgateSecret{},
	}

	clusterNames := map[string]struct{}{}
	for _, hr := range routes {
		if !httpRouteReferencesGateway(hr, &gw) {
			continue
		}
		vh, clusters := buildDxgateVirtualHost(gw, hr, domainSuffix)
		if len(vh.Routes) == 0 {
			continue
		}
		cfg.Listeners[0].VirtualHosts = append(cfg.Listeners[0].VirtualHosts, vh)
		for _, cluster := range clusters {
			if _, found := clusterNames[cluster.Name]; found {
				continue
			}
			clusterNames[cluster.Name] = struct{}{}
			cfg.Clusters = append(cfg.Clusters, cluster)
		}
	}

	rendered, err := yaml.Marshal(cfg)
	if err != nil {
		return "", "", fmt.Errorf("marshal dxgate runtime config: %v", err)
	}
	sum := sha256.Sum256(rendered)
	return string(rendered), hex.EncodeToString(sum[:]), nil
}

func dxgateRuntimeVersion(gw gateway.Gateway, routes []*gateway.HTTPRoute) string {
	parts := []string{
		fmt.Sprintf("gateway/%s/%s/%s", gw.Namespace, gw.Name, gw.ResourceVersion),
	}
	for _, hr := range routes {
		if httpRouteReferencesGateway(hr, &gw) {
			parts = append(parts, fmt.Sprintf("httproute/%s/%s/%s", hr.Namespace, hr.Name, hr.ResourceVersion))
		}
	}
	return strings.Join(parts, ";")
}

func buildDxgateVirtualHost(gw gateway.Gateway, hr *gateway.HTTPRoute, domainSuffix string) (dxgateVirtualHost, []dxgateCluster) {
	vh := dxgateVirtualHost{
		Name:    fmt.Sprintf("%s-%s", hr.Namespace, hr.Name),
		Domains: dxgateRouteDomains(gw, hr),
		Routes:  []dxgateRoute{},
	}
	clusters := []dxgateCluster{}

	for ruleIdx, rule := range hr.Spec.Rules {
		for matchIdx, match := range dxgateMatches(rule.Matches) {
			route := dxgateRoute{
				Name:             fmt.Sprintf("%s-%s-%d-%d", hr.Namespace, hr.Name, ruleIdx, matchIdx),
				Matches:          []dxgateRouteMatch{match},
				WeightedClusters: []dxgateWeightedCluster{},
			}
			for backendIdx, backendRef := range rule.BackendRefs {
				if !isServiceBackend(backendRef) || backendRef.Port == nil {
					continue
				}
				weight := uint32(1)
				if backendRef.Weight != nil {
					if *backendRef.Weight == 0 {
						continue
					}
					weight = uint32(*backendRef.Weight)
				}

				backendNamespace := hr.Namespace
				if backendRef.Namespace != nil {
					backendNamespace = string(*backendRef.Namespace)
				}
				port := uint16(*backendRef.Port)
				clusterName := fmt.Sprintf("%s-%s-%d-%d", hr.Namespace, hr.Name, ruleIdx, backendIdx)

				route.WeightedClusters = append(route.WeightedClusters, dxgateWeightedCluster{
					Name:   clusterName,
					Weight: weight,
				})
				clusters = append(clusters, dxgateCluster{
					Name: clusterName,
					Endpoints: []dxgateEndpoint{
						{
							Address: fmt.Sprintf("%s.%s.svc.%s", backendRef.Name, backendNamespace, domainSuffix),
							Port:    port,
							Healthy: true,
						},
					},
				})
			}
			if len(route.WeightedClusters) > 0 {
				vh.Routes = append(vh.Routes, route)
			}
		}
	}
	return vh, clusters
}

func dxgateRouteDomains(gw gateway.Gateway, hr *gateway.HTTPRoute) []string {
	domains := map[string]struct{}{}
	for _, hostname := range hr.Spec.Hostnames {
		if hostname != "" {
			domains[string(hostname)] = struct{}{}
		}
	}
	if len(domains) == 0 {
		for _, listener := range gw.Spec.Listeners {
			if listener.Protocol != gateway.HTTPProtocolType || listener.Hostname == nil || *listener.Hostname == "" {
				continue
			}
			domains[string(*listener.Hostname)] = struct{}{}
		}
	}
	if len(domains) == 0 {
		return []string{"*"}
	}
	out := make([]string, 0, len(domains))
	for domain := range domains {
		out = append(out, domain)
	}
	sort.Strings(out)
	return out
}

func dxgateMatches(matches []gateway.HTTPRouteMatch) []dxgateRouteMatch {
	if len(matches) == 0 {
		return []dxgateRouteMatch{defaultDxgateRouteMatch()}
	}
	out := make([]dxgateRouteMatch, 0, len(matches))
	for _, match := range matches {
		out = append(out, dxgateRouteMatch{
			Path:    dxgatePath(match.Path),
			Headers: dxgateHeaders(match.Headers),
		})
	}
	return out
}

func defaultDxgateRouteMatch() dxgateRouteMatch {
	return dxgateRouteMatch{
		Path: dxgatePathMatch{
			Type:  "prefix",
			Value: "/",
		},
		Headers: []dxgateHeaderMatch{},
	}
}

func dxgatePath(path *gateway.HTTPPathMatch) dxgatePathMatch {
	if path == nil {
		return defaultDxgateRouteMatch().Path
	}
	value := "/"
	if path.Value != nil && *path.Value != "" {
		value = *path.Value
	}
	if path.Type != nil && *path.Type == gateway.PathMatchExact {
		return dxgatePathMatch{Type: "exact", Value: value}
	}
	return dxgatePathMatch{Type: "prefix", Value: value}
}

func dxgateHeaders(headers []gateway.HTTPHeaderMatch) []dxgateHeaderMatch {
	out := make([]dxgateHeaderMatch, 0, len(headers))
	for _, header := range headers {
		if header.Type != nil && *header.Type != gateway.HeaderMatchExact {
			continue
		}
		out = append(out, dxgateHeaderMatch{
			Name:  string(header.Name),
			Value: header.Value,
		})
	}
	return out
}

func httpRouteReferencesGateway(hr *gateway.HTTPRoute, gw *gateway.Gateway) bool {
	if hr == nil || gw == nil {
		return false
	}
	for _, parentRef := range hr.Spec.ParentRefs {
		if parentRef.Group != nil && string(*parentRef.Group) != gateway.GroupName {
			continue
		}
		if parentRef.Kind != nil && string(*parentRef.Kind) != "Gateway" {
			continue
		}
		namespace := hr.Namespace
		if parentRef.Namespace != nil {
			namespace = string(*parentRef.Namespace)
		}
		if string(parentRef.Name) != gw.Name || namespace != gw.Namespace {
			continue
		}
		if !parentRefMatchesHTTPListener(parentRef, gw) {
			continue
		}
		return true
	}
	return false
}

func parentRefMatchesHTTPListener(parentRef gateway.ParentReference, gw *gateway.Gateway) bool {
	for _, listener := range gw.Spec.Listeners {
		if listener.Protocol != gateway.HTTPProtocolType {
			continue
		}
		if parentRef.SectionName != nil && *parentRef.SectionName != listener.Name {
			continue
		}
		if parentRef.Port != nil && int32(*parentRef.Port) != listener.Port {
			continue
		}
		return true
	}
	return false
}

func isServiceBackend(ref gateway.HTTPBackendRef) bool {
	if ref.Group != nil && string(*ref.Group) != "" {
		return false
	}
	if ref.Kind != nil && string(*ref.Kind) != "" && string(*ref.Kind) != "Service" {
		return false
	}
	return true
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

func (d *DeploymentController) cleanupLegacyGatewayResources(ctx context.Context, log *dubbolog.Logger, namespace, legacyName, currentName string) error {
	if d.client == nil || legacyName == "" || legacyName == currentName {
		return nil
	}

	var errs []string
	for _, cleanup := range []struct {
		kind string
		name string
		fn   func(context.Context, string, string) error
	}{
		{kind: "ConfigMap", name: legacyName + "-bootstrap", fn: d.deleteManagedConfigMap},
		{kind: "ServiceAccount", name: legacyName, fn: d.deleteManagedServiceAccount},
		{kind: "Deployment", name: legacyName, fn: d.deleteManagedDeployment},
		{kind: "Service", name: legacyName, fn: d.deleteManagedService},
	} {
		if err := cleanup.fn(ctx, namespace, cleanup.name); err != nil {
			errs = append(errs, fmt.Sprintf("%s/%s: %v", cleanup.kind, cleanup.name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}

	log.Debugf("checked legacy dxgate resources %s/%s", namespace, legacyName)
	return nil
}

func (d *DeploymentController) deleteManagedConfigMap(ctx context.Context, namespace, name string) error {
	obj, err := d.client.Kube().CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !isManagedGatewayResource(obj.Labels) {
		return nil
	}
	err = d.client.Kube().CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (d *DeploymentController) deleteManagedServiceAccount(ctx context.Context, namespace, name string) error {
	obj, err := d.client.Kube().CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !isManagedGatewayResource(obj.Labels) {
		return nil
	}
	err = d.client.Kube().CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (d *DeploymentController) deleteManagedDeployment(ctx context.Context, namespace, name string) error {
	obj, err := d.client.Kube().AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !isManagedGatewayResource(obj.Labels) {
		return nil
	}
	err = d.client.Kube().AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (d *DeploymentController) deleteManagedService(ctx context.Context, namespace, name string) error {
	obj, err := d.client.Kube().CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !isManagedGatewayResource(obj.Labels) {
		return nil
	}
	err = d.client.Kube().CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func isManagedGatewayResource(labels map[string]string) bool {
	_, ok := labels["gateway.dubbo.apache.org/managed"]
	return ok
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

func getDefaultName(_ string, _ *gateway.GatewaySpec, _ bool) string {
	return defaultDxgateGatewayName
}

func getLegacyDefaultName(name string, kgw *gateway.GatewaySpec, disableNameSuffix bool) string {
	if disableNameSuffix {
		return name
	}
	return fmt.Sprintf("%v-%v", name, kgw.GatewayClassName)
}

func extractServicePorts(gw gateway.Gateway) []corev1.ServicePort {
	svcPorts := make([]corev1.ServicePort, 0, len(gw.Spec.Listeners))
	for i, l := range gw.Spec.Listeners {
		if l.Protocol != gateway.HTTPProtocolType {
			continue
		}
		name := string(l.Name)
		if name == "" {
			name = fmt.Sprintf("%s-%d", strings.ToLower(string(l.Protocol)), i)
		}
		appProtocol := strings.ToLower(string(l.Protocol))
		svcPorts = append(svcPorts, corev1.ServicePort{
			Name:        name,
			Port:        l.Port,
			TargetPort:  intstr.FromString("http"),
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
