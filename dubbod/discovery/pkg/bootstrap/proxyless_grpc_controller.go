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

package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	discoverymodel "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	pkgbootstrap "github.com/apache/dubbo-kubernetes/pkg/bootstrap"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	configlabels "github.com/apache/dubbo-kubernetes/pkg/config/labels"
	meshconfig "github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/dubboagency/grpcxds"
	kubelib "github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	pkgmodel "github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/spiffe"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"

	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/nodeagent/util"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/ca"
	pkiutil "github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/util"
	caserver "github.com/apache/dubbo-kubernetes/dubbod/security/pkg/server/ca"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

const (
	proxylessGRPCControllerName = "proxyless grpc workloads"
	serviceNodeSeparator        = "~"
)

var proxylessGRPCLog = log.RegisterScope("proxylessgrpc", "proxyless grpc workload controller")

type proxylessGRPCWorkloadController struct {
	server *Server
	pods   kclient.Client[*corev1.Pod]
	queue  controllers.Queue

	bundleWatcherID int32
	bundleWatcherCh chan struct{}

	timerMu sync.Mutex
	timers  map[types.NamespacedName]*time.Timer
}

func (s *Server) initProxylessGRPCWorkloads() error {
	if s.kubeClient == nil {
		return nil
	}

	controller := newProxylessGRPCWorkloadController(s)
	s.proxylessGRPCWorkloadController = controller
	s.addStartFunc(proxylessGRPCControllerName, func(stop <-chan struct{}) error {
		go controller.Run(stop)
		return nil
	})
	return nil
}

func newProxylessGRPCWorkloadController(s *Server) *proxylessGRPCWorkloadController {
	c := &proxylessGRPCWorkloadController{
		server: s,
		pods: kclient.NewFiltered[*corev1.Pod](s.kubeClient, kclient.Filter{
			ObjectFilter: s.kubeClient.ObjectFilter(),
		}),
		timers: make(map[types.NamespacedName]*time.Timer),
	}
	c.queue = controllers.NewQueue(proxylessGRPCControllerName,
		controllers.WithReconciler(func(key types.NamespacedName) error {
			return c.reconcile(key)
		}),
		controllers.WithMaxAttempts(5),
	)

	c.pods.AddEventHandler(controllers.FilteredObjectSpecHandler(c.queue.AddObject, func(o controllers.Object) bool {
		pod := controllers.Extract[*corev1.Pod](o)
		return shouldManageProxylessGRPCPod(pod)
	}))

	return c
}

func (c *proxylessGRPCWorkloadController) Run(stop <-chan struct{}) {
	c.pods.Start(stop)
	if !kubelib.WaitForCacheSync(proxylessGRPCControllerName, stop, c.pods.HasSynced) {
		c.queue.ShutDownEarly()
		return
	}

	c.bundleWatcherID, c.bundleWatcherCh = c.server.dubbodCertBundleWatcher.AddWatcher()
	defer c.server.dubbodCertBundleWatcher.RemoveWatcher(c.bundleWatcherID)
	defer c.stopAllTimers()

	c.enqueueAllPods()
	go c.watchBundleChanges(stop)
	c.queue.Run(stop)
}

func (c *proxylessGRPCWorkloadController) HasSynced() bool {
	return c.pods.HasSynced() && c.queue.HasSynced()
}

func (c *proxylessGRPCWorkloadController) watchBundleChanges(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case _, ok := <-c.bundleWatcherCh:
			if !ok {
				return
			}
			c.enqueueAllPods()
		}
	}
}

func (c *proxylessGRPCWorkloadController) enqueueAllPods() {
	for _, pod := range c.pods.List(metav1.NamespaceAll, labels.Everything()) {
		if shouldManageProxylessGRPCPod(pod) {
			c.queue.AddObject(pod)
		}
	}
}

func (c *proxylessGRPCWorkloadController) reconcile(key types.NamespacedName) error {
	pod := c.pods.Get(key.Name, key.Namespace)
	if pod == nil {
		c.clearRotation(key)
		return c.deleteSecret(key)
	}
	if !shouldManageProxylessGRPCPod(pod) || pod.DeletionTimestamp != nil {
		c.clearRotation(key)
		return c.deleteSecretForPod(pod)
	}
	secrets := c.server.kubeClient.Kube().CoreV1().Secrets(pod.Namespace)
	secretName := inject.ProxylessGRPCSecretNameForMeta(pod.ObjectMeta)
	current, err := secrets.Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		current = nil
	}

	desired, expireAt, err := c.buildSecret(pod, current)
	if err != nil {
		return err
	}

	if current == nil {
		if _, err := secrets.Create(context.Background(), desired, metav1.CreateOptions{}); err != nil {
			return err
		}
	} else if !reflect.DeepEqual(current.Data, desired.Data) || !reflect.DeepEqual(current.OwnerReferences, desired.OwnerReferences) {
		current.Data = desired.Data
		current.OwnerReferences = desired.OwnerReferences
		if _, err := secrets.Update(context.Background(), current, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	c.scheduleRotation(key, expireAt)
	return nil
}

const proxylessGRPCRuntimeConfigVersion = "dubbo.apache.org/proxyless-grpc/v1"

type proxylessGRPCWorkloadContext struct {
	node             *pkgmodel.Node
	nodeID           string
	podName          string
	podNamespace     string
	podIP            string
	serviceAccount   string
	trustDomain      string
	discoveryAddress string
}

type proxylessGRPCRuntimeConfig struct {
	Version      string                              `json:"version"`
	Mode         string                              `json:"mode"`
	Env          map[string]string                   `json:"env"`
	Bootstrap    proxylessGRPCBootstrapRuntimeConfig `json:"bootstrap"`
	Certificates proxylessGRPCCertRuntimeConfig      `json:"certificates"`
	Workload     proxylessGRPCWorkloadRuntimeConfig  `json:"workload"`
	Services     []proxylessGRPCServiceRuntimeConfig `json:"services,omitempty"`
	Routes       []proxylessGRPCRouteRuntimeConfig   `json:"routes,omitempty"`
}

type proxylessGRPCBootstrapRuntimeConfig struct {
	Path             string `json:"path"`
	DiscoveryAddress string `json:"discoveryAddress"`
	Resolver         string `json:"resolver"`
}

type proxylessGRPCCertRuntimeConfig struct {
	Provider   string `json:"provider"`
	Directory  string `json:"directory"`
	CertChain  string `json:"certChain"`
	PrivateKey string `json:"privateKey"`
	RootCert   string `json:"rootCert"`
}

type proxylessGRPCWorkloadRuntimeConfig struct {
	NodeID         string `json:"nodeId"`
	PodName        string `json:"podName"`
	Namespace      string `json:"namespace"`
	PodIP          string `json:"podIP"`
	ServiceAccount string `json:"serviceAccount"`
	TrustDomain    string `json:"trustDomain"`
	ClusterID      string `json:"clusterId"`
}

type proxylessGRPCServiceRuntimeConfig struct {
	Host      string                               `json:"host"`
	Namespace string                               `json:"namespace"`
	Name      string                               `json:"name"`
	Ports     []proxylessGRPCPortRuntimeConfig     `json:"ports,omitempty"`
	Endpoints []proxylessGRPCEndpointRuntimeConfig `json:"endpoints,omitempty"`
}

type proxylessGRPCPortRuntimeConfig struct {
	Name string `json:"name,omitempty"`
	Port int    `json:"port"`
}

type proxylessGRPCRouteRuntimeConfig struct {
	Host         string                                  `json:"host"`
	Port         int                                     `json:"port"`
	Destinations []proxylessGRPCDestinationRuntimeConfig `json:"destinations"`
}

type proxylessGRPCDestinationRuntimeConfig struct {
	Host      string                               `json:"host"`
	Subset    string                               `json:"subset,omitempty"`
	Weight    int                                  `json:"weight"`
	Endpoints []proxylessGRPCEndpointRuntimeConfig `json:"endpoints,omitempty"`
}

type proxylessGRPCEndpointRuntimeConfig struct {
	Address        string            `json:"address"`
	Port           uint32            `json:"port"`
	Labels         map[string]string `json:"labels,omitempty"`
	ServiceAccount string            `json:"serviceAccount,omitempty"`
	WorkloadName   string            `json:"workloadName,omitempty"`
	HealthStatus   string            `json:"healthStatus,omitempty"`
}

func (c *proxylessGRPCWorkloadController) buildSecret(pod *corev1.Pod, current *corev1.Secret) (*corev1.Secret, time.Time, error) {
	workload, err := c.buildWorkloadContext(pod)
	if err != nil {
		return nil, time.Time{}, err
	}

	bootstrapJSON, err := buildBootstrapJSON(workload)
	if err != nil {
		return nil, time.Time{}, err
	}

	certChain, keyPEM, rootCert, expireAt, reusedCert := reusableWorkloadCertificate(current)
	if !reusedCert {
		certChain, keyPEM, rootCert, expireAt, err = c.issueWorkloadCertificate(pod)
		if err != nil {
			return nil, time.Time{}, err
		}
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inject.ProxylessGRPCSecretNameForMeta(pod.ObjectMeta),
			Namespace: pod.Namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "v1",
				Kind:       "Pod",
				Name:       pod.Name,
				UID:        pod.UID,
			}},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			inject.ProxylessGRPCBootstrapFileName:      bootstrapJSON,
			constants.CertChainFilename:                certChain,
			constants.KeyFilename:                      keyPEM,
			constants.CACertNamespaceConfigMapDataName: rootCert,
		},
	}

	return secret, expireAt, nil
}

func reusableWorkloadCertificate(secret *corev1.Secret) ([]byte, []byte, []byte, time.Time, bool) {
	if secret == nil || len(secret.Data) == 0 {
		return nil, nil, nil, time.Time{}, false
	}
	certChain := secret.Data[constants.CertChainFilename]
	keyPEM := secret.Data[constants.KeyFilename]
	rootCert := secret.Data[constants.CACertNamespaceConfigMapDataName]
	if len(certChain) == 0 || len(keyPEM) == 0 || len(rootCert) == 0 {
		return nil, nil, nil, time.Time{}, false
	}
	expireAt, err := util.ParseCertAndGetExpiryTimestamp(certChain)
	if err != nil {
		return nil, nil, nil, time.Time{}, false
	}
	if time.Until(expireAt) <= workloadCertTTL.Get()/5 {
		return nil, nil, nil, time.Time{}, false
	}
	return certChain, keyPEM, rootCert, expireAt, true
}

func (c *proxylessGRPCWorkloadController) buildWorkloadContext(pod *corev1.Pod) (*proxylessGRPCWorkloadContext, error) {
	trustDomain := constants.DefaultClusterLocalDomain
	if meshCfg := c.server.environment.Mesh(); meshCfg != nil && meshCfg.GetTrustDomain() != "" {
		trustDomain = meshCfg.GetTrustDomain()
	}

	proxyConfig := meshconfig.DefaultProxyConfig()
	if meshCfg := c.server.environment.Mesh(); meshCfg != nil && meshCfg.GetDefaultConfig() != nil {
		proxyConfig = meshCfg.GetDefaultConfig()
	}

	serviceAccount := pod.Spec.ServiceAccountName
	if serviceAccount == "" {
		serviceAccount = "default"
	}
	domainSuffix := c.server.environment.DomainSuffix
	if domainSuffix == "" {
		domainSuffix = constants.DefaultClusterLocalDomain
	}
	podIP := pod.Status.PodIP
	if podIP == "" {
		podIP = "0.0.0.0"
	}

	nodeID := strings.Join([]string{
		string(pkgmodel.Proxyless),
		podIP,
		pod.Name + "." + pod.Namespace,
		fmt.Sprintf("%s.svc.%s", pod.Namespace, domainSuffix),
	}, serviceNodeSeparator)

	node, err := pkgbootstrap.GetNodeMetaData(pkgbootstrap.MetadataOptions{
		ID:          nodeID,
		InstanceIPs: []string{podIP},
		ProxyConfig: proxyConfig,
		Envs: []string{
			"DUBBO_META_GENERATOR=grpc",
			"DUBBO_META_CLUSTER_ID=" + constants.DefaultClusterName,
			"DUBBO_META_NAMESPACE=" + pod.Namespace,
			"DUBBO_META_MESH_ID=" + trustDomain,
			"TRUST_DOMAIN=" + trustDomain,
			"POD_NAME=" + pod.Name,
			"POD_NAMESPACE=" + pod.Namespace,
			"INSTANCE_IP=" + podIP,
			"SERVICE_ACCOUNT=" + serviceAccount,
		},
	})
	if err != nil {
		return nil, err
	}

	return &proxylessGRPCWorkloadContext{
		node:             node,
		nodeID:           nodeID,
		podName:          pod.Name,
		podNamespace:     pod.Namespace,
		podIP:            podIP,
		serviceAccount:   serviceAccount,
		trustDomain:      trustDomain,
		discoveryAddress: proxyConfig.GetDiscoveryAddress(),
	}, nil
}

func buildBootstrapJSON(workload *proxylessGRPCWorkloadContext) ([]byte, error) {
	bootstrapCfg, err := grpcxds.GenerateBootstrap(grpcxds.GenerateBootstrapOptions{
		Node:             workload.node,
		DiscoveryAddress: workload.discoveryAddress,
		CertDir:          inject.ProxylessXDSMountPath,
	})
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(bootstrapCfg, "", "  ")
}

func buildRuntimeConfigJSON(workload *proxylessGRPCWorkloadContext, services []proxylessGRPCServiceRuntimeConfig, routes []proxylessGRPCRouteRuntimeConfig) ([]byte, error) {
	cfg := proxylessGRPCRuntimeConfig{
		Version: proxylessGRPCRuntimeConfigVersion,
		Mode:    "proxyless-grpc",
		Env: map[string]string{
			"GRPC_XDS_BOOTSTRAP":                     inject.ProxylessGRPCBootstrapPath,
			inject.ProxylessGRPCConfigEnvName:        inject.ProxylessGRPCConfigPath,
			"GRPC_XDS_EXPERIMENTAL_SECURITY_SUPPORT": "true",
			"DUBBO_GRPC_XDS_CREDENTIALS":             "true",
			"DUBBO_GRPC_XDS_RESOLVER":                "xds:///",
			"DUBBO_META_GENERATOR":                   "grpc",
			"DUBBO_META_CLUSTER_ID":                  constants.DefaultClusterName,
			"DUBBO_META_NAMESPACE":                   workload.podNamespace,
			"DUBBO_META_MESH_ID":                     workload.trustDomain,
			"TRUST_DOMAIN":                           workload.trustDomain,
			"POD_NAME":                               workload.podName,
			"POD_NAMESPACE":                          workload.podNamespace,
			"INSTANCE_IP":                            workload.podIP,
			"SERVICE_ACCOUNT":                        workload.serviceAccount,
		},
		Bootstrap: proxylessGRPCBootstrapRuntimeConfig{
			Path:             inject.ProxylessGRPCBootstrapPath,
			DiscoveryAddress: workload.discoveryAddress,
			Resolver:         "xds:///",
		},
		Certificates: proxylessGRPCCertRuntimeConfig{
			Provider:   grpcxds.FileWatcherCertProviderName,
			Directory:  inject.ProxylessXDSMountPath,
			CertChain:  inject.ProxylessXDSMountPath + "/" + constants.CertChainFilename,
			PrivateKey: inject.ProxylessXDSMountPath + "/" + constants.KeyFilename,
			RootCert:   inject.ProxylessXDSMountPath + "/" + constants.CACertNamespaceConfigMapDataName,
		},
		Workload: proxylessGRPCWorkloadRuntimeConfig{
			NodeID:         workload.nodeID,
			PodName:        workload.podName,
			Namespace:      workload.podNamespace,
			PodIP:          workload.podIP,
			ServiceAccount: workload.serviceAccount,
			TrustDomain:    workload.trustDomain,
			ClusterID:      constants.DefaultClusterName,
		},
		Services: services,
		Routes:   routes,
	}
	return json.MarshalIndent(cfg, "", "  ")
}

func (c *proxylessGRPCWorkloadController) buildRuntimeTrafficConfig() ([]proxylessGRPCServiceRuntimeConfig, []proxylessGRPCRouteRuntimeConfig) {
	if c.server == nil || c.server.environment == nil {
		return nil, nil
	}
	push := c.server.environment.PushContext()
	if push == nil || !push.InitDone.Load() {
		return nil, nil
	}
	endpoints := c.server.environment.EndpointIndex
	services := push.GetAllServices()
	sort.Slice(services, func(i, j int) bool {
		return string(services[i].Hostname) < string(services[j].Hostname)
	})

	serviceConfigs := make([]proxylessGRPCServiceRuntimeConfig, 0, len(services))
	routeConfigs := make([]proxylessGRPCRouteRuntimeConfig, 0, len(services))
	for _, svc := range services {
		if svc == nil {
			continue
		}
		serviceConfigs = append(serviceConfigs, buildRuntimeServiceConfig(endpoints, svc))
		for _, port := range svc.Ports {
			if port == nil {
				continue
			}
			routeConfigs = append(routeConfigs, buildRuntimeRouteConfig(push, endpoints, svc, port.Port))
		}
	}
	return serviceConfigs, routeConfigs
}

func buildRuntimeServiceConfig(endpointIndex *discoverymodel.EndpointIndex, svc *discoverymodel.Service) proxylessGRPCServiceRuntimeConfig {
	cfg := proxylessGRPCServiceRuntimeConfig{
		Host:      string(svc.Hostname),
		Namespace: svc.Attributes.Namespace,
		Name:      svc.Attributes.Name,
	}
	for _, port := range svc.Ports {
		if port == nil {
			continue
		}
		cfg.Ports = append(cfg.Ports, proxylessGRPCPortRuntimeConfig{Name: port.Name, Port: port.Port})
		cfg.Endpoints = append(cfg.Endpoints, runtimeEndpointsForService(endpointIndex, svc, port.Port, nil)...)
	}
	sortRuntimeEndpoints(cfg.Endpoints)
	return cfg
}

func buildRuntimeRouteConfig(push *discoverymodel.PushContext, endpointIndex *discoverymodel.EndpointIndex, svc *discoverymodel.Service, port int) proxylessGRPCRouteRuntimeConfig {
	cfg := proxylessGRPCRouteRuntimeConfig{
		Host: string(svc.Hostname),
		Port: port,
	}
	vs := push.VirtualServiceForHost(svc.Hostname)
	if vs == nil || len(vs.Http) == 0 {
		cfg.Destinations = []proxylessGRPCDestinationRuntimeConfig{{
			Host:      string(svc.Hostname),
			Weight:    100,
			Endpoints: runtimeEndpointsForService(endpointIndex, svc, port, nil),
		}}
		return cfg
	}

	for _, httpRoute := range vs.Http {
		if httpRoute == nil {
			continue
		}
		for _, weighted := range httpRoute.Route {
			if weighted == nil {
				continue
			}
			targetHost := string(svc.Hostname)
			subset := ""
			if weighted.Destination != nil {
				if weighted.Destination.Host != "" {
					targetHost = weighted.Destination.Host
				}
				subset = weighted.Destination.Subset
			}
			targetSvc := push.ServiceForHostname(nil, host.Name(targetHost))
			if targetSvc == nil {
				targetSvc = svc
				targetHost = string(svc.Hostname)
			}
			selector, subsetFound := runtimeSubsetSelector(push, targetSvc.Attributes.Namespace, targetSvc.Hostname, subset)
			var endpoints []proxylessGRPCEndpointRuntimeConfig
			if subsetFound {
				endpoints = runtimeEndpointsForService(endpointIndex, targetSvc, port, selector)
			}
			cfg.Destinations = append(cfg.Destinations, proxylessGRPCDestinationRuntimeConfig{
				Host:      targetHost,
				Subset:    subset,
				Weight:    int(weighted.Weight),
				Endpoints: endpoints,
			})
		}
	}
	if len(cfg.Destinations) == 0 {
		cfg.Destinations = []proxylessGRPCDestinationRuntimeConfig{{
			Host:      string(svc.Hostname),
			Weight:    100,
			Endpoints: runtimeEndpointsForService(endpointIndex, svc, port, nil),
		}}
	}
	normalizeRuntimeRouteWeights(cfg.Destinations)
	return cfg
}

func runtimeSubsetSelector(push *discoverymodel.PushContext, namespace string, hostname host.Name, subset string) (configlabels.Instance, bool) {
	if subset == "" {
		return nil, true
	}
	rule := push.DestinationRuleForService(namespace, hostname)
	if rule == nil {
		return nil, false
	}
	for _, ss := range rule.Subsets {
		if ss.Name == subset {
			return configlabels.Instance(ss.Labels), true
		}
	}
	return nil, false
}

func normalizeRuntimeRouteWeights(destinations []proxylessGRPCDestinationRuntimeConfig) {
	hasPositiveWeight := false
	for i := range destinations {
		if destinations[i].Weight < 0 {
			destinations[i].Weight = 0
		}
		if destinations[i].Weight > 0 {
			hasPositiveWeight = true
		}
	}
	if hasPositiveWeight || len(destinations) == 0 {
		return
	}
	if len(destinations) == 1 {
		destinations[0].Weight = 100
		return
	}
	for i := range destinations {
		destinations[i].Weight = 1
	}
}

func runtimeEndpointsForService(endpointIndex *discoverymodel.EndpointIndex, svc *discoverymodel.Service, port int, selector configlabels.Instance) []proxylessGRPCEndpointRuntimeConfig {
	if endpointIndex == nil || svc == nil {
		return nil
	}
	portMap := map[string]int{}
	for _, servicePort := range svc.Ports {
		if servicePort != nil {
			portMap[servicePort.Name] = servicePort.Port
		}
	}
	shards, ok := endpointIndex.ShardsForService(string(svc.Hostname), svc.Attributes.Namespace)
	if !ok {
		return nil
	}
	byPort := shards.CopyEndpoints(portMap, sets.New(port))
	eps := byPort[port]
	out := make([]proxylessGRPCEndpointRuntimeConfig, 0, len(eps))
	for _, ep := range eps {
		if ep == nil || ep.FirstAddressOrNil() == "" {
			continue
		}
		if len(selector) > 0 && !selector.SubsetOf(ep.Labels) {
			continue
		}
		out = append(out, proxylessGRPCEndpointRuntimeConfig{
			Address:        ep.FirstAddressOrNil(),
			Port:           ep.EndpointPort,
			Labels:         copyStringMap(ep.Labels),
			ServiceAccount: ep.ServiceAccount,
			WorkloadName:   ep.WorkloadName,
			HealthStatus:   healthStatusString(ep.HealthStatus),
		})
	}
	sortRuntimeEndpoints(out)
	return out
}

func sortRuntimeEndpoints(endpoints []proxylessGRPCEndpointRuntimeConfig) {
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Address != endpoints[j].Address {
			return endpoints[i].Address < endpoints[j].Address
		}
		if endpoints[i].Port != endpoints[j].Port {
			return endpoints[i].Port < endpoints[j].Port
		}
		return endpoints[i].WorkloadName < endpoints[j].WorkloadName
	})
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func healthStatusString(status discoverymodel.HealthStatus) string {
	switch status {
	case discoverymodel.Healthy:
		return "HEALTHY"
	case discoverymodel.UnHealthy:
		return "UNHEALTHY"
	case discoverymodel.Draining:
		return "DRAINING"
	case discoverymodel.Terminating:
		return "TERMINATING"
	default:
		return "UNKNOWN"
	}
}

func (c *proxylessGRPCWorkloadController) issueWorkloadCertificate(pod *corev1.Pod) ([]byte, []byte, []byte, time.Time, error) {
	authority := c.activeAuthority()
	if authority == nil {
		return nil, nil, nil, time.Time{}, fmt.Errorf("workload certificate authority is not initialized")
	}

	trustDomain := constants.DefaultClusterLocalDomain
	if meshCfg := c.server.environment.Mesh(); meshCfg != nil && meshCfg.GetTrustDomain() != "" {
		trustDomain = meshCfg.GetTrustDomain()
	}

	serviceAccount := pod.Spec.ServiceAccountName
	if serviceAccount == "" {
		serviceAccount = "default"
	}
	identity := spiffe.Identity{
		TrustDomain:    trustDomain,
		Namespace:      pod.Namespace,
		ServiceAccount: serviceAccount,
	}

	csrPEM, keyPEM, err := pkiutil.GenCSR(pkiutil.CertOptions{
		Host:       identity.String(),
		RSAKeySize: 2048,
	})
	if err != nil {
		return nil, nil, nil, time.Time{}, err
	}

	respCertChain, err := authority.SignWithCertChain(csrPEM, ca.CertOpts{
		SubjectIDs: []string{identity.String()},
		TTL:        workloadCertTTL.Get(),
		ForCA:      false,
	})
	if err != nil {
		return nil, nil, nil, time.Time{}, err
	}

	certChain := concatPEM(respCertChain)
	expireAt, err := util.ParseCertAndGetExpiryTimestamp(certChain)
	if err != nil {
		return nil, nil, nil, time.Time{}, err
	}

	rootCert := authority.GetCAKeyCertBundle().GetRootCertPem()
	if len(rootCert) == 0 && len(respCertChain) > 0 {
		rootCert = []byte(respCertChain[len(respCertChain)-1])
	}

	return certChain, keyPEM, rootCert, expireAt, nil
}

func (c *proxylessGRPCWorkloadController) activeAuthority() caserver.CertificateAuthority {
	if c.server.RA != nil {
		return c.server.RA
	}
	return c.server.CA
}

func concatPEM(certs []string) []byte {
	if len(certs) == 0 {
		return nil
	}
	var b strings.Builder
	for i, cert := range certs {
		b.WriteString(cert)
		if i < len(certs)-1 && !strings.HasSuffix(cert, "\n") {
			b.WriteByte('\n')
		}
	}
	return []byte(b.String())
}

func shouldManageProxylessGRPCPod(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}
	templates := pod.Annotations[inject.ProxylessInjectTemplatesAnnoName]
	for _, templateName := range strings.Split(templates, ",") {
		if strings.TrimSpace(templateName) == inject.ProxylessGRPCTemplateName {
			return true
		}
	}
	for _, container := range pod.Spec.Containers {
		for _, envVar := range container.Env {
			if envVar.Name == "GRPC_XDS_BOOTSTRAP" && envVar.Value == inject.ProxylessGRPCBootstrapPath {
				return true
			}
		}
	}
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == inject.ProxylessXDSVolumeName && volume.Secret != nil {
			return true
		}
	}
	return false
}

func (c *proxylessGRPCWorkloadController) deleteSecret(key types.NamespacedName) error {
	err := c.server.kubeClient.Kube().CoreV1().Secrets(key.Namespace).Delete(context.Background(),
		inject.ProxylessGRPCSecretName(key.Name), metav1.DeleteOptions{})
	return controllers.IgnoreNotFound(err)
}

func (c *proxylessGRPCWorkloadController) deleteSecretForPod(pod *corev1.Pod) error {
	err := c.server.kubeClient.Kube().CoreV1().Secrets(pod.Namespace).Delete(context.Background(),
		inject.ProxylessGRPCSecretNameForMeta(pod.ObjectMeta), metav1.DeleteOptions{})
	return controllers.IgnoreNotFound(err)
}

func (c *proxylessGRPCWorkloadController) scheduleRotation(key types.NamespacedName, expireAt time.Time) {
	c.clearRotation(key)
	delay := workloadRotationDelay(time.Now(), expireAt)
	timer := time.AfterFunc(delay, func() {
		c.queue.Add(key)
	})
	c.timerMu.Lock()
	c.timers[key] = timer
	c.timerMu.Unlock()
}

func (c *proxylessGRPCWorkloadController) clearRotation(key types.NamespacedName) {
	c.timerMu.Lock()
	defer c.timerMu.Unlock()
	if timer, found := c.timers[key]; found {
		timer.Stop()
		delete(c.timers, key)
	}
}

func (c *proxylessGRPCWorkloadController) stopAllTimers() {
	c.timerMu.Lock()
	defer c.timerMu.Unlock()
	for key, timer := range c.timers {
		timer.Stop()
		delete(c.timers, key)
	}
}

func workloadRotationDelay(now, expireAt time.Time) time.Duration {
	lifetime := expireAt.Sub(now)
	if lifetime <= 0 {
		return time.Second
	}
	return lifetime * 4 / 5
}
