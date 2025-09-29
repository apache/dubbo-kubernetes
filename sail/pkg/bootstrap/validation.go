package bootstrap

import (
	"k8s.io/klog/v2"
)

func (s *Server) initConfigValidation(args *SailArgs) error {
	if s.kubeClient == nil {
		return nil
	}
	klog.Info("initializing config validator")
	return nil
}
