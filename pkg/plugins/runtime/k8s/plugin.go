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
	"github.com/pkg/errors"
	kube_core "k8s.io/api/core/v1"

	kube_ctrl "sigs.k8s.io/controller-runtime"
	kube_webhook "sigs.k8s.io/controller-runtime/pkg/webhook"
	kube_admission "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

import (
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/zone"
	core_plugins "github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	core_registry "github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	k8s_common "github.com/apache/dubbo-kubernetes/pkg/plugins/common/k8s"
	k8s_extensions "github.com/apache/dubbo-kubernetes/pkg/plugins/extensions/k8s"
	k8s_registry "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/pkg/registry"
	k8s_controllers "github.com/apache/dubbo-kubernetes/pkg/plugins/runtime/k8s/controllers"
	k8s_webhooks "github.com/apache/dubbo-kubernetes/pkg/plugins/runtime/k8s/webhooks"
)

var log = core.Log.WithName("plugin").WithName("runtime").WithName("k8s")

var _ core_plugins.RuntimePlugin = &plugin{}

type plugin struct{}

func init() {
	core_plugins.Register(core_plugins.Kubernetes, &plugin{})
}

func (p *plugin) Customize(rt core_runtime.Runtime) error {
	if rt.Config().DeployMode != config_core.KubernetesMode {
		return nil
	}
	mgr, ok := k8s_extensions.FromManagerContext(rt.Extensions())
	if !ok {
		return errors.Errorf("k8s controller runtime Manager hasn't been configured")
	}

	converter, ok := k8s_extensions.FromResourceConverterContext(rt.Extensions())
	if !ok {
		return errors.Errorf("k8s resource converter hasn't been configured")
	}

	if err := addControllers(mgr, rt, converter); err != nil {
		return err
	}

	// Mutators and Validators convert resources from Request (not from the Store)
	// these resources doesn't have ResourceVersion, we can't cache them
	//simpleConverter := k8s.NewSimpleConverter()
	//if err := addValidators(mgr, rt, simpleConverter); err != nil {
	//	return err
	//}
	//
	//if err := addMutators(mgr, rt, simpleConverter); err != nil {
	//	return err
	//}

	return nil
}

func addControllers(mgr kube_ctrl.Manager, rt core_runtime.Runtime, converter k8s_common.Converter) error {
	if err := addPodReconciler(mgr, rt, converter); err != nil {
		return err
	}
	return nil
}

func addPodReconciler(mgr kube_ctrl.Manager, rt core_runtime.Runtime, converter k8s_common.Converter) error {
	reconciler := &k8s_controllers.PodReconciler{
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor("k8s.dubbo.io/dataplane-generator"),
		Scheme:        mgr.GetScheme(),
		Log:           core.Log.WithName("controllers").WithName("Pod"),
		PodConverter: k8s_controllers.PodConverter{
			ServiceGetter: mgr.GetClient(),
			NodeGetter:    mgr.GetClient(),
			InboundConverter: k8s_controllers.InboundConverter{
				NameExtractor: k8s_controllers.NameExtractor{
					ReplicaSetGetter: mgr.GetClient(),
					JobGetter:        mgr.GetClient(),
				},
			},
			Zone:              rt.Config().Multizone.Zone.Name,
			ResourceConverter: converter,
		},
		ResourceConverter: converter,
		SystemNamespace:   rt.Config().Store.Kubernetes.SystemNamespace,
	}
	return reconciler.SetupWithManager(mgr, rt.Config().Runtime.Kubernetes.ControllersConcurrency.PodController)
}

func addValidators(mgr kube_ctrl.Manager, rt core_runtime.Runtime, converter k8s_common.Converter) error {
	composite, ok := k8s_extensions.FromCompositeValidatorContext(rt.Extensions())
	if !ok {
		return errors.Errorf("could not find composite validator in the extensions context")
	}
	handler := k8s_webhooks.NewValidatingWebhook(converter, core_registry.Global(), k8s_registry.Global(), rt.Config().Mode, rt.Config().IsFederatedZoneCP(), rt.Config().Multizone.Zone.DisableOriginLabelValidation)
	composite.AddValidator(handler)

	coreZoneValidator := zone.Validator{Store: rt.ResourceManager()}
	k8sZoneValidator := k8s_webhooks.NewZoneValidatorWebhook(coreZoneValidator, rt.Config().Store.UnsafeDelete)
	composite.AddValidator(k8sZoneValidator)

	composite.AddValidator(&k8s_webhooks.PolicyNamespaceValidator{
		SystemNamespace: rt.Config().Store.Kubernetes.SystemNamespace,
	})

	composite.AddValidator(&k8s_webhooks.ContainerPatchValidator{
		SystemNamespace: rt.Config().Store.Kubernetes.SystemNamespace,
	})

	mgr.GetWebhookServer().Register("/validate-dubbo-io-v1alpha1", composite.IntoWebhook(mgr.GetScheme()))

	return nil
}

func addMutators(mgr kube_ctrl.Manager, rt core_runtime.Runtime, converter k8s_common.Converter) error {
	ownerRefMutator := &k8s_webhooks.OwnerReferenceMutator{
		Client:       mgr.GetClient(),
		CoreRegistry: core_registry.Global(),
		K8sRegistry:  k8s_registry.Global(),
		Scheme:       mgr.GetScheme(),
		Decoder:      kube_admission.NewDecoder(mgr.GetScheme()),
	}
	mgr.GetWebhookServer().Register("/owner-reference-dubbo-io-v1alpha1", &kube_webhook.Admission{Handler: ownerRefMutator})

	defaultMutator := k8s_webhooks.DefaultingWebhookFor(mgr.GetScheme(), converter)
	mgr.GetWebhookServer().Register("/default-dubbo-io-v1alpha1-mesh", defaultMutator)

	podEnvInjectMutator := k8s_webhooks.PodMutatingWebhook(
		func(ctx context.Context, pod *kube_core.Pod) error {
			RegistryCenterUrl := rt.RegistryCenter().GetURL()
			if pod.Labels["dubbo.io/name"] != "" {
				pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, kube_core.EnvVar{
					Name:  "DUBBO_REGISTRY_ADDRESS",
					Value: RegistryCenterUrl.String(),
				})
				//TODO support more
			}
			return nil
		})
	mgr.GetWebhookServer().Register("/pod-env-injection-dubbo-io-v1alpha1", podEnvInjectMutator)

	return nil
}
