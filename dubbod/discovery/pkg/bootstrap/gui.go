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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"

	discoverymodel "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/gui"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/nodeagent/util"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/monitoring"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/version"
	dto "github.com/prometheus/client_model/go"
	sigsk8siogatewayapiapisv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type guiOverview struct {
	Product     string              `json:"product"`
	Version     string              `json:"version"`
	Cluster     string              `json:"clusterId"`
	Namespace   string              `json:"namespace"`
	PodName     string              `json:"podName,omitempty"`
	Mesh        guiOverviewMesh     `json:"mesh"`
	Server      guiOverviewServer   `json:"server"`
	Status      guiOverviewStatus   `json:"status"`
	Counts      guiOverviewCounts   `json:"counts"`
	ConfigKinds []guiConfigKind     `json:"configKinds"`
	Registries  []guiRegistry       `json:"registries"`
	Services    []guiService        `json:"services"`
	Instances   []guiDubbodInstance `json:"instances"`
	DataPlane   []guiWorkload       `json:"dataPlane"`
	// Total running pods per namespace that a sidecar is expected in, so the
	// console can show "5/6 injected" when one pod predates the label.
	DataPlanePods    map[string]int      `json:"dataPlanePods,omitempty"`
	Routes           []guiRoute          `json:"routes"`
	XDSClients       []guiXDSClient      `json:"xdsClients"`
	GatewayInstances []guiDubbodInstance `json:"gatewayInstances"`
	UpdatedAt        time.Time           `json:"updatedAt"`
}

// guiWorkload is one injected data plane pod, joined with the xDS stream it
// holds open against this control plane (if any).
type guiWorkload struct {
	Name           string     `json:"name"`
	Namespace      string     `json:"namespace"`
	IP             string     `json:"ip,omitempty"`
	Phase          string     `json:"phase"`
	Ready          bool       `json:"ready"`
	SidecarReady   bool       `json:"sidecarReady"`
	ServiceAccount string     `json:"serviceAccount,omitempty"`
	Image          string     `json:"image,omitempty"`
	Inbound        string     `json:"inbound,omitempty"`
	Upstream       string     `json:"upstream,omitempty"`
	XDSAddress     string     `json:"xdsAddress,omitempty"`
	Restarts       int32      `json:"restarts"`
	MTLSModes      []string   `json:"mtlsModes,omitempty"`
	CertExpiresAt  *time.Time `json:"certExpiresAt,omitempty"`
	CertRootActive bool       `json:"certRootActive"`
	ConfigError    string     `json:"configError,omitempty"`
	Connected      bool       `json:"connected"`
	NodeID         string     `json:"nodeId,omitempty"`
	NodeType       string     `json:"nodeType,omitempty"`
	ConnectedAt    *time.Time `json:"connectedAt,omitempty"`
	Watched        []string   `json:"watched,omitempty"`
}

// guiXDSClient is one live ADS stream. Only proxies that actually open a stream
// appear here — the inbound sidecar reads certificates off disk and never
// connects, so its absence is expected rather than a fault.
type guiXDSClient struct {
	NodeID      string    `json:"nodeId"`
	NodeType    string    `json:"nodeType,omitempty"`
	Peer        string    `json:"peer,omitempty"`
	ConnectedAt time.Time `json:"connectedAt"`
	Watched     []string  `json:"watched,omitempty"`
}

// guiRoute is one HTTPRoute as the control plane has it loaded: which parent it
// attaches to, and where each rule sends traffic. This is the call graph dubbod
// has programmed, not observed traffic — nothing in the xDS wire protocol
// reports which call actually happened.
type guiRoute struct {
	Name      string         `json:"name"`
	Namespace string         `json:"namespace"`
	Parents   []string       `json:"parents,omitempty"`
	Hostnames []string       `json:"hostnames,omitempty"`
	Rules     []guiRouteRule `json:"rules,omitempty"`
}

type guiRouteRule struct {
	Match    string            `json:"match,omitempty"`
	Backends []guiRouteBackend `json:"backends,omitempty"`
}

type guiRouteBackend struct {
	Name   string `json:"name"`
	Port   int32  `json:"port,omitempty"`
	Weight int32  `json:"weight,omitempty"`
}

type guiDubbodInstance struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	IP              string `json:"ip"`
	IsReady         bool   `json:"isReady"`
	GatewayClass    string `json:"gatewayClass,omitempty"`
	GatewayName     string `json:"gatewayName,omitempty"`
	ReadyReplicas   int32  `json:"readyReplicas,omitempty"`
	DesiredReplicas int32  `json:"desiredReplicas,omitempty"`
}

type guiOverviewMesh struct {
	TrustDomain      string `json:"trustDomain,omitempty"`
	RootNamespace    string `json:"rootNamespace,omitempty"`
	DiscoveryAddress string `json:"discoveryAddress,omitempty"`
}

type guiOverviewServer struct {
	GUIPath           string `json:"guiPath"`
	HTTPAddress       string `json:"httpAddress,omitempty"`
	GRPCAddress       string `json:"grpcAddress,omitempty"`
	SecureGRPCAddress string `json:"secureGrpcAddress,omitempty"`
	OverviewPath      string `json:"overviewPath"`
	MetricsPath       string `json:"metricsPath"`
	VersionPath       string `json:"versionPath"`
	ReadyPath         string `json:"readyPath,omitempty"`
}

type guiOverviewStatus struct {
	XDSServerReady  bool `json:"xdsServerReady"`
	CachesSynced    bool `json:"cachesSynced"`
	ServicesSynced  bool `json:"servicesSynced"`
	ConfigSynced    bool `json:"configSynced"`
	ProxylessSynced bool `json:"proxylessSynced"`
	InjectorReady   bool `json:"injectorReady"`
	ValidationReady bool `json:"validationReady"`
}

type guiOverviewCounts struct {
	Services               int `json:"services"`
	EndpointServices       int `json:"endpointServices"`
	XDSConnections         int `json:"xdsConnections"`
	Registries             int `json:"registries"`
	PeerAuthentications    int `json:"peerAuthentications"`
	RequestAuthentications int `json:"requestAuthentications"`
	AuthorizationPolicies  int `json:"authorizationPolicies"`
	HTTPRoutes             int `json:"httpRoutes"`
	GatewayClasses         int `json:"gatewayClasses"`
	Gateways               int `json:"gateways"`
}

type guiRegistry struct {
	Provider string `json:"provider"`
	Cluster  string `json:"cluster"`
	Synced   bool   `json:"synced"`
}

type guiConfigKind struct {
	Kind        string `json:"kind"`
	Count       int    `json:"count"`
	Description string `json:"description"`
}

type guiService struct {
	Name            string `json:"name"`
	Hostname        string `json:"hostname"`
	Namespace       string `json:"namespace"`
	Registry        string `json:"registry"`
	Ports           string `json:"ports"`
	Exposure        string `json:"exposure"`
	ServiceAccounts int    `json:"serviceAccounts"`
	DefaultAddress  string `json:"defaultAddress,omitempty"`
	MeshExternal    bool   `json:"meshExternal"`
	MTLSMode        string `json:"mtlsMode,omitempty"`
	MTLSFromPolicy  bool   `json:"mtlsFromPolicy"`
}

type guiMetricsResponse struct {
	Families  []guiMetricFamily `json:"families"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

type guiMetricFamily struct {
	Name    string            `json:"name"`
	Help    string            `json:"help,omitempty"`
	Type    string            `json:"type"`
	Metrics []guiMetricSample `json:"metrics"`
}

type guiMetricSample struct {
	Labels  map[string]string `json:"labels,omitempty"`
	Value   *float64          `json:"value,omitempty"`
	Count   *uint64           `json:"count,omitempty"`
	Sum     *float64          `json:"sum,omitempty"`
	Buckets []guiMetricBucket `json:"buckets,omitempty"`
}

type guiMetricBucket struct {
	LE    float64 `json:"le"`
	Count uint64  `json:"count"`
}

type guiLogsResponse struct {
	Kind      string      `json:"kind"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Pods      []guiPodLog `json:"pods"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

type guiPodLog struct {
	Name      string `json:"name"`
	Container string `json:"container"`
	Phase     string `json:"phase"`
	Ready     bool   `json:"ready"`
	Logs      string `json:"logs,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (s *Server) initGUI(args *DubboArgs) error {
	s.guiPath = gui.NormalizeBasePath(args.ServerOptions.GUIPath)

	handler, err := gui.NewHandler(s.guiPath, gui.Config{
		Product: version.Product,
	})
	if err != nil {
		return err
	}

	overviewPath := s.guiOverviewPath()
	logsPath := s.guiLogsPath()
	metricsAPIPath := s.guiMetricsPath()
	if s.guiPath == "/" {
		s.guiMux.HandleFunc(overviewPath, s.guiOverviewHandler)
		s.guiMux.HandleFunc(logsPath, s.guiLogsHandler)
		s.guiMux.HandleFunc(metricsAPIPath, s.guiMetricsHandler)
		s.guiMux.Handle("/", handler)
		return nil
	}

	s.guiMux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, s.guiPath+"/", http.StatusTemporaryRedirect)
	})
	s.guiMux.HandleFunc(s.guiPath, func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, s.guiPath+"/", http.StatusTemporaryRedirect)
	})
	s.guiMux.HandleFunc(overviewPath, s.guiOverviewHandler)
	s.guiMux.HandleFunc(logsPath, s.guiLogsHandler)
	s.guiMux.HandleFunc(metricsAPIPath, s.guiMetricsHandler)
	s.guiMux.Handle(s.guiPath+"/", handler)

	return nil
}

func (s *Server) initGUIServer(addr string) error {
	s.addStartFunc("gui", func(stop <-chan struct{}) error {
		if addr == "" {
			return nil
		}

		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("unable to listen on gui socket: %v", err)
		}
		s.guiAddr = listener.Addr().String()

		guiServer := &http.Server{
			Addr:        listener.Addr().String(),
			Handler:     s.guiMux,
			IdleTimeout: 90 * time.Second,
			ReadTimeout: 30 * time.Second,
		}

		go func() {
			log.Infof("starting GUI Server at %s", listener.Addr())
			if err := guiServer.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Errorf("error serving GUI Server: %v", err)
			}
		}()

		go func() {
			<-stop
			if err := guiServer.Close(); err != nil {
				log.Errorf("error closing GUI Server: %v", err)
			}
		}()

		return nil
	})

	return nil
}

func (s *Server) guiOverviewPath() string {
	return path.Join(s.guiPath, "api/overview")
}

func (s *Server) guiLogsPath() string {
	return path.Join(s.guiPath, "api/logs")
}

func (s *Server) guiMetricsPath() string {
	return path.Join(s.guiPath, "api/metrics")
}

func (s *Server) guiMetricsHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	families, err := monitoring.GetRegistry().Gather()
	if err != nil {
		writeGUIError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	response := guiMetricsResponse{
		Families:  make([]guiMetricFamily, 0, len(families)),
		UpdatedAt: time.Now().UTC(),
	}
	for _, family := range families {
		if family == nil {
			continue
		}
		out := guiMetricFamily{
			Name:    family.GetName(),
			Help:    family.GetHelp(),
			Type:    family.GetType().String(),
			Metrics: make([]guiMetricSample, 0, len(family.GetMetric())),
		}
		for _, metric := range family.GetMetric() {
			out.Metrics = append(out.Metrics, guiMetricSampleFrom(family.GetType(), metric))
		}
		response.Families = append(response.Families, out)
	}

	encoder := json.NewEncoder(writer)
	_ = encoder.Encode(response)
}

func guiMetricSampleFrom(kind dto.MetricType, metric *dto.Metric) guiMetricSample {
	sample := guiMetricSample{}
	if labels := metric.GetLabel(); len(labels) > 0 {
		sample.Labels = make(map[string]string, len(labels))
		for _, pair := range labels {
			sample.Labels[pair.GetName()] = pair.GetValue()
		}
	}

	floatPtr := func(v float64) *float64 { return &v }
	uintPtr := func(v uint64) *uint64 { return &v }

	switch kind {
	case dto.MetricType_COUNTER:
		sample.Value = floatPtr(metric.GetCounter().GetValue())
	case dto.MetricType_GAUGE:
		sample.Value = floatPtr(metric.GetGauge().GetValue())
	case dto.MetricType_UNTYPED:
		sample.Value = floatPtr(metric.GetUntyped().GetValue())
	case dto.MetricType_HISTOGRAM:
		histogram := metric.GetHistogram()
		sample.Count = uintPtr(histogram.GetSampleCount())
		sample.Sum = floatPtr(histogram.GetSampleSum())
		sample.Buckets = make([]guiMetricBucket, 0, len(histogram.GetBucket()))
		for _, bucket := range histogram.GetBucket() {
			sample.Buckets = append(sample.Buckets, guiMetricBucket{
				LE:    bucket.GetUpperBound(),
				Count: bucket.GetCumulativeCount(),
			})
		}
	case dto.MetricType_SUMMARY:
		summary := metric.GetSummary()
		sample.Count = uintPtr(summary.GetSampleCount())
		sample.Sum = floatPtr(summary.GetSampleSum())
	}
	return sample
}

func (s *Server) guiOverviewHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(s.buildGUIOverview())
}

func (s *Server) guiLogsHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	if s.kubeClient == nil {
		writeGUIError(writer, http.StatusServiceUnavailable, "kubernetes client is unavailable")
		return
	}

	kind := strings.TrimSpace(request.URL.Query().Get("kind"))
	namespace := strings.TrimSpace(request.URL.Query().Get("namespace"))
	name := strings.TrimSpace(request.URL.Query().Get("name"))
	tailLines := guiLogTailLines(request.URL.Query().Get("tail"))

	var response guiLogsResponse
	var err error
	switch kind {
	case "dubbod":
		if namespace == "" {
			namespace = s.namespace
		}
		if name == "" {
			name = "dubbod"
		}
		response, err = s.deploymentLogs(request.Context(), "dubbod", namespace, name, "execute", tailLines)
	case "gateway":
		if namespace == "" || name == "" {
			writeGUIError(writer, http.StatusBadRequest, "gateway logs require namespace and name")
			return
		}
		response, err = s.deploymentLogs(request.Context(), "gateway", namespace, name, "dxgate", tailLines)
	default:
		writeGUIError(writer, http.StatusBadRequest, "unknown log kind")
		return
	}
	if err != nil {
		writeGUIError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(response)
}

func writeGUIError(writer http.ResponseWriter, status int, message string) {
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]string{"error": message})
}

func guiLogTailLines(raw string) int64 {
	const (
		defaultTailLines int64 = 200
		maxTailLines     int64 = 2000
	)
	if raw == "" {
		return defaultTailLines
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return defaultTailLines
	}
	if n > maxTailLines {
		return maxTailLines
	}
	return n
}

func (s *Server) deploymentLogs(ctx context.Context, kind, namespace, name, preferredContainer string, tailLines int64) (guiLogsResponse, error) {
	deployment, err := s.kubeClient.Kube().AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return guiLogsResponse{}, fmt.Errorf("get deployment %s/%s: %v", namespace, name, err)
	}

	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return guiLogsResponse{}, fmt.Errorf("build pod selector for deployment %s/%s: %v", namespace, name, err)
	}
	if selector.Empty() {
		selector = klabels.SelectorFromSet(deployment.Spec.Template.Labels)
	}

	pods, err := s.kubeClient.Kube().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return guiLogsResponse{}, fmt.Errorf("list pods for deployment %s/%s: %v", namespace, name, err)
	}
	sort.SliceStable(pods.Items, func(i, j int) bool {
		return pods.Items[i].Name < pods.Items[j].Name
	})

	out := guiLogsResponse{
		Kind:      kind,
		Name:      name,
		Namespace: namespace,
		Pods:      make([]guiPodLog, 0, len(pods.Items)),
		UpdatedAt: time.Now().UTC(),
	}
	for _, pod := range pods.Items {
		out.Pods = append(out.Pods, s.podLogs(ctx, pod, preferredContainer, tailLines)...)
	}
	return out, nil
}

func (s *Server) podLogs(ctx context.Context, pod corev1.Pod, preferredContainer string, tailLines int64) []guiPodLog {
	containers := guiLogContainers(pod, preferredContainer)
	out := make([]guiPodLog, 0, len(containers))
	for _, container := range containers {
		entry := guiPodLog{
			Name:      pod.Name,
			Container: container,
			Phase:     string(pod.Status.Phase),
			Ready:     podReady(pod),
		}
		raw, err := s.kubeClient.Kube().CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
			Container:  container,
			TailLines:  &tailLines,
			Timestamps: true,
		}).DoRaw(ctx)
		if err != nil {
			entry.Error = err.Error()
		} else {
			entry.Logs = string(raw)
		}
		out = append(out, entry)
	}
	return out
}

func guiLogContainers(pod corev1.Pod, preferredContainer string) []string {
	if preferredContainer != "" {
		for _, container := range pod.Spec.Containers {
			if container.Name == preferredContainer {
				return []string{preferredContainer}
			}
		}
	}
	out := make([]string, 0, len(pod.Spec.Containers))
	for _, container := range pod.Spec.Containers {
		out = append(out, container.Name)
	}
	return out
}

func podReady(pod corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (s *Server) buildGUIOverview() guiOverview {
	meshOverview := guiOverviewMesh{}
	if meshConfig := s.environment.Mesh(); meshConfig != nil {
		meshOverview.TrustDomain = meshConfig.GetTrustDomain()
		meshOverview.RootNamespace = meshConfig.GetRootNamespace()
		if host, port, err := s.environment.GetDiscoveryAddress(); err == nil {
			meshOverview.DiscoveryAddress = string(host) + ":" + port
		}
	}

	dataPlane, dataPlanePods := s.buildGUIDataPlane()

	registries := s.buildGUIRegistries()
	configKinds := s.buildGUIConfigKinds()
	services := s.buildGUIServices()
	instances := s.buildGUIDubbodInstances()

	readyPath := ""
	if s.httpsAddr != "" {
		readyPath = "https://" + localLinkAddress(s.httpsAddr) + "/ready"
	}
	overviewPath := s.guiOverviewPath()
	if s.guiAddr != "" {
		overviewPath = buildLocalHTTPURL(s.guiAddr, s.guiOverviewPath())
	}
	metricsURL := metricsPath
	if s.httpAddr != "" {
		metricsURL = buildLocalHTTPURL(s.httpAddr, metricsPath)
	}
	versionURL := versionPath
	if s.httpAddr != "" {
		versionURL = buildLocalHTTPURL(s.httpAddr, versionPath)
	}

	return guiOverview{
		Product:   version.Product,
		Version:   version.Info.String(),
		Cluster:   string(s.clusterID),
		Namespace: s.namespace,
		PodName:   s.podName,
		Mesh:      meshOverview,
		Server: guiOverviewServer{
			GUIPath:           s.guiPath,
			HTTPAddress:       s.guiAddr,
			GRPCAddress:       s.grpcAddress,
			SecureGRPCAddress: s.secureGrpcAddress,
			OverviewPath:      overviewPath,
			MetricsPath:       metricsURL,
			VersionPath:       versionURL,
			ReadyPath:         readyPath,
		},
		Status: guiOverviewStatus{
			XDSServerReady:  s.XDSServer.IsServerReady(),
			CachesSynced:    s.cachesSynced(),
			ServicesSynced:  s.ServiceController().HasSynced(),
			ConfigSynced:    s.configController != nil && s.configController.HasSynced(),
			ProxylessSynced: s.proxylessGRPCWorkloadsSynced(),
			InjectorReady:   s.kubeClient == nil || s.readinessFlags.InjectorReady.Load(),
			ValidationReady: s.kubeClient == nil || s.readinessFlags.configValidationReady.Load(),
		},
		Counts: guiOverviewCounts{
			Services:               len(services),
			EndpointServices:       len(s.environment.EndpointIndex.AllServices()),
			XDSConnections:         len(s.XDSServer.AllClients()),
			Registries:             len(registries),
			PeerAuthentications:    s.countConfigs(gvk.PeerAuthentication),
			RequestAuthentications: s.countConfigs(gvk.RequestAuthentication),
			AuthorizationPolicies:  s.countConfigs(gvk.AuthorizationPolicy),
			HTTPRoutes:             s.countConfigs(gvk.HTTPRoute),
			GatewayClasses:         s.countConfigs(gvk.GatewayClass),
			Gateways:               s.countConfigs(gvk.KubernetesGateway),
		},
		ConfigKinds:      configKinds,
		Registries:       registries,
		Services:         services,
		Instances:        instances,
		Routes:           s.buildGUIRoutes(),
		XDSClients:       s.buildGUIXDSClients(),
		DataPlane:        dataPlane,
		DataPlanePods:    dataPlanePods,
		GatewayInstances: s.buildGUIGatewayInstances(),
		UpdatedAt:        time.Now().UTC(),
	}
}

const (
	// guiDubbodPodSelector matches the label the dubbod chart puts on control
	// plane pods (manifests/charts/dubbod/templates/deployment.yaml).
	guiDubbodPodSelector = "app=dubbod"

	// Gateway deployments provisioned by dubbod carry these labels; the same
	// pair identifies gateway pods so the data plane listing can skip them.
	guiGatewayNameLabel = "app.kubernetes.io/name"
	guiGatewayNameValue = "dxgate"
	guiGatewaySelector  = guiGatewayNameLabel + "=" + guiGatewayNameValue + ",app.kubernetes.io/managed-by=dubbod"

	// What the inbound sidecar enforces when no PeerAuthentication applies.
	grpcInboundFallbackMTLSMode = "PERMISSIVE"
)

func (s *Server) buildGUIDubbodInstances() []guiDubbodInstance {
	instances := make([]guiDubbodInstance, 0)

	// Off-cluster runs (`go run ./dubbod/discovery/cmd`) have no pods to list, so
	// report this process as-is rather than inventing an address for it.
	if s.kubeClient == nil {
		return append(instances, guiDubbodInstance{
			Name:      s.podName,
			Namespace: s.namespace,
			IsReady:   true,
		})
	}

	pods, err := s.kubeClient.Kube().CoreV1().Pods(s.namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: guiDubbodPodSelector,
	})
	if err != nil {
		log.Warnf("gui: listing control plane pods in %s: %v", s.namespace, err)
		return instances
	}

	for _, pod := range pods.Items {
		instances = append(instances, guiDubbodInstance{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			IP:        pod.Status.PodIP,
			IsReady:   podReady(pod),
		})
	}

	sort.SliceStable(instances, func(i, j int) bool { return instances[i].Name < instances[j].Name })

	return instances
}

// xdsTypeShortName turns an xDS type URL into the three-letter name operators
// actually use. Unknown URLs fall back to their last path segment rather than
// being dropped, so a new resource type shows up instead of silently vanishing.
func xdsTypeShortName(typeURL string) string {
	switch {
	case strings.HasSuffix(typeURL, ".Cluster"):
		return "CDS"
	case strings.HasSuffix(typeURL, ".ClusterLoadAssignment"):
		return "EDS"
	case strings.HasSuffix(typeURL, ".Listener"):
		return "LDS"
	case strings.HasSuffix(typeURL, ".RouteConfiguration"):
		return "RDS"
	case strings.HasSuffix(typeURL, ".Secret"):
		return "SDS"
	}
	if idx := strings.LastIndex(typeURL, "."); idx >= 0 && idx+1 < len(typeURL) {
		return typeURL[idx+1:]
	}
	return typeURL
}

func (s *Server) buildGUIXDSClients() []guiXDSClient {
	clients := make([]guiXDSClient, 0)
	for _, conn := range s.XDSServer.AllClients() {
		proxy := conn.Proxy()
		if proxy == nil {
			continue
		}
		entry := guiXDSClient{
			NodeID:      proxy.ID,
			NodeType:    string(proxy.Type),
			Peer:        conn.Peer(),
			ConnectedAt: conn.ConnectedAt().UTC(),
		}
		proxy.RLock()
		for typeURL := range proxy.WatchedResources {
			entry.Watched = append(entry.Watched, xdsTypeShortName(typeURL))
		}
		proxy.RUnlock()
		sort.Strings(entry.Watched)
		clients = append(clients, entry)
	}
	sort.SliceStable(clients, func(i, j int) bool { return clients[i].NodeID < clients[j].NodeID })
	return clients
}

// guiXDSClientsByIP indexes live ADS streams by every IP their node reported, so
// a pod row can be joined to the stream it actually holds open.
func (s *Server) guiXDSClientsByIP() map[string]guiWorkload {
	byIP := make(map[string]guiWorkload)
	for _, conn := range s.XDSServer.AllClients() {
		proxy := conn.Proxy()
		if proxy == nil {
			continue
		}

		entry := guiWorkload{
			Connected: true,
			NodeID:    proxy.ID,
			NodeType:  string(proxy.Type),
		}
		if at := conn.ConnectedAt(); !at.IsZero() {
			utc := at.UTC()
			entry.ConnectedAt = &utc
		}

		proxy.RLock()
		for typeURL := range proxy.WatchedResources {
			entry.Watched = append(entry.Watched, xdsTypeShortName(typeURL))
		}
		proxy.RUnlock()
		sort.Strings(entry.Watched)

		for _, ip := range proxy.IPAddresses {
			if ip != "" {
				byIP[ip] = entry
			}
		}
	}
	return byIP
}

// guiSidecarContainer returns the dxplane inbound sidecar in a pod, or nil when
// the pod is not part of the data plane.
func guiSidecarContainer(pod corev1.Pod) *corev1.Container {
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == inject.ProxylessGRPCInboundContainerName {
			return &pod.Spec.Containers[i]
		}
	}
	return nil
}

// guiSidecarAddresses reads the listener and upstream dxplane was started with.
// The sidecar takes both as `--listen`/`--upstream` flags, so the running
// configuration is readable from the pod spec without contacting the pod.
func guiSidecarAddresses(container corev1.Container) (inbound, upstream string) {
	args := container.Args
	for i := 0; i < len(args); i++ {
		var flag, value string
		if eq := strings.Index(args[i], "="); eq > 0 {
			flag, value = args[i][:eq], args[i][eq+1:]
		} else if i+1 < len(args) {
			flag, value = args[i], args[i+1]
		} else {
			continue
		}
		switch flag {
		case "--listen", "-listen":
			inbound = value
		case "--upstream", "-upstream":
			upstream = value
		}
	}
	if inbound == "" {
		for _, port := range container.Ports {
			if port.ContainerPort > 0 {
				inbound = ":" + strconv.Itoa(int(port.ContainerPort))
				break
			}
		}
	}
	return inbound, upstream
}

func guiContainerEnv(container corev1.Container, name string) string {
	for _, env := range container.Env {
		if env.Name == name {
			return env.Value
		}
	}
	return ""
}

// guiActiveRootCert mirrors proxylessGRPCWorkloadController.activeRootCert so
// the console compares workload secrets against the same root the issuer uses.
func (s *Server) guiActiveRootCert() []byte {
	authority := s.RA
	if authority == nil {
		if s.CA == nil {
			return nil
		}
		if s.CA.GetCAKeyCertBundle() == nil {
			return nil
		}
		return s.CA.GetCAKeyCertBundle().GetRootCertPem()
	}
	if authority.GetCAKeyCertBundle() == nil {
		return nil
	}
	return authority.GetCAKeyCertBundle().GetRootCertPem()
}

// guiWorkloadSecretState reads the per-workload secret dubbod itself generates.
// That secret holds the exact runtime config and certificate the sidecar is
// running with, so the mTLS mode and certificate expiry reported here are what
// is actually in force — not a restatement of the policy dubbod intended.
func (s *Server) guiWorkloadSecretState(ctx context.Context, pod corev1.Pod, activeRoot []byte) (modes []string, expiresAt *time.Time, rootActive bool, problem string) {
	name := inject.ProxylessGRPCSecretNameForMeta(pod.ObjectMeta)
	secret, err := s.kubeClient.Kube().CoreV1().Secrets(pod.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, false, fmt.Sprintf("workload secret %s not readable: %v", name, err)
	}

	if raw := secret.Data[inject.ProxylessGRPCConfigFileName]; len(raw) > 0 {
		var runtime struct {
			Services []struct {
				Ports []struct {
					MTLSMode string `json:"mtlsMode"`
				} `json:"ports"`
			} `json:"services"`
		}
		if err := json.Unmarshal(raw, &runtime); err != nil {
			problem = fmt.Sprintf("runtime config in %s is not parseable: %v", name, err)
		} else {
			seen := sets.New[string]()
			for _, svc := range runtime.Services {
				for _, port := range svc.Ports {
					if mode := strings.TrimSpace(port.MTLSMode); mode != "" {
						seen.Insert(mode)
					}
				}
			}
			modes = seen.UnsortedList()
			sort.Strings(modes)
		}
	} else {
		problem = fmt.Sprintf("workload secret %s carries no runtime config", name)
	}

	if chain := secret.Data[constants.CertChainFilename]; len(chain) > 0 {
		if at, err := util.ParseCertAndGetExpiryTimestamp(chain); err == nil {
			utc := at.UTC()
			expiresAt = &utc
		} else if problem == "" {
			problem = fmt.Sprintf("workload certificate is not parseable: %v", err)
		}
	} else if problem == "" {
		problem = "workload secret carries no certificate chain"
	}

	// A workload still holding a superseded root will keep failing handshakes
	// once the old root is retired, so surface the mismatch rather than the
	// reassuring "cert is valid until X".
	rootActive = len(activeRoot) > 0 && bytes.Equal(secret.Data[constants.CACertNamespaceConfigMapDataName], activeRoot)

	return modes, expiresAt, rootActive, problem
}

// buildGUIDataPlane lists the pods carrying the proxyless gRPC sidecar and joins
// each to its ADS stream. Gateway pods are excluded — they are reported
// separately as the external data plane.
func (s *Server) buildGUIDataPlane() ([]guiWorkload, map[string]int) {
	workloads := make([]guiWorkload, 0)
	candidates := make(map[string]int)
	if s.kubeClient == nil {
		return workloads, candidates
	}

	pods, err := s.kubeClient.Kube().CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Warnf("gui: listing data plane pods: %v", err)
		return workloads, candidates
	}

	clients := s.guiXDSClientsByIP()
	activeRoot := s.guiActiveRootCert()
	for _, pod := range pods.Items {
		if pod.Labels[guiGatewayNameLabel] == guiGatewayNameValue {
			continue
		}

		// Finished pods are not part of the running data plane and would skew
		// the injected ratio.
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}
		candidates[pod.Namespace]++

		sidecar := guiSidecarContainer(pod)
		if sidecar == nil {
			continue
		}

		sidecarReady := false
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == sidecar.Name {
				sidecarReady = status.Ready
			}
		}

		inbound, upstream := guiSidecarAddresses(*sidecar)
		workload := guiWorkload{
			Name:           pod.Name,
			Namespace:      pod.Namespace,
			IP:             pod.Status.PodIP,
			Phase:          string(pod.Status.Phase),
			Ready:          podReady(pod),
			SidecarReady:   sidecarReady,
			ServiceAccount: pod.Spec.ServiceAccountName,
			Image:          sidecar.Image,
			Inbound:        inbound,
			Upstream:       upstream,
			XDSAddress:     guiContainerEnv(*sidecar, "XDS_ADDRESS"),
		}
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == sidecar.Name {
				workload.Restarts = status.RestartCount
			}
		}
		workload.MTLSModes, workload.CertExpiresAt, workload.CertRootActive, workload.ConfigError =
			s.guiWorkloadSecretState(context.TODO(), pod, activeRoot)
		if client, ok := clients[pod.Status.PodIP]; ok {
			workload.Connected = client.Connected
			workload.NodeID = client.NodeID
			workload.NodeType = client.NodeType
			workload.ConnectedAt = client.ConnectedAt
			workload.Watched = client.Watched
		}
		workloads = append(workloads, workload)
	}

	sort.SliceStable(workloads, func(i, j int) bool {
		if workloads[i].Namespace != workloads[j].Namespace {
			return workloads[i].Namespace < workloads[j].Namespace
		}
		return workloads[i].Name < workloads[j].Name
	})

	// Only namespaces that actually run part of the data plane are interesting;
	// the rest of the cluster is not something dubbod is failing to inject.
	injectedNamespaces := sets.New[string]()
	for _, workload := range workloads {
		injectedNamespaces.Insert(workload.Namespace)
	}
	for namespace := range candidates {
		if !injectedNamespaces.Contains(namespace) {
			delete(candidates, namespace)
		}
	}

	return workloads, candidates
}

func (s *Server) buildGUIGatewayInstances() []guiDubbodInstance {
	instances := make([]guiDubbodInstance, 0)

	if s.kubeClient != nil {
		deployments, err := s.kubeClient.Kube().AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{
			LabelSelector: guiGatewaySelector,
		})
		if err == nil && len(deployments.Items) > 0 {
			for _, deployment := range deployments.Items {
				instances = append(instances, guiGatewayInstanceFromDeployment(deployment))
			}
		}
	}

	return instances
}

func guiGatewayInstanceFromDeployment(deployment appsv1.Deployment) guiDubbodInstance {
	desired := int32(1)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}
	ready := deployment.Status.ReadyReplicas
	gatewayName := deployment.Labels["gateway.networking.k8s.io/gateway-name"]

	return guiDubbodInstance{
		Name:            deployment.Name,
		Namespace:       deployment.Namespace,
		IsReady:         desired > 0 && ready >= desired,
		GatewayClass:    "dubbo",
		GatewayName:     gatewayName,
		ReadyReplicas:   ready,
		DesiredReplicas: desired,
	}
}

// buildGUIRoutes flattens the HTTPRoutes in the config store into the shape the
// console draws. Matches are rendered as the operator wrote them so a rule in
// the UI can be grepped for in `kubectl get httproute -o yaml`.
func (s *Server) buildGUIRoutes() []guiRoute {
	routes := make([]guiRoute, 0)
	if s.environment.ConfigStore == nil {
		return routes
	}

	for _, cfg := range s.environment.List(gvk.HTTPRoute, "") {
		spec, ok := cfg.Spec.(*sigsk8siogatewayapiapisv1.HTTPRouteSpec)
		if !ok {
			continue
		}

		entry := guiRoute{Name: cfg.Name, Namespace: cfg.Namespace}
		for _, parent := range spec.ParentRefs {
			entry.Parents = append(entry.Parents, string(parent.Name))
		}
		for _, hostname := range spec.Hostnames {
			entry.Hostnames = append(entry.Hostnames, string(hostname))
		}

		for _, rule := range spec.Rules {
			out := guiRouteRule{Match: guiRouteMatch(rule.Matches)}
			for _, backend := range rule.BackendRefs {
				item := guiRouteBackend{Name: string(backend.Name)}
				if backend.Port != nil {
					item.Port = int32(*backend.Port)
				}
				if backend.Weight != nil {
					item.Weight = *backend.Weight
				}
				out.Backends = append(out.Backends, item)
			}
			entry.Rules = append(entry.Rules, out)
		}
		routes = append(routes, entry)
	}

	sort.SliceStable(routes, func(i, j int) bool {
		if routes[i].Namespace != routes[j].Namespace {
			return routes[i].Namespace < routes[j].Namespace
		}
		return routes[i].Name < routes[j].Name
	})

	return routes
}

// guiRouteMatch renders the match conditions of one rule; an empty match set is
// Gateway API's "match everything", which reads better as an explicit path.
func guiRouteMatch(matches []sigsk8siogatewayapiapisv1.HTTPRouteMatch) string {
	if len(matches) == 0 {
		return "/"
	}

	parts := make([]string, 0, len(matches))
	for _, match := range matches {
		segment := ""
		if match.Path != nil && match.Path.Value != nil {
			segment = *match.Path.Value
			if match.Path.Type != nil && *match.Path.Type == sigsk8siogatewayapiapisv1.PathMatchPathPrefix {
				segment += "*"
			}
		}
		for _, header := range match.Headers {
			segment = strings.TrimSpace(segment + " " + string(header.Name) + "=" + header.Value)
		}
		if match.Method != nil {
			segment = strings.TrimSpace(string(*match.Method) + " " + segment)
		}
		if segment == "" {
			segment = "/"
		}
		parts = append(parts, segment)
	}
	return strings.Join(parts, " | ")
}

func (s *Server) buildGUIRegistries() []guiRegistry {
	registries := s.ServiceController().GetRegistries()
	items := make([]guiRegistry, 0, len(registries))
	for _, registry := range registries {
		items = append(items, guiRegistry{
			Provider: string(registry.Provider()),
			Cluster:  string(registry.Cluster()),
			Synced:   registry.HasSynced(),
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Provider != items[j].Provider {
			return items[i].Provider < items[j].Provider
		}
		return items[i].Cluster < items[j].Cluster
	})

	return items
}

func (s *Server) buildGUIConfigKinds() []guiConfigKind {
	return []guiConfigKind{
		{
			Kind:        "PeerAuthentication",
			Count:       s.countConfigs(gvk.PeerAuthentication),
			Description: "mTLS posture and workload identity policy.",
		},
		{
			Kind:        "RequestAuthentication",
			Count:       s.countConfigs(gvk.RequestAuthentication),
			Description: "JWT request authentication policy.",
		},
		{
			Kind:        "AuthorizationPolicy",
			Count:       s.countConfigs(gvk.AuthorizationPolicy),
			Description: "Request authorization policy.",
		},
		{
			Kind:        "HTTPRoute",
			Count:       s.countConfigs(gvk.HTTPRoute),
			Description: "Gateway API HTTP routing resources.",
		},
		{
			Kind:        "GatewayClass",
			Count:       s.countConfigs(gvk.GatewayClass),
			Description: "Gateway controller classes in scope.",
		},
		{
			Kind:        "Gateway",
			Count:       s.countConfigs(gvk.KubernetesGateway),
			Description: "Gateway instances served by the control plane.",
		},
	}
}

func (s *Server) buildGUIServices() []guiService {
	injectedNamespaces := make(map[string]bool)
	if s.kubeClient != nil {
		nsList, err := s.kubeClient.Kube().CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
		if err == nil {
			for _, ns := range nsList.Items {
				if ns.Labels != nil {
					if ns.Labels["dubbo-injection"] == "enabled" || ns.Labels["dubbo.apache.org/rev"] != "" {
						injectedNamespaces[ns.Name] = true
					}
				}
			}
		}
	}

	services := s.environment.Services()
	items := make([]guiService, 0, len(services))
	for _, service := range services {
		injected := injectedNamespaces[service.Attributes.Namespace]
		if !injected && s.environment.EndpointIndex != nil {
			if shards, ok := s.environment.EndpointIndex.ShardsForService(string(service.Hostname), service.Attributes.Namespace); ok {
				shards.RLock()
				for _, eps := range shards.Shards {
					for _, ep := range eps {
						if ep.Labels != nil && (ep.Labels["proxyless.dubbo.apache.org/inject"] == "true" || ep.Labels["dubbo.apache.org/rev"] != "") {
							injected = true
							break
						}
					}
					if injected {
						break
					}
				}
				shards.RUnlock()
			}
		}

		if !injected {
			continue
		}

		mode, fromPolicy := s.guiServiceMTLSMode(service)
		items = append(items, guiService{
			MTLSMode:        mode,
			MTLSFromPolicy:  fromPolicy,
			Name:            service.Attributes.Name,
			Hostname:        string(service.Hostname),
			Namespace:       service.Attributes.Namespace,
			Registry:        string(service.Attributes.ServiceRegistry),
			Ports:           servicePortsSummary(service),
			Exposure:        serviceExposure(service),
			ServiceAccounts: len(service.ServiceAccounts),
			DefaultAddress:  service.DefaultAddress,
			MeshExternal:    service.MeshExternal,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Namespace != items[j].Namespace {
			return items[i].Namespace < items[j].Namespace
		}
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		return items[i].Hostname < items[j].Hostname
	})

	return items
}

// guiServiceMTLSMode resolves the inbound mTLS mode callers of this service will
// meet. When no PeerAuthentication selects the workload the effective mode is
// UNKNOWN, and the sidecar falls back to PERMISSIVE
// (cmd/app/grpc_inbound.go effectiveMTLSMode) — reporting that fallback rather
// than "strict" keeps the console from claiming an encryption guarantee the data
// plane is not making.
func (s *Server) guiServiceMTLSMode(service *discoverymodel.Service) (mode string, fromPolicy bool) {
	push := s.environment.PushContext()
	if push == nil || push.AuthenticationPolicies == nil {
		return grpcInboundFallbackMTLSMode, false
	}

	strongest := discoverymodel.MTLSUnknown
	for _, port := range service.Ports {
		if port == nil {
			continue
		}
		effective := push.AuthenticationPolicies.EffectiveMutualTLSMode(
			service.Attributes.Namespace, nil, uint32(port.Port))
		if effective == discoverymodel.MTLSUnknown {
			continue
		}
		// Report the weakest mode any port allows: that is what an attacker gets.
		if strongest == discoverymodel.MTLSUnknown || effective < strongest {
			strongest = effective
		}
	}

	switch strongest {
	case discoverymodel.MTLSDisable:
		return "DISABLE", true
	case discoverymodel.MTLSPermissive:
		return "PERMISSIVE", true
	case discoverymodel.MTLSStrict:
		return "STRICT", true
	}
	return grpcInboundFallbackMTLSMode, false
}

func (s *Server) countConfigs(kind config.GroupVersionKind) int {
	if s.environment.ConfigStore == nil {
		return 0
	}
	return len(s.environment.List(kind, ""))
}

func servicePortsSummary(service *discoverymodel.Service) string {
	if len(service.Ports) == 0 {
		return "n/a"
	}

	segments := make([]string, 0, len(service.Ports))
	for _, port := range service.Ports {
		if port == nil {
			continue
		}

		segment := fmt.Sprintf("%d", port.Port)
		if port.Name != "" {
			segment = port.Name + ":" + segment
		}
		if port.Protocol != "" {
			segment += "/" + string(port.Protocol)
		}
		segments = append(segments, segment)
	}

	if len(segments) == 0 {
		return "n/a"
	}

	return strings.Join(segments, ", ")
}

func serviceExposure(service *discoverymodel.Service) string {
	switch {
	case service.MeshExternal:
		return "mesh-external"
	case service.Attributes.Type != "":
		// Verbatim `Service.spec.type` (ClusterIP, NodePort, LoadBalancer,
		// ExternalName) so the column matches what kubectl prints. The cases
		// below are resolution modes, not Kubernetes types, and stay lowercase
		// to keep that distinction visible.
		return service.Attributes.Type
	case service.Resolution == discoverymodel.Passthrough:
		return "passthrough"
	case service.Resolution == discoverymodel.DNSLB || service.Resolution == discoverymodel.DNSRoundRobinLB:
		return "dns"
	default:
		return "internal"
	}
}

func localLinkAddress(addr string) string {
	trimmed := strings.TrimSpace(addr)
	switch {
	case strings.HasPrefix(trimmed, ":"):
		return "127.0.0.1" + trimmed
	case strings.HasPrefix(trimmed, "0.0.0.0:"):
		return "127.0.0.1:" + strings.TrimPrefix(trimmed, "0.0.0.0:")
	case strings.HasPrefix(trimmed, "[::]:"):
		return "127.0.0.1:" + strings.TrimPrefix(trimmed, "[::]:")
	default:
		return trimmed
	}
}

func buildLocalHTTPURL(addr, requestPath string) string {
	return "http://" + localLinkAddress(addr) + requestPath
}
