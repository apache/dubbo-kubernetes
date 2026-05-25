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
