package bootstrap

import (
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
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/version"
)

type guiOverview struct {
	Product          string              `json:"product"`
	Version          string              `json:"version"`
	Cluster          string              `json:"clusterId"`
	Namespace        string              `json:"namespace"`
	PodName          string              `json:"podName,omitempty"`
	Mesh             guiOverviewMesh     `json:"mesh"`
	Server           guiOverviewServer   `json:"server"`
	Status           guiOverviewStatus   `json:"status"`
	Counts           guiOverviewCounts   `json:"counts"`
	ConfigKinds      []guiConfigKind     `json:"configKinds"`
	Registries       []guiRegistry       `json:"registries"`
	Services         []guiService        `json:"services"`
	Instances        []guiDubbodInstance `json:"instances"`
	GatewayInstances []guiDubbodInstance `json:"gatewayInstances"`
	UpdatedAt        time.Time           `json:"updatedAt"`
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
	Services            int `json:"services"`
	EndpointServices    int `json:"endpointServices"`
	XDSConnections      int `json:"xdsConnections"`
	Registries          int `json:"registries"`
	DestinationRules    int `json:"destinationRules"`
	VirtualServices     int `json:"virtualServices"`
	PeerAuthentications int `json:"peerAuthentications"`
	HTTPRoutes          int `json:"httpRoutes"`
	GatewayClasses      int `json:"gatewayClasses"`
	Gateways            int `json:"gateways"`
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
	if s.guiPath == "/" {
		s.guiMux.HandleFunc(overviewPath, s.guiOverviewHandler)
		s.guiMux.HandleFunc(logsPath, s.guiLogsHandler)
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
		response, err = s.deploymentLogs(request.Context(), "dubbod", namespace, name, "startup", tailLines)
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
			ProxylessSynced: s.proxylessGRPCWorkloadController == nil || s.proxylessGRPCWorkloadController.HasSynced(),
			InjectorReady:   s.kubeClient == nil || s.readinessFlags.InjectorReady.Load(),
			ValidationReady: s.kubeClient == nil || s.readinessFlags.configValidationReady.Load(),
		},
		Counts: guiOverviewCounts{
			Services:            len(services),
			EndpointServices:    len(s.environment.EndpointIndex.AllServices()),
			XDSConnections:      len(s.XDSServer.AllClients()),
			Registries:          len(registries),
			DestinationRules:    configKinds[0].Count,
			VirtualServices:     configKinds[1].Count,
			PeerAuthentications: configKinds[2].Count,
			HTTPRoutes:          configKinds[3].Count,
			GatewayClasses:      configKinds[4].Count,
			Gateways:            configKinds[5].Count,
		},
		ConfigKinds:      configKinds,
		Registries:       registries,
		Services:         services,
		Instances:        instances,
		GatewayInstances: s.buildGUIGatewayInstances(),
		UpdatedAt:        time.Now().UTC(),
	}
}

func (s *Server) buildGUIDubbodInstances() []guiDubbodInstance {
	instances := make([]guiDubbodInstance, 0)

	if s.kubeClient != nil {
		pods, err := s.kubeClient.Kube().CoreV1().Pods(s.namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: "app=dubbo-control-plane",
		})
		if err == nil && len(pods.Items) > 0 {
			for _, pod := range pods.Items {
				ready := false
				for _, cond := range pod.Status.Conditions {
					if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
						ready = true
						break
					}
				}
				instances = append(instances, guiDubbodInstance{
					Name:      pod.Name,
					Namespace: pod.Namespace,
					IP:        pod.Status.PodIP,
					IsReady:   ready,
				})
			}
		}
	}

	// fallback if no pods found or kubeclient is nil
	if len(instances) == 0 {
		ip := "127.0.0.1"
		if s.podName != "" {
			ip = "localhost" // Fallback IP when running locally
		}
		instances = append(instances, guiDubbodInstance{
			Name:      s.podName,
			Namespace: s.namespace,
			IP:        ip,
			IsReady:   true,
		})
	}

	return instances
}

func (s *Server) buildGUIGatewayInstances() []guiDubbodInstance {
	instances := make([]guiDubbodInstance, 0)

	if s.kubeClient != nil {
		deployments, err := s.kubeClient.Kube().AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=dxgate,app.kubernetes.io/managed-by=dubbod",
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
			Kind:        "DestinationRule",
			Count:       s.countConfigs(gvk.DestinationRule),
			Description: "Traffic policy and subset routing intent.",
		},
		{
			Kind:        "VirtualService",
			Count:       s.countConfigs(gvk.VirtualService),
			Description: "Traffic routing and service composition.",
		},
		{
			Kind:        "PeerAuthentication",
			Count:       s.countConfigs(gvk.PeerAuthentication),
			Description: "mTLS posture and workload identity policy.",
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

		items = append(items, guiService{
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
		return strings.ToLower(service.Attributes.Type)
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
