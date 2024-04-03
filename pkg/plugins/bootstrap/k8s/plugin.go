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

package k8s

import (
	"context"
	"time"
)

import (
	"github.com/pkg/errors"

	kube_core "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	kube_runtime "k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/rest"

	kube_ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	kube_client "sigs.k8s.io/controller-runtime/pkg/client"
	kube_manager "sigs.k8s.io/controller-runtime/pkg/manager"
	kube_metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	kube_webhook "sigs.k8s.io/controller-runtime/pkg/webhook"
)

import (
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_plugins "github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	k8s_common "github.com/apache/dubbo-kubernetes/pkg/plugins/common/k8s"
	k8s_extensions "github.com/apache/dubbo-kubernetes/pkg/plugins/extensions/k8s"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s"
	"github.com/apache/dubbo-kubernetes/pkg/util/pointer"
)

var _ core_plugins.BootstrapPlugin = &plugin{}

var log = core.Log.WithName("plugins").WithName("bootstrap").WithName("k8s")

type plugin struct{}

func init() {
	core_plugins.Register(core_plugins.Kubernetes, &plugin{})
}

func (p *plugin) BeforeBootstrap(b *core_runtime.Builder, cfg core_plugins.PluginConfig) error {
	// 半托管模式和纯k8s模式都可以使用这一个插件
	if b.Config().DeployMode == config_core.UniversalMode {
		return nil
	}
	scheme, err := NewScheme()
	if err != nil {
		return err
	}
	restClientConfig := kube_ctrl.GetConfigOrDie()
	restClientConfig.QPS = float32(b.Config().Runtime.Kubernetes.ClientConfig.Qps)
	restClientConfig.Burst = b.Config().Runtime.Kubernetes.ClientConfig.BurstQps

	systemNamespace := b.Config().Store.Kubernetes.SystemNamespace
	mgr, err := kube_ctrl.NewManager(
		restClientConfig,
		kube_ctrl.Options{
			Scheme: scheme,
			Cache: cache.Options{
				DefaultUnsafeDisableDeepCopy: pointer.To(true),
			},
			// Admission WebHook Server
			WebhookServer: kube_webhook.NewServer(kube_webhook.Options{
				Host:    b.Config().Runtime.Kubernetes.AdmissionServer.Address,
				Port:    int(b.Config().Runtime.Kubernetes.AdmissionServer.Port),
				CertDir: b.Config().Runtime.Kubernetes.AdmissionServer.CertDir,
			}),
			LeaderElection:          true,
			LeaderElectionID:        "cp-leader-lease",
			LeaderElectionNamespace: systemNamespace,
			Logger:                  core.Log.WithName("kube-manager"),
			LeaseDuration:           &b.Config().Runtime.Kubernetes.LeaderElection.LeaseDuration.Duration,
			RenewDeadline:           &b.Config().Runtime.Kubernetes.LeaderElection.RenewDeadline.Duration,
			// Disable metrics bind address as we use kube metrics registry directly.
			Metrics: kube_metricsserver.Options{
				BindAddress: "0",
			},
		},
	)
	if err != nil {
		return err
	}

	secretClient, err := createSecretClient(b.AppCtx(), scheme, systemNamespace, restClientConfig, mgr.GetRESTMapper())
	if err != nil {
		return err
	}

	b.WithExtensions(k8s_extensions.NewManagerContext(b.Extensions(), mgr))
	b.WithComponentManager(&kubeComponentManager{Manager: mgr})

	b.WithExtensions(k8s_extensions.NewSecretClientContext(b.Extensions(), secretClient))
	if expTime := b.Config().Runtime.Kubernetes.MarshalingCacheExpirationTime.Duration; expTime > 0 {
		b.WithExtensions(k8s_extensions.NewResourceConverterContext(b.Extensions(), k8s.NewCachingConverter(expTime)))
	} else {
		b.WithExtensions(k8s_extensions.NewResourceConverterContext(b.Extensions(), k8s.NewSimpleConverter()))
	}
	b.WithExtensions(k8s_extensions.NewCompositeValidatorContext(b.Extensions(), &k8s_common.CompositeValidator{}))
	return nil
}

// We need separate client for Secrets, because we don't have (get/list/watch) RBAC for all namespaces / cluster scope.
// Kubernetes cache lists resources under the hood from all Namespace unless we specify the "Namespace" in Options.
// If we try to use regular cached client for Secrets then we will see following error: E1126 10:42:52.097662       1 reflector.go:178] pkg/mod/k8s.io/client-go@v0.18.9/tools/cache/reflector.go:125: Failed to list *v1.Secret: secrets is forbidden: User "system:serviceaccount:dubbo-system:dubbo-control-plane" cannot list resource "secrets" in API group "" at the cluster scope
// We cannot specify this Namespace parameter for the main cache in ControllerManager because it affect all the resources, therefore we need separate client with cache for Secrets.
// The alternative was to use non-cached client, but it had performance problems.
func createSecretClient(appCtx context.Context, scheme *kube_runtime.Scheme, systemNamespace string, config *rest.Config, restMapper meta.RESTMapper) (kube_client.Client, error) {
	resyncPeriod := 10 * time.Hour // default resyncPeriod in Kubernetes
	kubeCache, err := cache.New(config, cache.Options{
		Scheme:            scheme,
		Mapper:            restMapper,
		SyncPeriod:        &resyncPeriod,
		DefaultNamespaces: map[string]cache.Config{systemNamespace: {}},
	})
	if err != nil {
		return nil, err
	}

	// We are listing secrets by our custom "type", therefore we need to add index by this field into cache
	err = kubeCache.IndexField(appCtx, &kube_core.Secret{}, "type", func(object kube_client.Object) []string {
		secret := object.(*kube_core.Secret)
		return []string{string(secret.Type)}
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not add index of Secret cache by field 'type'")
	}

	// According to ControllerManager code, cache needs to start before all the Runnables (our Components)
	// So we need separate go routine to start a cache and then wait for cache
	go func() {
		if err := kubeCache.Start(appCtx); err != nil {
			// According to implementations, there is no case when error is returned. It just for the Runnable contract.
			log.Error(err, "could not start the secret k8s cache")
		}
	}()

	if ok := kubeCache.WaitForCacheSync(appCtx); !ok {
		// ControllerManager ignores case when WaitForCacheSync returns false.
		// It might be a better idea to return an error and stop the Control Plane altogether, but sticking to return error for now.
		core.Log.Error(errors.New("could not sync secret cache"), "failed to wait for cache")
	}

	return kube_client.New(config, kube_client.Options{
		Scheme: scheme,
		Mapper: restMapper,
		Cache: &kube_client.CacheOptions{
			Reader: kubeCache,
		},
	})
}

func (p *plugin) AfterBootstrap(b *core_runtime.Builder, _ core_plugins.PluginConfig) error {
	if b.Config().DeployMode != config_core.KubernetesMode {
		return nil
	}

	return nil
}

func (p *plugin) Name() core_plugins.PluginName {
	return core_plugins.Kubernetes
}

func (p *plugin) Order() int {
	return core_plugins.EnvironmentPreparingOrder
}

type kubeComponentManager struct {
	kube_ctrl.Manager
	gracefulComponents []component.GracefulComponent
}

var _ component.Manager = &kubeComponentManager{}

func (cm *kubeComponentManager) Start(done <-chan struct{}) error {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		<-done
	}()

	defer cm.waitForDone()

	if err := cm.Manager.Start(ctx); err != nil {
		return errors.Wrap(err, "error running Kubernetes Manager")
	}
	return nil
}

// Extra check that component.Component implements LeaderElectionRunnable so the leader election works so we won't break leader election on K8S when refactoring component.Component
var _ kube_manager.LeaderElectionRunnable = component.ComponentFunc(func(i <-chan struct{}) error {
	return nil
})

func (k *kubeComponentManager) Add(components ...component.Component) error {
	for _, c := range components {
		if gc, ok := c.(component.GracefulComponent); ok {
			k.gracefulComponents = append(k.gracefulComponents, gc)
		}
		if err := k.Manager.Add(&componentRunnableAdaptor{Component: c}); err != nil {
			return err
		}
	}
	return nil
}

func (k *kubeComponentManager) waitForDone() {
	for _, gc := range k.gracefulComponents {
		gc.WaitForDone()
	}
}

// This adaptor is required unless component.Component takes a context as input
type componentRunnableAdaptor struct {
	component.Component
}

func (c componentRunnableAdaptor) Start(ctx context.Context) error {
	return c.Component.Start(ctx.Done())
}

func (c componentRunnableAdaptor) NeedLeaderElection() bool {
	return c.Component.NeedLeaderElection()
}

var (
	_ kube_manager.LeaderElectionRunnable = &componentRunnableAdaptor{}
	_ kube_manager.Runnable               = &componentRunnableAdaptor{}
)
