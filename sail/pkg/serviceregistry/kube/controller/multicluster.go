package controller

import (
	kubelib "github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/multicluster"
	"github.com/apache/dubbo-kubernetes/sail/pkg/keycertbundle"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/server"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/aggregate"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/serviceentry"
	"k8s.io/klog/v2"
)

type kubeController struct {
	MeshServiceController *aggregate.Controller
	*Controller
	workloadEntryController *serviceentry.Controller
	stop                    chan struct{}
}

func (k *kubeController) Close() {
	close(k.stop)
}

type Multicluster struct {
	// serverID of this pilot instance used for leader election
	serverID string

	// options to use when creating kube controllers
	opts Options

	s server.Instance

	clusterLocal model.ClusterLocalProvider

	distributeCACert bool
	caBundleWatcher  *keycertbundle.Watcher
	revision         string

	component *multicluster.Component[*kubeController]
}

func NewMulticluster(
	serverID string,
	opts Options,
	caBundleWatcher *keycertbundle.Watcher,
	revision string,
	distributeCACert bool,
	clusterLocal model.ClusterLocalProvider,
	s server.Instance,
	controller *multicluster.Controller,
) *Multicluster {
	mc := &Multicluster{
		serverID:         serverID,
		opts:             opts,
		distributeCACert: distributeCACert,
		caBundleWatcher:  caBundleWatcher,
		revision:         revision,
		clusterLocal:     clusterLocal,
		s:                s,
	}
	mc.component = multicluster.BuildMultiClusterComponent(controller, func(cluster *multicluster.Cluster) *kubeController {
		stop := make(chan struct{})
		client := cluster.Client
		configCluster := opts.ClusterID == cluster.ID

		options := opts
		options.ClusterID = cluster.ID
		klog.Infof("Initializing Kubernetes service registry %q", options.ClusterID)
		options.ConfigCluster = configCluster
		kubeRegistry := NewController(client, options)
		kubeController := &kubeController{
			MeshServiceController: opts.MeshServiceController,
			Controller:            kubeRegistry,
			stop:                  stop,
		}
		mc.initializeCluster(cluster, kubeController, kubeRegistry, options, configCluster, stop)
		return kubeController
	})

	return mc
}

func (m *Multicluster) initializeCluster(cluster *multicluster.Cluster, kubeController *kubeController, kubeRegistry *Controller,
	options Options, configCluster bool, clusterStopCh <-chan struct{},
) {
	// run after WorkloadHandler is added
	m.opts.MeshServiceController.AddRegistryAndRun(kubeRegistry, clusterStopCh)
}

func (m *Multicluster) checkShouldLead(client kubelib.Client, systemNamespace string, stop <-chan struct{}) bool {
	var res bool
	return res
}
