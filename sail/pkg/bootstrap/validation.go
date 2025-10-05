package bootstrap

import (
	"github.com/apache/dubbo-kubernetes/pkg/webhooks/server"
	"github.com/apache/dubbo-kubernetes/pkg/webhooks/validation/controller"
	"github.com/apache/dubbo-kubernetes/sail/pkg/features"
	"k8s.io/klog/v2"
)

func (s *Server) initConfigValidation(args *SailArgs) error {
	if s.kubeClient == nil {
		return nil
	}
	klog.Info("initializing config validator")
	params := server.Options{
		DomainSuffix: args.RegistryOptions.KubeOptions.DomainSuffix,
		Mux:          s.httpsMux,
	}
	_, err := server.New(params)
	if err != nil {
		return err
	}
	s.readinessFlags.configValidationReady.Store(true)

	if features.ValidationWebhookConfigName != "" && s.kubeClient != nil {
		s.addStartFunc("validation controller", func(stop <-chan struct{}) error {
			klog.Infof("Starting validation controller")
			go controller.NewValidatingWebhookController(
				s.kubeClient, args.Revision, args.Namespace, s.dubbodCertBundleWatcher).Run(stop)
			return nil
		})
	}
	return nil
}
