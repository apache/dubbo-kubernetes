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
	"sync"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/kube/gateway"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/features"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"
	"github.com/apache/dubbo-kubernetes/pkg/kube/multicluster"
	"github.com/apache/dubbo-kubernetes/pkg/log"
)

type remoteGatewayDeploymentController struct {
	stop      chan struct{}
	closeOnce sync.Once
}

func (c *remoteGatewayDeploymentController) Close() {
	c.closeOnce.Do(func() {
		close(c.stop)
	})
}

func (c *remoteGatewayDeploymentController) HasSynced() bool {
	return true
}

func (s *Server) initMulticlusterGatewayDeploymentControllers(args *DubboArgs) {
	if s.multiclusterController == nil || !features.EnableGatewayAPI || !features.EnableGatewayAPIDeploymentController {
		return
	}
	multicluster.BuildMultiClusterComponent(s.multiclusterController, func(remote *multicluster.Cluster) *remoteGatewayDeploymentController {
		c := &remoteGatewayDeploymentController{stop: make(chan struct{})}
		if remote == nil || remote.Client == nil || remote.ID == s.clusterID {
			return c
		}
		go func() {
			if !remote.Client.CrdWatcher().WaitForCRD(gvr.KubernetesGateway, c.stop) {
				return
			}
			controller := gateway.NewDeploymentController(remote.Client, remote.ID, s.environment,
				s.webhookInfo.getWebhookConfig, s.webhookInfo.addHandler, nil, args.Revision, args.Namespace)
			log.Infof("starting remote gateway deployment controller for cluster %s", remote.ID)
			controller.Run(c.stop)
		}()
		return c
	})
}
