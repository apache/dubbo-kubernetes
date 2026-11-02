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
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	discoverymodel "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	pkgbootstrap "github.com/apache/dubbo-kubernetes/pkg/bootstrap"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	meshconfig "github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	kubelib "github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	pkgmodel "github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/spiffe"
	"github.com/apache/dubbo-kubernetes/pkg/xdsbootstrap"

	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/nodeagent/util"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/ca"
	pkiutil "github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/util"
	caserver "github.com/apache/dubbo-kubernetes/dubbod/security/pkg/server/ca"
	meshv1alpha1 "github.com/dubml/api/mesh/v1alpha1"
	"google.golang.org/protobuf/proto"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

const (
	inherentXDSControllerName = "Inherent xDS workloads"
	serviceNodeSeparator      = "~"

	inherentXDSManagedPodIndex     = "inherent-xds-managed"
	inherentXDSManagedPodIndexKey  = "true"
	inherentXDSBundleRequeueBatch  = 250
	inherentXDSBundleRequeuePeriod = 100 * time.Millisecond
)

var inherentXDSLog = log.RegisterScope("inherentxds", "Inherent xDS workload controller")

type inherentXDSWorkloadController struct {
	server  *Server
	client  kubelib.Client
	pods    kclient.Client[*corev1.Pod]
	secrets kclient.Client[*corev1.Secret]
	// managedPods keeps bundle/cert updates from scanning every Pod in the cluster.
	managedPods kclient.RawIndexer
	queue       controllers.Queue

	bundleWatcherID int32
	bundleWatcherCh chan struct{}

	rotationMu    sync.Mutex
	rotations     map[types.NamespacedName]time.Time
	rotationTimer *time.Timer
	nextRotation  time.Time
}

func (s *Server) initInherentXDSWorkloads() error {
	if s.kubeClient == nil {
		return nil
	}

	controller := newInherentXDSWorkloadController(s, s.kubeClient)
	s.inherentXDSWorkloadController = controller
	s.environment.AddConfigHandler(func(change *discoverymodel.ConfigChange) {
		controller.handleWorkloadConfigUpdate(change)
	})

	s.addStartFunc(inherentXDSControllerName, func(stop <-chan struct{}) error {
		go controller.Run(stop)
		return nil
	})
	return nil
}

func (s *Server) inherentXDSWorkloadsSynced() bool {
	if s.inherentXDSWorkloadController != nil && !s.inherentXDSWorkloadController.HasSynced() {
		return false
	}
	return true
}

func newInherentXDSWorkloadController(s *Server, client kubelib.Client) *inherentXDSWorkloadController {
	c := &inherentXDSWorkloadController{
		server: s,
		client: client,
		pods: kclient.NewFiltered[*corev1.Pod](client, kclient.Filter{
			ObjectFilter: client.ObjectFilter(),
		}),
		secrets: kclient.NewFiltered[*corev1.Secret](client, kclient.Filter{
			ObjectFilter: client.ObjectFilter(),
		}),
		rotations: make(map[types.NamespacedName]time.Time),
	}
	c.managedPods = c.pods.Index(inherentXDSManagedPodIndex, func(pod *corev1.Pod) []string {
		if shouldManageInherentXDSPod(pod) {
			return []string{inherentXDSManagedPodIndexKey}
		}
		return nil
	})
	c.queue = controllers.NewQueue(inherentXDSControllerName,
		controllers.WithReconciler(func(key types.NamespacedName) error {
			return c.reconcile(key)
		}),
		controllers.WithMaxAttempts(5),
	)

	c.pods.AddEventHandler(controllers.FilteredObjectSpecHandler(c.queue.AddObject, func(o controllers.Object) bool {
		pod := controllers.Extract[*corev1.Pod](o)
		return shouldManageInherentXDSPod(pod)
	}))
	c.secrets.AddEventHandler(controllers.ObjectHandler(func(o controllers.Object) {
		secret := controllers.Extract[*corev1.Secret](o)
		if secret == nil {
			return
		}
		for _, owner := range secret.OwnerReferences {
			if owner.Kind != "Pod" || owner.APIVersion != "v1" {
				continue
			}
			pod := c.pods.Get(owner.Name, secret.Namespace)
			if pod != nil && pod.UID == owner.UID && secret.Name == inject.InherentXDSSecretNameForMeta(pod.ObjectMeta) {
				c.queue.AddObject(pod)
				if s.XDSServer != nil {
					s.XDSServer.ProxyUpdate(client.ClusterID(), pod.Status.PodIP)
				}
			}
		}
	}))

	return c
}

func (c *inherentXDSWorkloadController) Run(stop <-chan struct{}) {
	c.pods.Start(stop)
	c.secrets.Start(stop)
	if !kubelib.WaitForCacheSync(inherentXDSControllerName, stop, c.pods.HasSynced, c.secrets.HasSynced) {
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

func (c *inherentXDSWorkloadController) HasSynced() bool {
	return c.pods.HasSynced() && c.secrets.HasSynced() && c.queue.HasSynced()
}

func (c *inherentXDSWorkloadController) watchBundleChanges(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case _, ok := <-c.bundleWatcherCh:
			if !ok {
				return
			}
			c.enqueueManagedPodsBatched(stop)
		}
	}
}

func (c *inherentXDSWorkloadController) enqueueAllPods() {
	for _, key := range c.managedPodKeys() {
		c.queue.Add(key)
	}
}

func (c *inherentXDSWorkloadController) handleWorkloadConfigUpdate(req *discoverymodel.ConfigChange) {
	if !inherentXDSWorkloadConfigNeedsUpdate(req) {
		return
	}
	c.enqueueAllPods()
}

func inherentXDSWorkloadConfigNeedsUpdate(req *discoverymodel.ConfigChange) bool {
	if req == nil {
		return false
	}
	if req.Full || req.Forced || len(req.AddressesUpdated) > 0 {
		return true
	}
	for cfg := range req.ConfigsUpdated {
		switch cfg.Kind {
		case kind.HTTPRoute, kind.BackendTLSPolicy, kind.CircuitBreakerPolicy, kind.FaultInjectionPolicy, kind.PeerAuthentication, kind.RequestAuthentication, kind.AuthorizationPolicy, kind.Telemetry, kind.Service, kind.EndpointSlice, kind.Endpoints, kind.Pod, kind.Namespace:
			return true
		}
	}
	if req.EndpointsChanged || req.Global {
		return true
	}
	return false
}

func (c *inherentXDSWorkloadController) enqueueManagedPodsBatched(stop <-chan struct{}) {
	keys := c.managedPodKeys()
	for start := 0; start < len(keys); start += inherentXDSBundleRequeueBatch {
		end := start + inherentXDSBundleRequeueBatch
		if end > len(keys) {
			end = len(keys)
		}
		for _, key := range keys[start:end] {
			c.queue.Add(key)
		}
		if end == len(keys) {
			return
		}
		timer := time.NewTimer(inherentXDSBundleRequeuePeriod)
		select {
		case <-stop:
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (c *inherentXDSWorkloadController) managedPodKeys() []types.NamespacedName {
	var pods []*corev1.Pod
	if c.managedPods != nil {
		for _, item := range c.managedPods.Lookup(inherentXDSManagedPodIndexKey) {
			pod, ok := item.(*corev1.Pod)
			if ok && pod != nil {
				pods = append(pods, pod)
			}
		}
	} else if c.pods != nil {
		pods = c.pods.List(metav1.NamespaceAll, labels.Everything())
	}

	keys := make([]types.NamespacedName, 0, len(pods))
	for _, pod := range pods {
		if shouldManageInherentXDSPod(pod) {
			keys = append(keys, types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace})
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Namespace != keys[j].Namespace {
			return keys[i].Namespace < keys[j].Namespace
		}
		return keys[i].Name < keys[j].Name
	})
	return keys
}

func (c *inherentXDSWorkloadController) reconcile(key types.NamespacedName) error {
	pod := c.pods.Get(key.Name, key.Namespace)
	if pod == nil {
		c.clearRotation(key)
		// Kubernetes garbage collection uses the Pod UID; deleting by name here
		// could remove credentials of a replacement Pod before our cache catches up.
		return nil
	}
	if !shouldManageInherentXDSPod(pod) || pod.DeletionTimestamp != nil {
		c.clearRotation(key)
		return c.deleteSecretForPod(pod)
	}
	secrets := c.client.Kube().CoreV1().Secrets(pod.Namespace)
	secretName := inject.InherentXDSSecretNameForMeta(pod.ObjectMeta)
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == inject.InherentXDSVolumeName && volume.Secret != nil && volume.Secret.SecretName != secretName {
			return fmt.Errorf("pod %s/%s mounts legacy shared credentials; recreate the Pod to obtain its own Secret", pod.Namespace, pod.Name)
		}
	}
	current, err := secrets.Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		current = nil
	}
	if current != nil && !workloadSecretOwnedByPod(current, pod) {
		return fmt.Errorf("refusing to replace credentials Secret %s/%s owned by another workload", pod.Namespace, secretName)
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

func workloadSecretOwnedByPod(secret *corev1.Secret, pod *corev1.Pod) bool {
	if secret.Namespace != pod.Namespace || pod.UID == "" {
		return false
	}
	for _, owner := range secret.OwnerReferences {
		if owner.APIVersion == "v1" && owner.Kind == "Pod" && owner.Name == pod.Name && owner.UID == pod.UID {
			return true
		}
	}
	return false
}

type inherentXDSWorkloadContext struct {
	node             *pkgmodel.Node
	nodeID           string
	podName          string
	podNamespace     string
	podIP            string
	serviceAccount   string
	trustDomain      string
	clusterID        string
	discoveryAddress string
	caAddress        string
}

func (c *inherentXDSWorkloadController) buildSecret(pod *corev1.Pod, current *corev1.Secret) (*corev1.Secret, time.Time, error) {
	workload, err := c.buildWorkloadContext(pod)
	if err != nil {
		return nil, time.Time{}, err
	}

	bootstrapJSON, err := buildBootstrapJSON(workload)
	if err != nil {
		return nil, time.Time{}, err
	}

	certChain, keyPEM, rootCert, expireAt, reusedCert := reusableWorkloadCertificate(current, c.activeRootCert())
	if !reusedCert {
		certChain, keyPEM, rootCert, expireAt, err = c.issueWorkloadCertificate(pod)
		if err != nil {
			return nil, time.Time{}, err
		}
	} else {
		// Java's file_watcher accepts PKCS#8. Normalize reused keys without
		// changing the certificate, identity, or private key material.
		privateKey, err := pkiutil.ParsePemEncodedKey(keyPEM)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("parse reused workload private key: %w", err)
		}
		encodedKey, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("encode workload private key as PKCS#8: %w", err)
		}
		keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encodedKey})
	}

	secret := buildInherentXDSSecret(pod, bootstrapJSON, certChain, keyPEM, rootCert)
	return secret, expireAt, nil
}

func buildInherentXDSSecret(pod *corev1.Pod, bootstrapJSON, certChain, keyPEM, rootCert []byte) *corev1.Secret {
	data := map[string][]byte{
		constants.CertChainFilename:                certChain,
		constants.KeyFilename:                      keyPEM,
		constants.CACertNamespaceConfigMapDataName: rootCert,
	}
	data[inject.InherentTripleBootstrapFileName] = bootstrapJSON
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inject.InherentXDSSecretNameForMeta(pod.ObjectMeta),
			Namespace: pod.Namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "v1",
				Kind:       "Pod",
				Name:       pod.Name,
				UID:        pod.UID,
			}},
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}
}

func reusableWorkloadCertificate(secret *corev1.Secret, activeRootCert []byte) ([]byte, []byte, []byte, time.Time, bool) {
	if secret == nil || len(secret.Data) == 0 {
		return nil, nil, nil, time.Time{}, false
	}
	certChain := secret.Data[constants.CertChainFilename]
	keyPEM := secret.Data[constants.KeyFilename]
	rootCert := secret.Data[constants.CACertNamespaceConfigMapDataName]
	if len(certChain) == 0 || len(keyPEM) == 0 || len(rootCert) == 0 {
		return nil, nil, nil, time.Time{}, false
	}
	if len(activeRootCert) > 0 && !bytes.Equal(rootCert, activeRootCert) {
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

func (c *inherentXDSWorkloadController) buildWorkloadContext(pod *corev1.Pod) (*inherentXDSWorkloadContext, error) {
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
	clusterID := string(c.server.clusterID)
	if clusterID == "" {
		clusterID = constants.DefaultClusterName
	}
	discoveryAddress := proxyConfig.GetDiscoveryAddress()
	caAddress := discoveryAddress
	proxyConfig = proto.Clone(proxyConfig).(*meshv1alpha1.ProxyConfig)
	proxyConfig.DiscoveryAddress = discoveryAddress

	nodeID := strings.Join([]string{
		string(pkgmodel.Inherent),
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
			"DUBBO_META_CLUSTER_ID=" + clusterID,
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

	return &inherentXDSWorkloadContext{
		node:             node,
		nodeID:           nodeID,
		podName:          pod.Name,
		podNamespace:     pod.Namespace,
		podIP:            podIP,
		serviceAccount:   serviceAccount,
		trustDomain:      trustDomain,
		clusterID:        clusterID,
		discoveryAddress: discoveryAddress,
		caAddress:        caAddress,
	}, nil
}

func buildBootstrapJSON(workload *inherentXDSWorkloadContext) ([]byte, error) {
	bootstrapCfg, err := xdsbootstrap.GenerateBootstrap(xdsbootstrap.GenerateBootstrapOptions{
		Node:             workload.node,
		DiscoveryAddress: workload.discoveryAddress,
		CertDir:          inject.InherentXDSMountPath,
	})
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(bootstrapCfg, "", "  ")
}

func (c *inherentXDSWorkloadController) issueWorkloadCertificate(pod *corev1.Pod) ([]byte, []byte, []byte, time.Time, error) {
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
		PKCS8Key:   true,
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

func (c *inherentXDSWorkloadController) activeAuthority() caserver.CertificateAuthority {
	if c.server.RA != nil {
		return c.server.RA
	}
	return c.server.CA
}

func (c *inherentXDSWorkloadController) activeRootCert() []byte {
	authority := c.activeAuthority()
	if authority == nil || authority.GetCAKeyCertBundle() == nil {
		return nil
	}
	return authority.GetCAKeyCertBundle().GetRootCertPem()
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

func isInherentTriplePod(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}
	for _, templateName := range strings.Split(pod.Annotations[inject.InherentInjectTemplatesAnnoName], ",") {
		if strings.TrimSpace(templateName) == inject.InherentTripleTemplateName {
			return true
		}
	}
	for _, container := range pod.Spec.Containers {
		for _, envVar := range container.Env {
			if envVar.Name == inject.InherentTripleBootstrapEnvName && envVar.Value == inject.InherentTripleBootstrapPath {
				return true
			}
		}
	}
	return false
}

func shouldManageInherentXDSPod(pod *corev1.Pod) bool {
	return isInherentTriplePod(pod)
}

func (c *inherentXDSWorkloadController) deleteSecretForPod(pod *corev1.Pod) error {
	secrets := c.client.Kube().CoreV1().Secrets(pod.Namespace)
	secret, err := secrets.Get(context.Background(), inject.InherentXDSSecretNameForMeta(pod.ObjectMeta), metav1.GetOptions{})
	if err != nil {
		return controllers.IgnoreNotFound(err)
	}
	if !workloadSecretOwnedByPod(secret, pod) {
		return nil
	}
	err = secrets.Delete(context.Background(), secret.Name, metav1.DeleteOptions{
		Preconditions: &metav1.Preconditions{UID: &secret.UID, ResourceVersion: &secret.ResourceVersion},
	})
	return controllers.IgnoreNotFound(err)
}

func (c *inherentXDSWorkloadController) scheduleRotation(key types.NamespacedName, expireAt time.Time) {
	now := time.Now()
	rotateAt := now.Add(workloadRotationDelay(now, expireAt))
	c.rotationMu.Lock()
	defer c.rotationMu.Unlock()
	oldRotateAt, hadOld := c.rotations[key]
	c.rotations[key] = rotateAt
	if c.rotationTimer == nil || rotateAt.Before(c.nextRotation) || (hadOld && oldRotateAt.Equal(c.nextRotation)) {
		c.resetRotationTimerLocked(now)
	}
}

func (c *inherentXDSWorkloadController) clearRotation(key types.NamespacedName) {
	c.rotationMu.Lock()
	defer c.rotationMu.Unlock()
	oldRotateAt, hadOld := c.rotations[key]
	delete(c.rotations, key)
	if hadOld && oldRotateAt.Equal(c.nextRotation) {
		c.resetRotationTimerLocked(time.Now())
	}
}

func (c *inherentXDSWorkloadController) stopAllTimers() {
	c.rotationMu.Lock()
	defer c.rotationMu.Unlock()
	if c.rotationTimer != nil {
		c.rotationTimer.Stop()
		c.rotationTimer = nil
	}
	c.nextRotation = time.Time{}
	for key := range c.rotations {
		delete(c.rotations, key)
	}
}

func (c *inherentXDSWorkloadController) resetRotationTimerLocked(now time.Time) {
	if c.rotationTimer != nil {
		c.rotationTimer.Stop()
		c.rotationTimer = nil
	}
	next, found := nextRotationTime(c.rotations)
	if !found {
		c.nextRotation = time.Time{}
		return
	}
	c.nextRotation = next
	delay := next.Sub(now)
	if delay < 0 {
		delay = 0
	}
	c.rotationTimer = time.AfterFunc(delay, c.flushDueRotations)
}

func (c *inherentXDSWorkloadController) flushDueRotations() {
	now := time.Now()
	due := make([]types.NamespacedName, 0)
	c.rotationMu.Lock()
	for key, rotateAt := range c.rotations {
		if !rotateAt.After(now) {
			due = append(due, key)
			delete(c.rotations, key)
		}
	}
	c.resetRotationTimerLocked(now)
	c.rotationMu.Unlock()

	sort.Slice(due, func(i, j int) bool {
		if due[i].Namespace != due[j].Namespace {
			return due[i].Namespace < due[j].Namespace
		}
		return due[i].Name < due[j].Name
	})
	for _, key := range due {
		c.queue.Add(key)
	}
}

func nextRotationTime(rotations map[types.NamespacedName]time.Time) (time.Time, bool) {
	var next time.Time
	found := false
	for _, rotateAt := range rotations {
		if !found || rotateAt.Before(next) {
			next = rotateAt
			found = true
		}
	}
	return next, found
}

func workloadRotationDelay(now, expireAt time.Time) time.Duration {
	lifetime := expireAt.Sub(now)
	if lifetime <= 0 {
		return time.Second
	}
	return lifetime * 4 / 5
}
