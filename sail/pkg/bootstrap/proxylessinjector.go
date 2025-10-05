package bootstrap

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/env"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/apache/dubbo-kubernetes/pkg/webhooks"
	"github.com/apache/dubbo-kubernetes/sail/pkg/features"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
)

const (
	webhookName                  = "proxyless-injector.dubbo.io"
	defaultInjectorConfigMapName = "dubbo-proxyless-injector"
)

var injectionEnabled = env.Register("INJECT_ENABLED", true, "Enable mutating webhook handler.")

func (s *Server) initProxylessInjector(args *SailArgs) (*inject.Webhook, error) {
	// currently the constant: "./var/lib/dubbo/inject"
	injectPath := args.InjectionOptions.InjectionDirectory
	if injectPath == "" || !injectionEnabled.Get() {
		klog.Infof("Skipping proxyless injector, injection path is missing or disabled.")
		return nil, nil
	}

	// If the injection config exists either locally or remotely, we will set up injection.
	var watcher inject.Watcher
	if _, err := os.Stat(filepath.Join(injectPath, "config")); !os.IsNotExist(err) {
		configFile := filepath.Join(injectPath, "config")
		valuesFile := filepath.Join(injectPath, "values")
		watcher, err = inject.NewFileWatcher(configFile, valuesFile)
		if err != nil {
			return nil, err
		}
	} else if s.kubeClient != nil {
		configMapName := getInjectorConfigMapName(args.Revision)
		cms := s.kubeClient.Kube().CoreV1().ConfigMaps(args.Namespace)
		if _, err := cms.Get(context.TODO(), configMapName, metav1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				klog.Infof("Skipping proxyless injector, template not found")
				return nil, nil
			}
			return nil, err
		}
		watcher = inject.NewConfigMapWatcher(s.kubeClient, args.Namespace, configMapName, "config", "values")
	} else {
		klog.Infof("Skipping proxyless injector, template not found")
		return nil, nil
	}

	klog.Info("initializing proxyless injector")

	parameters := inject.WebhookParameters{
		Watcher:  watcher,
		Env:      s.environment,
		Mux:      s.httpsMux,
		Revision: args.Revision,
	}

	wh, err := inject.NewWebhook(parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to create injection webhook: %v", err)
	}

	if features.InjectionWebhookConfigName != "" {
		s.addStartFunc("injection patcher", func(stop <-chan struct{}) error {
			patcher, err := webhooks.NewWebhookCertPatcher(s.kubeClient, webhookName, args.Revision, s.dubbodCertBundleWatcher)
			if err != nil {
				klog.Errorf("failed to create webhook cert patcher: %v", err)
				return nil
			}

			go patcher.Run(stop)
			return nil
		})
	}

	s.addStartFunc("injection server", func(stop <-chan struct{}) error {
		wh.Run(stop)
		return nil
	})
	return wh, nil
}

func getInjectorConfigMapName(revision string) string {
	name := defaultInjectorConfigMapName
	if revision == "" || revision == "default" {
		return name
	}
	return name + "-" + revision
}
