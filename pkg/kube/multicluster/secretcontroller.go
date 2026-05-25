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

package multicluster

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	RemoteClusterSecretLabel          = "dubbo.apache.org/remote-cluster"
	RemoteClusterSecretLabelValue     = "true"
	RemoteClusterSecretKubeconfigKey  = "kubeconfig"
	RemoteClusterSecretClusterNameKey = "cluster-name"

	defaultRemoteClusterSyncTimeout = 30 * time.Second
)

var log = dubbolog.RegisterScope("multicluster", "multicluster debugging")

type handler interface {
	clusterAdded(cluster *Cluster) ComponentConstraint
	clusterUpdated(cluster *Cluster) ComponentConstraint
	clusterDeleted(clusterID cluster.ID)
	HasSynced() bool
}

type ComponentBuilder interface {
	registerHandler(h handler)
}

type Controller struct {
	namespace            string
	configClusterID      cluster.ID
	configCluster        *Cluster
	configClusterSyncers []ComponentConstraint
	syncTimeout          time.Duration

	queue           controllers.Queue
	secrets         kclient.Client[*corev1.Secret]
	configOverrides []func(*rest.Config)

	cs *ClusterStore

	meshWatcher mesh.Watcher
	handlers    []handler

	mu                 sync.RWMutex
	secretToCluster    map[types.NamespacedName]cluster.ID
	clusterToSecret    map[cluster.ID]types.NamespacedName
	newRemoteClient    func([]byte, cluster.ID, ...func(*rest.Config)) (kube.Client, error)
	startRemoteCluster func(*Cluster)
}

func NewController(kubeclientset kube.Client, namespace string, clusterID cluster.ID,
	meshWatcher mesh.Watcher, configOverrides ...func(*rest.Config),
) *Controller {
	controller := &Controller{
		namespace:       namespace,
		configClusterID: clusterID,
		configCluster:   &Cluster{Client: kubeclientset, ID: clusterID},
		syncTimeout:     defaultRemoteClusterSyncTimeout,
		configOverrides: configOverrides,
		meshWatcher:     meshWatcher,
		cs:              NewClusterStore(),
		secretToCluster: make(map[types.NamespacedName]cluster.ID),
		clusterToSecret: make(map[cluster.ID]types.NamespacedName),
		newRemoteClient: newRemoteClient,
	}
	controller.startRemoteCluster = func(cluster *Cluster) {
		cluster.Start(controller.syncTimeout)
	}
	controller.queue = controllers.NewQueue("multicluster secret controller",
		controllers.WithReconciler(controller.reconcile),
		controllers.WithMaxAttempts(5),
	)
	controller.secrets = kclient.NewFiltered[*corev1.Secret](kubeclientset, kclient.Filter{
		Namespace:     namespace,
		LabelSelector: RemoteClusterSecretLabel + "=" + RemoteClusterSecretLabelValue,
	})
	controller.secrets.AddEventHandler(controllers.ObjectHandler(controller.queue.AddObject))
	return controller
}

func BuildMultiClusterComponent[T ComponentConstraint](c ComponentBuilder, constructor func(cluster *Cluster) T) *Component[T] {
	comp := &Component[T]{
		constructor: constructor,
		clusters:    make(map[cluster.ID]T),
	}
	c.registerHandler(comp)
	return comp
}

func (c *Controller) Run(stopCh <-chan struct{}) error {
	// run handlers for the config cluster; do not store this *Cluster in the ClusterStore or give it a SyncTimeout
	// this is done outside the goroutine, we should block other Run/startFuncs until this is registered
	c.configClusterSyncers = c.handleAdd(c.configCluster)
	go func() {
		c.secrets.Start(stopCh)
		if !kube.WaitForCacheSync("multicluster secrets", stopCh, c.secrets.HasSynced) {
			c.queue.ShutDownEarly()
			return
		}
		c.queue.Run(stopCh)
	}()
	return nil
}

func (c *Controller) HasSynced() bool {
	if !c.queue.HasSynced() {
		return false
	}
	// Check all config cluster components are synced
	// c.ConfigClusterHandler.HasSynced does not work; config cluster is handle specially
	if !kube.AllSynced(c.configClusterSyncers) {
		return false
	}
	// Check all remote clusters are synced (or timed out)
	return c.cs.HasSynced()
}

func (c *Controller) handleAdd(cluster *Cluster) []ComponentConstraint {
	syncers := make([]ComponentConstraint, 0, len(c.handlers))
	for _, handler := range c.handlers {
		syncers = append(syncers, handler.clusterAdded(cluster))
	}
	return syncers
}

func (c *Controller) handleDelete(key cluster.ID) {
	for _, handler := range c.handlers {
		handler.clusterDeleted(key)
	}
}

func (c *Controller) registerHandler(h handler) {
	// Intentionally no lock. The controller today requires that handlers are registered before execution and not in parallel.
	c.handlers = append(c.handlers, h)
}

func (c *Controller) reconcile(key types.NamespacedName) error {
	secret := c.secrets.Get(key.Name, key.Namespace)
	if secret == nil {
		c.deleteSecret(key)
		return nil
	}
	if err := c.reconcileSecret(key, secret); err != nil {
		log.Warnf("ignoring remote cluster secret %s/%s: %v", key.Namespace, key.Name, err)
		c.deleteSecret(key)
	}
	return nil
}

func (c *Controller) reconcileSecret(key types.NamespacedName, secret *corev1.Secret) error {
	clusterID, kubeconfig, err := parseRemoteClusterSecret(secret)
	if err != nil {
		return err
	}
	if clusterID == c.configClusterID {
		return fmt.Errorf("remote cluster %q matches the config cluster", clusterID)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if existingKey, ok := c.clusterToSecret[clusterID]; ok && existingKey != key {
		return fmt.Errorf("remote cluster %q is already registered by secret %s/%s", clusterID, existingKey.Namespace, existingKey.Name)
	}
	if previousID, ok := c.secretToCluster[key]; ok && previousID != clusterID {
		c.removeClusterLocked(previousID)
	}

	kubeconfigSha := sha256.Sum256(kubeconfig)
	if existing, ok := c.cs.Get(clusterID); ok && existing.kubeConfigSha == kubeconfigSha {
		c.secretToCluster[key] = clusterID
		c.clusterToSecret[clusterID] = key
		return nil
	}

	client, err := c.newRemoteClient(kubeconfig, clusterID, c.configOverrides...)
	if err != nil {
		return err
	}
	cluster := NewRemoteCluster(clusterID, client, kubeconfigSha)
	if existing, ok := c.cs.Get(clusterID); ok {
		for _, handler := range c.handlers {
			handler.clusterUpdated(cluster)
		}
		existing.Close()
	} else {
		c.handleAdd(cluster)
	}
	c.cs.Store(cluster)
	c.secretToCluster[key] = clusterID
	c.clusterToSecret[clusterID] = key
	c.startRemoteCluster(cluster)
	return nil
}

func (c *Controller) deleteSecret(key types.NamespacedName) {
	c.mu.Lock()
	defer c.mu.Unlock()
	clusterID, ok := c.secretToCluster[key]
	if !ok {
		return
	}
	c.removeClusterLocked(clusterID)
}

func (c *Controller) removeClusterLocked(clusterID cluster.ID) {
	cluster, ok := c.cs.Delete(clusterID)
	if !ok {
		return
	}
	key := c.clusterToSecret[clusterID]
	delete(c.clusterToSecret, clusterID)
	delete(c.secretToCluster, key)
	c.handleDelete(clusterID)
	cluster.Close()
}

func parseRemoteClusterSecret(secret *corev1.Secret) (cluster.ID, []byte, error) {
	if secret.Type != corev1.SecretTypeOpaque {
		return "", nil, fmt.Errorf("secret type %q must be %q", secret.Type, corev1.SecretTypeOpaque)
	}
	clusterName := strings.TrimSpace(string(secret.Data[RemoteClusterSecretClusterNameKey]))
	if clusterName == "" {
		return "", nil, fmt.Errorf("missing %q", RemoteClusterSecretClusterNameKey)
	}
	kubeconfig := secret.Data[RemoteClusterSecretKubeconfigKey]
	if len(kubeconfig) == 0 {
		return "", nil, fmt.Errorf("missing %q", RemoteClusterSecretKubeconfigKey)
	}
	return cluster.ID(clusterName), kubeconfig, nil
}

func newRemoteClient(kubeconfig []byte, clusterID cluster.ID, configOverrides ...func(*rest.Config)) (kube.Client, error) {
	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("parse kubeconfig: %w", err)
	}
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build kubeconfig: %w", err)
	}
	for _, override := range configOverrides {
		override(restConfig)
	}
	restConfig = kube.SetRestDefaults(restConfig)
	client, err := kube.NewClient(kube.NewClientConfigForRestConfig(restConfig), clusterID)
	if err != nil {
		return nil, err
	}
	return kube.EnableCrdWatcher(client), nil
}
