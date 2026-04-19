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
	"strings"
	"sync"
	"time"

	pkgbootstrap "github.com/apache/dubbo-kubernetes/pkg/bootstrap"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	meshconfig "github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/dubboagency/grpcxds"
	kubelib "github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	pkgmodel "github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/spiffe"

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
	if pod == nil || !shouldManageProxylessGRPCPod(pod) || pod.DeletionTimestamp != nil {
		c.clearRotation(key)
		return c.deleteSecret(key)
	}
	if pod.Status.PodIP == "" {
		return nil
	}

	desired, expireAt, err := c.buildSecret(pod)
	if err != nil {
		return err
	}

	secrets := c.server.kubeClient.Kube().CoreV1().Secrets(pod.Namespace)
	current, err := secrets.Get(context.Background(), desired.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			if _, err := secrets.Create(context.Background(), desired, metav1.CreateOptions{}); err != nil {
				return err
			}
		} else {
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

func (c *proxylessGRPCWorkloadController) buildSecret(pod *corev1.Pod) (*corev1.Secret, time.Time, error) {
	bootstrapJSON, err := c.buildBootstrapJSON(pod)
	if err != nil {
		return nil, time.Time{}, err
	}

	certChain, keyPEM, rootCert, expireAt, err := c.issueWorkloadCertificate(pod)
	if err != nil {
		return nil, time.Time{}, err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inject.ProxylessGRPCSecretName(pod.Name),
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

func (c *proxylessGRPCWorkloadController) buildBootstrapJSON(pod *corev1.Pod) ([]byte, error) {
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

	serviceNode := strings.Join([]string{
		string(pkgmodel.Proxyless),
		pod.Status.PodIP,
		pod.Name + "." + pod.Namespace,
		fmt.Sprintf("%s.svc.%s", pod.Namespace, domainSuffix),
	}, serviceNodeSeparator)

	node, err := pkgbootstrap.GetNodeMetaData(pkgbootstrap.MetadataOptions{
		ID:          serviceNode,
		InstanceIPs: []string{pod.Status.PodIP},
		ProxyConfig: proxyConfig,
		Envs: []string{
			"DUBBO_META_GENERATOR=grpc",
			"DUBBO_META_CLUSTER_ID=" + constants.DefaultClusterName,
			"DUBBO_META_NAMESPACE=" + pod.Namespace,
			"DUBBO_META_MESH_ID=" + trustDomain,
			"TRUST_DOMAIN=" + trustDomain,
			"POD_NAME=" + pod.Name,
			"POD_NAMESPACE=" + pod.Namespace,
			"INSTANCE_IP=" + pod.Status.PodIP,
			"SERVICE_ACCOUNT=" + serviceAccount,
		},
	})
	if err != nil {
		return nil, err
	}

	bootstrapCfg, err := grpcxds.GenerateBootstrap(grpcxds.GenerateBootstrapOptions{
		Node:             node,
		DiscoveryAddress: proxyConfig.GetDiscoveryAddress(),
		CertDir:          inject.ProxylessXDSMountPath,
	})
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(bootstrapCfg, "", "  ")
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
	if templates == "" {
		return false
	}
	for _, templateName := range strings.Split(templates, ",") {
		if strings.TrimSpace(templateName) == inject.ProxylessGRPCTemplateName {
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
