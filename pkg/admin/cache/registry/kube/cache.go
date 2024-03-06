/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kube

import (
	"fmt"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/admin/cache"
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache/selector"
	"github.com/apache/dubbo-kubernetes/pkg/admin/constant"
	"github.com/apache/dubbo-kubernetes/pkg/core/kubeclient/client"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	appsv1Listers "k8s.io/client-go/listers/apps/v1"
	corev1Listers "k8s.io/client-go/listers/core/v1"
	kubeToolsCache "k8s.io/client-go/tools/cache"
)

var KubernetesCacheInstance *KubernetesCache

func NewKubernetesCache(kc *client.KubeClient, clusterScoped bool) *KubernetesCache {
	return &KubernetesCache{
		client:               kc,
		clusterScoped:        clusterScoped,
		namespaceCacheLister: make(map[string]*cacheLister),
		namespaceStopChan:    make(map[string]chan struct{}),
	}
}

type KubernetesCache struct {
	lock                 sync.RWMutex
	client               *client.KubeClient
	clusterScoped        bool
	refreshDuration      time.Duration
	clusterCacheLister   *cacheLister
	clusterStopChan      chan struct{}
	namespaceCacheLister map[string]*cacheLister
	namespaceStopChan    map[string]chan struct{}
}

type cacheLister struct {
	configMapLister   corev1Listers.ConfigMapLister
	daemonSetLister   appsv1Listers.DaemonSetLister
	deploymentLister  appsv1Listers.DeploymentLister
	endpointLister    corev1Listers.EndpointsLister
	podLister         corev1Listers.PodLister
	replicaSetLister  appsv1Listers.ReplicaSetLister
	serviceLister     corev1Listers.ServiceLister
	statefulSetLister appsv1Listers.StatefulSetLister

	cachesSynced []kubeToolsCache.InformerSynced
}

func (c *KubernetesCache) GetApplications(namespace string) ([]*cache.ApplicationModel, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	applicationSet := make(map[string]struct{})
	deployments, err := c.getCacheLister(namespace).deploymentLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	res := make([]*cache.ApplicationModel, 0)
	for _, deployment := range deployments {
		if _, ok := deployment.Labels[constant.ApplicationLabel]; ok {
			if _, exist := applicationSet[constant.ApplicationLabel]; !exist {
				applicationSet[deployment.Labels[constant.ApplicationLabel]] = struct{}{} // mark as exist
				res = append(res, &cache.ApplicationModel{
					Name: deployment.Name,
				})
			}
		}
	}

	return res, nil
}

func (c *KubernetesCache) GetApplicationsWithSelector(namespace string, selector selector.Selector) ([]*cache.ApplicationModel, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	applicationSet := make(map[string]struct{})
	deployments, err := c.getCacheLister(namespace).deploymentLister.List(selector.AsLabelsSelector())
	if err != nil {
		return nil, err
	}

	res := make([]*cache.ApplicationModel, 0)
	for _, deployment := range deployments {
		if _, ok := deployment.Labels[constant.ApplicationLabel]; ok {
			if _, exist := applicationSet[constant.ApplicationLabel]; !exist {
				applicationSet[deployment.Labels[constant.ApplicationLabel]] = struct{}{} // mark as exist
				res = append(res, &cache.ApplicationModel{
					Name: deployment.Name,
				})
			}
		}
	}

	return res, nil
}

func (c *KubernetesCache) GetWorkloads(namespace string) ([]*cache.WorkloadModel, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	res := make([]*cache.WorkloadModel, 0)

	deployments, err := c.getCacheLister(namespace).deploymentLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	for _, deployment := range deployments {
		if _, ok := deployment.Labels[constant.ApplicationLabel]; ok {
			res = append(res, &cache.WorkloadModel{
				Application: &cache.ApplicationModel{
					Name: deployment.Labels[constant.ApplicationLabel],
				},
				Name:   deployment.Name,
				Type:   constant.DeploymentType,
				Image:  deployment.Spec.Template.Spec.Containers[0].Image,
				Labels: deployment.Labels,
			})
		}
	}

	statefulSets, err := c.getCacheLister(namespace).statefulSetLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	for _, statefulSet := range statefulSets {
		if _, ok := statefulSet.Labels[constant.ApplicationLabel]; ok {
			res = append(res, &cache.WorkloadModel{
				Application: &cache.ApplicationModel{
					Name: statefulSet.Labels[constant.ApplicationLabel],
				},
				Name:   statefulSet.Name,
				Type:   constant.StatefulSetType,
				Image:  statefulSet.Spec.Template.Spec.Containers[0].Image,
				Labels: statefulSet.Labels,
			})
		}
	}

	daemonSets, err := c.getCacheLister(namespace).daemonSetLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	for _, daemonSet := range daemonSets {
		if _, ok := daemonSet.Labels[constant.ApplicationLabel]; ok {
			res = append(res, &cache.WorkloadModel{
				Application: &cache.ApplicationModel{
					Name: daemonSet.Labels[constant.ApplicationLabel],
				},
				Name:   daemonSet.Name,
				Type:   constant.DaemonSetType,
				Image:  daemonSet.Spec.Template.Spec.Containers[0].Image,
				Labels: daemonSet.Labels,
			})
		}
	}

	return res, nil
}

func (c *KubernetesCache) GetWorkloadsWithSelector(namespace string, selector selector.Selector) ([]*cache.WorkloadModel, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	res := make([]*cache.WorkloadModel, 0)

	deployments, err := c.getCacheLister(namespace).deploymentLister.List(selector.AsLabelsSelector())
	if err != nil {
		return nil, err
	}
	for _, deployment := range deployments {
		res = append(res, &cache.WorkloadModel{
			Application: &cache.ApplicationModel{
				Name: deployment.Labels[constant.ApplicationLabel],
			},
			Name:   deployment.Name,
			Type:   constant.DeploymentType,
			Image:  deployment.Spec.Template.Spec.Containers[0].Image,
			Labels: deployment.Labels,
		})
	}

	statefulSets, err := c.getCacheLister(namespace).statefulSetLister.List(selector.AsLabelsSelector())
	if err != nil {
		return nil, err
	}
	for _, statefulSet := range statefulSets {
		res = append(res, &cache.WorkloadModel{
			Application: &cache.ApplicationModel{
				Name: statefulSet.Labels[constant.ApplicationLabel],
			},
			Name:   statefulSet.Name,
			Type:   constant.StatefulSetType,
			Image:  statefulSet.Spec.Template.Spec.Containers[0].Image,
			Labels: statefulSet.Labels,
		})
	}

	daemonSets, err := c.getCacheLister(namespace).daemonSetLister.List(selector.AsLabelsSelector())
	if err != nil {
		return nil, err
	}
	for _, daemonSet := range daemonSets {
		res = append(res, &cache.WorkloadModel{
			Application: &cache.ApplicationModel{
				Name: daemonSet.Labels[constant.ApplicationLabel],
			},
			Name:   daemonSet.Name,
			Type:   constant.DaemonSetType,
			Image:  daemonSet.Spec.Template.Spec.Containers[0].Image,
			Labels: daemonSet.Labels,
		})
	}

	return res, nil
}

func (c *KubernetesCache) GetInstances(namespace string) ([]*cache.InstanceModel, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	res := make([]*cache.InstanceModel, 0)

	pods, err := c.getCacheLister(namespace).podLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	for _, pod := range pods {
		if _, ok := pod.Labels[constant.ApplicationLabel]; ok {
			res = append(res, &cache.InstanceModel{
				Application: &cache.ApplicationModel{
					Name: pod.Labels[constant.ApplicationLabel],
				},
				Workload: &cache.WorkloadModel{
					// TODO: implement me
				},
				Name:   pod.Name,
				Ip:     pod.Status.PodIP,
				Status: string(pod.Status.Phase),
				Node:   pod.Spec.NodeName,
				Labels: pod.Labels,
			})
		}
	}

	return res, nil
}

func (c *KubernetesCache) GetInstancesWithSelector(namespace string, selector selector.Selector) ([]*cache.InstanceModel, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	res := make([]*cache.InstanceModel, 0)

	pods, err := c.getCacheLister(namespace).podLister.List(selector.AsLabelsSelector())
	if err != nil {
		return nil, err
	}
	for _, pod := range pods {
		res = append(res, &cache.InstanceModel{
			Application: &cache.ApplicationModel{
				Name: pod.Labels[constant.ApplicationLabel],
			},
			Workload: &cache.WorkloadModel{
				// TODO: implement me
			},
			Name:   pod.Name,
			Ip:     pod.Status.PodIP,
			Status: string(pod.Status.Phase),
			Node:   pod.Spec.NodeName,
			Labels: pod.Labels,
		})
	}

	return res, nil
}

func (c *KubernetesCache) GetServices(namespace string) ([]*cache.ServiceModel, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	res := make([]*cache.ServiceModel, 0)
	services, err := c.getCacheLister(namespace).serviceLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	for _, service := range services {
		if _, ok := service.Labels[constant.ApplicationLabel]; ok {
			res = append(res, &cache.ServiceModel{
				Application: &cache.ApplicationModel{
					Name: service.Labels[constant.ApplicationLabel],
				},
				Category:   constant.ProviderSide,
				Name:       service.Name,
				Labels:     service.Labels,
				ServiceKey: service.Labels[constant.ServiceKeyLabel],
				Group:      service.Labels[constant.GroupLabel],
				Version:    service.Labels[constant.VersionLabel],
			})
		}
	}

	return res, nil
}

func (c *KubernetesCache) GetServicesWithSelector(namespace string, selector selector.Selector) ([]*cache.ServiceModel, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	res := make([]*cache.ServiceModel, 0)
	services, err := c.getCacheLister(namespace).serviceLister.List(selector.AsLabelsSelector())
	if err != nil {
		return nil, err
	}
	for _, service := range services {
		res = append(res, &cache.ServiceModel{
			Application: &cache.ApplicationModel{
				Name: service.Labels[constant.ApplicationLabel],
			},
			Category:   constant.ProviderSide,
			Name:       service.Name,
			Labels:     service.Labels,
			ServiceKey: service.Labels[constant.ServiceKeyLabel],
			Group:      service.Labels[constant.GroupLabel],
			Version:    service.Labels[constant.VersionLabel],
		})
	}

	return res, nil
}

// getCacheLister returns the cache lister for the given namespace if the cache is namespace scoped, otherwise it returns the cluster cache lister.
func (c *KubernetesCache) getCacheLister(namespace string) *cacheLister {
	if c.clusterScoped {
		return c.clusterCacheLister
	}
	return c.namespaceCacheLister[namespace]
}

func (c *KubernetesCache) createInformer(namespace string) informers.SharedInformerFactory {
	var informer informers.SharedInformerFactory
	if c.clusterScoped {
		informer = informers.NewSharedInformerFactoryWithOptions(c.client, c.refreshDuration)
	} else {
		informer = informers.NewSharedInformerFactoryWithOptions(c.client, c.refreshDuration, informers.WithNamespace(namespace))
	}

	lister := &cacheLister{
		deploymentLister:  informer.Apps().V1().Deployments().Lister(),
		statefulSetLister: informer.Apps().V1().StatefulSets().Lister(),
		daemonSetLister:   informer.Apps().V1().DaemonSets().Lister(),
		serviceLister:     informer.Core().V1().Services().Lister(),
		endpointLister:    informer.Core().V1().Endpoints().Lister(),
		podLister:         informer.Core().V1().Pods().Lister(),
		replicaSetLister:  informer.Apps().V1().ReplicaSets().Lister(),
		configMapLister:   informer.Core().V1().ConfigMaps().Lister(),
	}
	lister.cachesSynced = append(lister.cachesSynced,
		informer.Apps().V1().Deployments().Informer().HasSynced,
		informer.Apps().V1().StatefulSets().Informer().HasSynced,
		informer.Apps().V1().DaemonSets().Informer().HasSynced,
		informer.Core().V1().Services().Informer().HasSynced,
		informer.Core().V1().Endpoints().Informer().HasSynced,
		informer.Core().V1().Pods().Informer().HasSynced,
		informer.Apps().V1().ReplicaSets().Informer().HasSynced,
		informer.Core().V1().ConfigMaps().Informer().HasSynced,
	)

	if c.clusterScoped {
		c.clusterCacheLister = lister
	} else {
		c.namespaceCacheLister[namespace] = lister
	}

	return informer
}

// startInformers starts informers to sync data from kubernetes.
func (c *KubernetesCache) startInformers(namespaces ...string) error {
	logger.Infof("[dubbo-cp cache] Starting informers")
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.clusterScoped {
		if err := c.startInformer(""); err != nil {
			return err
		}
	} else {
		for _, namespace := range namespaces {
			if err := c.startInformer(namespace); err != nil {
				return err
			}
		}
	}
	return nil
}

// startInformer starts the informer for the given namespace.
func (c *KubernetesCache) startInformer(namespace string) error {
	informer := c.createInformer(namespace)
	var scope string
	stop := make(chan struct{})
	if c.clusterScoped {
		scope = "cluster-scoped"
		c.clusterStopChan = stop
	} else {
		scope = fmt.Sprintf("namespace-scoped for namespace: %s", namespace)
		c.namespaceStopChan[namespace] = stop
	}
	logger.Debugf("[dubbo-cp cache] Starting %s informer", scope)
	go informer.Start(stop)

	logger.Infof("[dubbo-cp cache] Waiting for %s informer caches to sync", scope)
	if !kubeToolsCache.WaitForCacheSync(stop, c.getCacheLister(namespace).cachesSynced...) {
		logger.Errorf("[dubbo-cp cache] Failed to sync %s informer caches", scope)
		return fmt.Errorf("failed to sync %s informer caches", scope)
	}
	logger.Infof("[dubbo-cp cache] Synced %s informer caches", scope)

	return nil
}

func (c *KubernetesCache) stopInformers() {
	logger.Infof("[dubbo-cp cache] Stopping informers")
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.clusterScoped {
		c.stopInformer("")
	} else {
		for namespace := range c.namespaceStopChan {
			logger.Debugf("[dubbo-cp cache] Stopping informer for namespace: %s", namespace)
			c.stopInformer(namespace)
		}
	}
}

func (c *KubernetesCache) stopInformer(namespace string) {
	if c.clusterScoped {
		close(c.clusterStopChan)
	} else {
		if ch, ok := c.namespaceStopChan[namespace]; ok {
			close(ch)
			delete(c.namespaceStopChan, namespace)
			delete(c.namespaceCacheLister, namespace)
		}
	}
}
