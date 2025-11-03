package multicluster

import (
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

type Controller struct {
	namespace            string
	configClusterID      cluster.ID
	configCluster        *Cluster
	configClusterSyncers []ComponentConstraint

	queue           controllers.Queue
	secrets         kclient.Client[*corev1.Secret]
	configOverrides []func(*rest.Config)

	cs *ClusterStore

	meshWatcher mesh.Watcher
	handlers    []handler
}

func NewController(kubeclientset kube.Client, namespace string, clusterID cluster.ID,
	meshWatcher mesh.Watcher, configOverrides ...func(*rest.Config),
) *Controller {
	controller := &Controller{
		namespace:       namespace,
		configClusterID: clusterID,
		configCluster:   &Cluster{Client: kubeclientset, ID: clusterID},

		configOverrides: configOverrides,
		meshWatcher:     meshWatcher,
	}
	return controller
}

func (c *Controller) Run(stopCh <-chan struct{}) error {
	// run handlers for the config cluster; do not store this *Cluster in the ClusterStore or give it a SyncTimeout
	// this is done outside the goroutine, we should block other Run/startFuncs until this is registered
	c.configClusterSyncers = c.handleAdd(c.configCluster)
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

type handler interface {
	clusterAdded(cluster *Cluster) ComponentConstraint
	clusterUpdated(cluster *Cluster) ComponentConstraint
	clusterDeleted(clusterID cluster.ID)
	HasSynced() bool
}

type ComponentBuilder interface {
	registerHandler(h handler)
}

func BuildMultiClusterComponent[T ComponentConstraint](c ComponentBuilder, constructor func(cluster *Cluster) T) *Component[T] {
	comp := &Component[T]{
		constructor: constructor,
		clusters:    make(map[cluster.ID]T),
	}
	c.registerHandler(comp)
	return comp
}

func (c *Controller) registerHandler(h handler) {
	// Intentionally no lock. The controller today requires that handlers are registered before execution and not in parallel.
	c.handlers = append(c.handlers, h)
}
