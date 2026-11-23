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

package nodeagent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	cniconstants "github.com/apache/dubbo-kubernetes/cni/pkg/constants"
	"github.com/apache/dubbo-kubernetes/cni/pkg/iptables"
	"github.com/apache/dubbo-kubernetes/cni/pkg/plugin"
	"github.com/apache/dubbo-kubernetes/cni/pkg/tunnel"
	"github.com/apache/dubbo-kubernetes/cni/pkg/xdsclient"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// Config configures the per-node Route C CNI agent.
type Config struct {
	NodeName         string
	NodeIP           string
	SystemNamespace  string
	DiscoveryAddress string
	SocketPath       string
	StatusAddr       string
	TunnelConfig     tunnel.Config
	Redirector       iptables.Redirector
	RedirectConfig   iptables.Config
	DisableRedirect  bool
}

// ManagedPodInfo is serialized by /debug/pods.
type ManagedPodInfo struct {
	Key          string           `json:"key"`
	PodName      string           `json:"pod_name"`
	Namespace    string           `json:"namespace"`
	PodIP        string           `json:"pod_ip,omitempty"`
	NetNS        string           `json:"netns,omitempty"`
	Redirected   bool             `json:"redirected"`
	SecretName   string           `json:"secret_name"`
	SecretLoaded bool             `json:"secret_loaded"`
	XDS          xdsclient.Status `json:"xds"`
}

type managedPod struct {
	podName      string
	namespace    string
	podIP        string
	netns        string
	redirected   bool
	secretName   string
	secretDigest string

	client *xdsclient.Client
	cancel context.CancelFunc
}

// Agent runs on every Kubernetes Node as part of dubbo-cni-node. It watches local
// enrolled Pods and their dubbo-xds-* Secrets, programs per-Pod netns iptables,
// maintains per-Pod authenticated xDS ADS streams to dubbod, and drives the node-level
// Triple mTLS tunnel Proxy.
type Agent struct {
	cfg        Config
	kubeClient kubernetes.Interface
	proxy      *tunnel.Proxy
	uds        *UDSServer
	statusSrv  *http.Server
	statusLn   net.Listener

	ready atomic.Bool

	mu           sync.RWMutex
	baseCtx      context.Context
	pods         map[string]*managedPod // key: namespace/podName
	secretToPod  map[string]string      // key: namespace/secretName -> namespace/podName
	netnsByPod   map[string]string      // key: namespace/podName -> netns path
	podCache     map[string]*corev1.Pod
	nsCache      map[string]*corev1.Namespace
	secretCache  map[string]*corev1.Secret
}

// New creates a node Agent.
func New(kubeClient kubernetes.Interface, cfg Config) *Agent {
	if cfg.SystemNamespace == "" {
		cfg.SystemNamespace = "dubbo-system"
	}
	if cfg.SocketPath == "" {
		cfg.SocketPath = cniconstants.DefaultSocketPath
	}
	if cfg.StatusAddr == "" {
		cfg.StatusAddr = fmt.Sprintf(":%d", cniconstants.DefaultStatusPort)
	}
	if cfg.RedirectConfig.OutboundPort == 0 {
		cfg.RedirectConfig = iptables.DefaultConfig()
	}
	if cfg.NodeIP != "" && cfg.RedirectConfig.NodeIP == "" {
		cfg.RedirectConfig.NodeIP = cfg.NodeIP
	}
	if cfg.Redirector == nil && !cfg.DisableRedirect {
		cfg.Redirector = &iptables.NetNSRedirector{}
	}

	a := &Agent{
		cfg:         cfg,
		kubeClient:  kubeClient,
		proxy:       tunnel.NewProxy(cfg.TunnelConfig),
		pods:        map[string]*managedPod{},
		secretToPod: map[string]string{},
		netnsByPod:  map[string]string{},
		podCache:    map[string]*corev1.Pod{},
		nsCache:     map[string]*corev1.Namespace{},
		secretCache: map[string]*corev1.Secret{},
	}
	a.uds = NewUDSServer(cfg.SocketPath, a)
	return a
}

// Proxy returns the underlying node-level tunnel Proxy.
func (a *Agent) Proxy() *tunnel.Proxy {
	return a.proxy
}

// StatusAddr returns the bound HTTP status/debug server address.
func (a *Agent) StatusAddr() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.statusLn == nil {
		return a.cfg.StatusAddr
	}
	return a.statusLn.Addr().String()
}

// Start starts the tunnel proxy, UDS CNI callback listener, HTTP status server,
// and Kubernetes informers for local node Pods, Namespaces, and dubbo-xds-* Secrets.
func (a *Agent) Start(ctx context.Context) error {
	a.mu.Lock()
	a.baseCtx = ctx
	a.mu.Unlock()

	if err := a.proxy.Start(ctx); err != nil {
		return err
	}
	if err := a.uds.Start(ctx); err != nil {
		_ = a.proxy.Close()
		return err
	}
	if err := a.startStatusServer(ctx); err != nil {
		_ = a.uds.Close()
		_ = a.proxy.Close()
		return err
	}

	if a.kubeClient != nil {
		if err := a.startInformers(ctx); err != nil {
			_ = a.Close()
			return err
		}
	}
	a.ready.Store(true)
	return nil
}

// Close tears down all per-Pod xDS clients, listeners, and UDS socket.
func (a *Agent) Close() error {
	a.ready.Store(false)

	a.mu.Lock()
	for _, mp := range a.pods {
		if mp.cancel != nil {
			mp.cancel()
		}
	}
	a.pods = map[string]*managedPod{}
	a.mu.Unlock()

	var errs []error
	if a.uds != nil {
		errs = append(errs, a.uds.Close())
	}
	if a.statusSrv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		errs = append(errs, a.statusSrv.Shutdown(shutdownCtx))
		cancel()
	}
	if a.proxy != nil {
		errs = append(errs, a.proxy.Close())
	}
	return errors.Join(errs...)
}

func (a *Agent) startInformers(ctx context.Context) error {
	var podOpts []informers.SharedInformerOption
	if a.cfg.NodeName != "" {
		podOpts = append(podOpts, informers.WithTweakListOptions(func(lo *metav1.ListOptions) {
			lo.FieldSelector = fields.OneTermEqualSelector("spec.nodeName", a.cfg.NodeName).String()
		}))
	}
	podFactory := informers.NewSharedInformerFactoryWithOptions(a.kubeClient, 0, podOpts...)
	sharedFactory := informers.NewSharedInformerFactory(a.kubeClient, 0)

	podInformer := podFactory.Core().V1().Pods().Informer()
	nsInformer := sharedFactory.Core().V1().Namespaces().Informer()
	secretInformer := sharedFactory.Core().V1().Secrets().Informer()

	_, _ = podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { a.onPodUpsert(obj) },
		UpdateFunc: func(_, obj any) { a.onPodUpsert(obj) },
		DeleteFunc: func(obj any) { a.onPodDelete(obj) },
	})
	_, _ = nsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { a.onNamespaceUpsert(obj) },
		UpdateFunc: func(_, obj any) { a.onNamespaceUpsert(obj) },
		DeleteFunc: func(obj any) { a.onNamespaceDelete(obj) },
	})
	_, _ = secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { a.onSecretUpsert(obj) },
		UpdateFunc: func(_, obj any) { a.onSecretUpsert(obj) },
		DeleteFunc: func(obj any) { a.onSecretDelete(obj) },
	})

	podFactory.Start(ctx.Done())
	sharedFactory.Start(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced, nsInformer.HasSynced, secretInformer.HasSynced) {
		return fmt.Errorf("timed out waiting for dubbo-cni-node informer caches to sync")
	}
	return nil
}

// HandleCNIAdd records the Pod's netns reported by dubbo-cni ADD and triggers reconciliation.
func (a *Agent) HandleCNIAdd(ctx context.Context, ev plugin.PodEvent) error {
	if ev.Namespace == "" || ev.PodName == "" {
		return fmt.Errorf("missing pod namespace or name")
	}
	key := podKey(ev.Namespace, ev.PodName)
	a.mu.Lock()
	if ev.NetNS != "" {
		a.netnsByPod[key] = ev.NetNS
	}
	pod := a.podCache[key]
	a.mu.Unlock()

	if pod != nil {
		return a.reconcilePodLockedOutside(ctx, pod)
	}
	return nil
}

// HandleCNIDel removes netns tracking and tears down per-Pod state on CNI DEL.
func (a *Agent) HandleCNIDel(ctx context.Context, ev plugin.PodEvent) error {
	if ev.Namespace == "" || ev.PodName == "" {
		return nil
	}
	key := podKey(ev.Namespace, ev.PodName)
	a.mu.Lock()
	delete(a.netnsByPod, key)
	a.mu.Unlock()
	a.teardownPod(ctx, key)
	return nil
}

func (a *Agent) onNamespaceUpsert(obj any) {
	ns, ok := obj.(*corev1.Namespace)
	if !ok || ns == nil {
		return
	}
	a.mu.Lock()
	a.nsCache[ns.Name] = ns.DeepCopy()
	var affected []*corev1.Pod
	for _, p := range a.podCache {
		if p.Namespace == ns.Name {
			affected = append(affected, p.DeepCopy())
		}
	}
	ctx := a.baseCtx
	a.mu.Unlock()

	for _, p := range affected {
		_ = a.reconcilePodLockedOutside(ctx, p)
	}
}

func (a *Agent) onNamespaceDelete(obj any) {
	ns, ok := extractObject[*corev1.Namespace](obj)
	if !ok {
		return
	}
	a.mu.Lock()
	delete(a.nsCache, ns.Name)
	a.mu.Unlock()
}

func (a *Agent) onPodUpsert(obj any) {
	pod, ok := obj.(*corev1.Pod)
	if !ok || pod == nil {
		return
	}
	if a.cfg.NodeName != "" && pod.Spec.NodeName != "" && pod.Spec.NodeName != a.cfg.NodeName {
		return
	}
	key := podKey(pod.Namespace, pod.Name)
	a.mu.Lock()
	a.podCache[key] = pod.DeepCopy()
	ctx := a.baseCtx
	a.mu.Unlock()

	_ = a.reconcilePodLockedOutside(ctx, pod)
}

func (a *Agent) onPodDelete(obj any) {
	pod, ok := extractObject[*corev1.Pod](obj)
	if !ok {
		return
	}
	key := podKey(pod.Namespace, pod.Name)
	a.mu.Lock()
	delete(a.podCache, key)
	ctx := a.baseCtx
	a.mu.Unlock()
	a.teardownPod(ctx, key)
}

func (a *Agent) onSecretUpsert(obj any) {
	secret, ok := obj.(*corev1.Secret)
	if !ok || secret == nil {
		return
	}
	sKey := podKey(secret.Namespace, secret.Name)
	a.mu.Lock()
	a.secretCache[sKey] = secret.DeepCopy()
	pKey, matched := a.secretToPod[sKey]
	var pod *corev1.Pod
	if matched {
		pod = a.podCache[pKey]
	} else {
		// Check if any cached pod maps to this secret name.
		for _, candidate := range a.podCache {
			if candidate.Namespace == secret.Namespace &&
				inject.InherentXDSSecretNameForMeta(candidate.ObjectMeta) == secret.Name {
				pod = candidate
				break
			}
		}
	}
	ctx := a.baseCtx
	a.mu.Unlock()

	if pod != nil {
		_ = a.reconcilePodLockedOutside(ctx, pod)
	}
}

func (a *Agent) onSecretDelete(obj any) {
	secret, ok := extractObject[*corev1.Secret](obj)
	if !ok {
		return
	}
	sKey := podKey(secret.Namespace, secret.Name)
	a.mu.Lock()
	delete(a.secretCache, sKey)
	a.mu.Unlock()
}

func (a *Agent) reconcilePodLockedOutside(ctx context.Context, pod *corev1.Pod) error {
	if ctx == nil {
		ctx = context.Background()
	}
	key := podKey(pod.Namespace, pod.Name)

	a.mu.RLock()
	ns := a.nsCache[pod.Namespace]
	netns := a.netnsByPod[key]
	secretName := inject.InherentXDSSecretNameForMeta(pod.ObjectMeta)
	secret := a.secretCache[podKey(pod.Namespace, secretName)]
	a.mu.RUnlock()

	if !IsEnrolledPod(pod, ns, a.cfg.SystemNamespace) {
		a.teardownPod(ctx, key)
		return nil
	}

	a.mu.Lock()
	mp, exists := a.pods[key]
	if !exists {
		mp = &managedPod{
			podName:    pod.Name,
			namespace:  pod.Namespace,
			secretName: secretName,
		}
		a.pods[key] = mp
	}
	a.secretToPod[podKey(pod.Namespace, secretName)] = key
	if pod.Status.PodIP != "" {
		mp.podIP = pod.Status.PodIP
	}
	if netns != "" {
		mp.netns = netns
	}
	needRedirect := !a.cfg.DisableRedirect && a.cfg.Redirector != nil && mp.netns != "" && !mp.redirected
	targetNetNS := mp.netns
	a.mu.Unlock()

	if needRedirect {
		if err := a.cfg.Redirector.Apply(ctx, targetNetNS, a.cfg.RedirectConfig); err != nil {
			return err
		}
		a.mu.Lock()
		if cur, ok := a.pods[key]; ok {
			cur.redirected = true
		}
		a.mu.Unlock()
	}

	if secret == nil {
		return nil
	}

	digest := secretDigest(secret.Data)
	a.mu.RLock()
	cur := a.pods[key]
	sameSecret := cur != nil && cur.secretDigest == digest && cur.client != nil
	a.mu.RUnlock()
	if sameSecret {
		return nil
	}

	xdsCfg, err := xdsclient.ConfigFromSecretData(secret.Data)
	if err != nil {
		return err
	}
	if a.cfg.DiscoveryAddress != "" {
		xdsCfg.ServerURI = a.cfg.DiscoveryAddress
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	cur, ok := a.pods[key]
	if !ok {
		return nil
	}
	cur.secretDigest = digest

	if cur.client != nil {
		if err := cur.client.UpdateConfig(xdsCfg); err != nil {
			return err
		}
	} else {
		client, err := xdsclient.New(xdsCfg)
		if err != nil {
			return err
		}
		runCtx, cancel := context.WithCancel(a.baseCtx)
		cur.client = client
		cur.cancel = cancel
		go func() {
			_ = client.Run(runCtx)
		}()
	}

	if cur.podIP != "" {
		a.proxy.RegisterWorkload(&tunnel.Workload{
			PodName:   cur.podName,
			Namespace: cur.namespace,
			PodIP:     cur.podIP,
			Certs:     xdsCfg.Certs,
			XDS:       cur.client,
		})
	}
	return nil
}

func (a *Agent) teardownPod(ctx context.Context, key string) {
	a.mu.Lock()
	mp, ok := a.pods[key]
	if !ok {
		a.mu.Unlock()
		return
	}
	delete(a.pods, key)
	delete(a.secretToPod, podKey(mp.namespace, mp.secretName))
	a.mu.Unlock()

	if mp.cancel != nil {
		mp.cancel()
	}
	if mp.podIP != "" {
		a.proxy.UnregisterWorkload(mp.podIP)
	}
	if mp.redirected && mp.netns != "" && a.cfg.Redirector != nil {
		if ctx == nil {
			ctx = context.Background()
		}
		_ = a.cfg.Redirector.Cleanup(ctx, mp.netns)
	}
}

// ManagedPods returns a deterministic snapshot of all locally managed enrolled Pods.
func (a *Agent) ManagedPods() []ManagedPodInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]ManagedPodInfo, 0, len(a.pods))
	for key, mp := range a.pods {
		info := ManagedPodInfo{
			Key:          key,
			PodName:      mp.podName,
			Namespace:    mp.namespace,
			PodIP:        mp.podIP,
			NetNS:        mp.netns,
			Redirected:   mp.redirected,
			SecretName:   mp.secretName,
			SecretLoaded: mp.secretDigest != "",
		}
		if mp.client != nil {
			info.XDS = mp.client.Status()
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func (a *Agent) startStatusServer(ctx context.Context) error {
	ln, err := net.Listen("tcp", a.cfg.StatusAddr)
	if err != nil {
		return fmt.Errorf("listen status %s: %w", a.cfg.StatusAddr, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !a.ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready\n"))
	})
	mux.HandleFunc("/debug/pods", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(a.ManagedPods())
	})
	mux.HandleFunc("/debug/syncz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pods := a.ManagedPods()
		statuses := make([]xdsclient.Status, 0, len(pods))
		for _, p := range pods {
			if p.XDS.NodeID != "" {
				statuses = append(statuses, p.XDS)
			}
		}
		_ = json.NewEncoder(w).Encode(statuses)
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		pods := a.ManagedPods()
		connected, ready := 0, 0
		for _, p := range pods {
			if p.XDS.Connected {
				connected++
			}
			if p.XDS.Ready {
				ready++
			}
		}
		m := a.proxy.Metrics()
		fmt.Fprintf(w,
			"dubbo_cni_managed_pods %d\ndubbo_cni_xds_connected_pods %d\ndubbo_cni_xds_ready_pods %d\ndubbo_cni_tunnel_outbound_total %d\ndubbo_cni_tunnel_inbound_total %d\ndubbo_cni_tunnel_active_connections %d\n",
			len(pods), connected, ready, m.OutboundTotal, m.InboundTotal, m.Active,
		)
	})

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	a.mu.Lock()
	a.statusLn = ln
	a.statusSrv = srv
	a.mu.Unlock()

	go func() { _ = srv.Serve(ln) }()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(shutdownCtx)
		cancel()
	}()
	return nil
}

func podKey(ns, name string) string {
	return ns + "/" + name
}

func secretDigest(data map[string][]byte) string {
	h := sha256.New()
	for _, key := range []string{
		inject.InherentTripleBootstrapFileName,
		constants.CertChainFilename,
		constants.KeyFilename,
		constants.CACertNamespaceConfigMapDataName,
	} {
		h.Write([]byte(key))
		h.Write([]byte{0})
		h.Write(bytes.TrimSpace(data[key]))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func extractObject[T any](obj any) (T, bool) {
	if typed, ok := obj.(T); ok {
		return typed, true
	}
	if tomb, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		if typed, ok := tomb.Obj.(T); ok {
			return typed, true
		}
	}
	var zero T
	return zero, false
}
