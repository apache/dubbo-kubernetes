package kube

import (
	"fmt"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/core/kubeclient/client"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"k8s.io/client-go/informers"
	appsv1 "k8s.io/client-go/listers/apps/v1"
	corev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
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
	configMapLister   corev1.ConfigMapLister
	daemonSetLister   appsv1.DaemonSetLister
	deploymentLister  appsv1.DeploymentLister
	endpointLister    corev1.EndpointsLister
	podLister         corev1.PodLister
	replicaSetLister  appsv1.ReplicaSetLister
	serviceLister     corev1.ServiceLister
	statefulSetLister appsv1.StatefulSetLister

	cachesSynced []cache.InformerSynced
}

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
	if !cache.WaitForCacheSync(stop, c.getCacheLister(namespace).cachesSynced...) {
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

