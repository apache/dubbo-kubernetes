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

package controller

import (
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/config/kube/clustertrustbundle"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/features"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/keycertbundle"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/leaderelection"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/server"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/serviceregistry/aggregate"
	"github.com/apache/dubbo-kubernetes/pkg/kube/multicluster"
)

type kubeController struct {
	MeshServiceController *aggregate.Controller
	*Controller
	stop chan struct{}
}

func (k *kubeController) Close() {
	close(k.stop)
}

type Multicluster struct {
	// serverID of this planet instance used for leader election
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
		log.Infof("Initializing Kubernetes service registry %q", options.ClusterID)
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
	client := cluster.Client

	// run after WorkloadHandler is added
	m.opts.MeshServiceController.AddRegistryAndRun(kubeRegistry, clusterStopCh)

	go func() {
		var shouldLead bool
		if !configCluster {
			shouldLead = m.checkShouldLead()
			log.Infof("should join leader-election for cluster %s: %t", cluster.ID, shouldLead)
		}

		if m.distributeCACert && (shouldLead || configCluster) {
			if features.EnableClusterTrustBundles {
				// Block server exit on graceful termination of the leader controller.
				m.s.RunComponentAsyncAndWait("clustertrustbundle controller", func(_ <-chan struct{}) error {
					election := leaderelection.
						NewLeaderElectionMulticluster(options.SystemNamespace, m.serverID, leaderelection.NamespaceController, m.revision, !configCluster, client)
					// For config cluster (single-node deployment), disable leader election even if globally enabled
					// because there's only one instance and no need for election.
					if configCluster {
						election.SetEnabled(false)
					}
					election.AddRunFunction(func(leaderStop <-chan struct{}) {
						log.Infof("starting clustertrustbundle controller for cluster %s", cluster.ID)
						c := clustertrustbundle.NewController(client, m.caBundleWatcher)
						client.RunAndWait(clusterStopCh)
						c.Run(leaderStop)
					})
					election.Run(clusterStopCh)
					return nil
				})
			} else {
				// Block server exit on graceful termination of the leader controller.
				m.s.RunComponentAsyncAndWait("namespace controller", func(_ <-chan struct{}) error {
					election := leaderelection.
						NewLeaderElectionMulticluster(options.SystemNamespace, m.serverID, leaderelection.NamespaceController, m.revision, !configCluster, client)
					// For config cluster (single-node deployment), disable leader election even if globally enabled
					// because there's only one instance and no need for election.
					if configCluster {
						election.SetEnabled(false)
					}
					election.AddRunFunction(func(leaderStop <-chan struct{}) {
						log.Infof("starting namespace controller for cluster %s", cluster.ID)
						nc := NewNamespaceController(client, m.caBundleWatcher)
						// Start informers again. This fixes the case where informers for namespace do not start,
						// as we create them only after acquiring the leader lock
						// Note: stop here should be the overall planet stop, NOT the leader election stop. We are
						// basically lazy loading the informer, if we stop it when we lose the lock we will never
						// recreate it again.
						client.RunAndWait(clusterStopCh)
						nc.Run(leaderStop)
					})
					election.Run(clusterStopCh)
					return nil
				})
			}
		}
	}()
}

func (m *Multicluster) checkShouldLead() bool {
	var res bool
	return res
}
