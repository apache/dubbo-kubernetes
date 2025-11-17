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

package kube

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/features"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
	"regexp"
	"strings"
)

const MaxRequestBodyBytes = int64(6 * 1024 * 1024)

var cronJobNameRegexp = regexp.MustCompile(`(.+)-\d{8,10}$`)

type Syncer interface {
	HasSynced() bool
}

func DefaultRestConfig(kubeconfig, configContext string, fns ...func(config *rest.Config)) (*rest.Config, error) {
	config, err := BuildClientConfig(kubeconfig, configContext)
	if err != nil {
		return nil, err
	}
	for _, fn := range fns {
		fn(config)
	}
	return config, nil
}

func BuildClientCmd(kubeconfig, context string, overrides ...func(configOverrides *clientcmd.ConfigOverrides)) clientcmd.ClientConfig {
	if kubeconfig != "" {
		info, err := os.Stat(kubeconfig)
		if err != nil || info.Size() == 0 {
			kubeconfig = ""
		}
	}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	loadingRules.ExplicitPath = kubeconfig
	configOverrides := &clientcmd.ConfigOverrides{
		ClusterDefaults: clientcmd.ClusterDefaults,
		CurrentContext:  context,
	}
	for _, fn := range overrides {
		fn(configOverrides)
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
}

func BuildClientConfig(kubeconfig, context string) (*rest.Config, error) {
	c, err := BuildClientCmd(kubeconfig, context).ClientConfig()
	if err != nil {
		return nil, err
	}
	return SetRestDefaults(c), nil
}

func SetRestDefaults(config *rest.Config) *rest.Config {
	if config.GroupVersion == nil || config.GroupVersion.Empty() {
		config.GroupVersion = &corev1.SchemeGroupVersion
	}
	if len(config.APIPath) == 0 {
		if len(config.GroupVersion.Group) == 0 {
			config.APIPath = "/api"
		} else {
			config.APIPath = "/apis"
		}
	}
	if len(config.ContentType) == 0 {
		if features.KubernetesClientContentType == "json" {
			config.ContentType = runtime.ContentTypeJSON
		} else {
			config.AcceptContentTypes = runtime.ContentTypeProtobuf + "," + runtime.ContentTypeJSON
			config.ContentType = runtime.ContentTypeJSON
		}
	}
	if config.NegotiatedSerializer == nil {
		config.NegotiatedSerializer = serializer.NewCodecFactory(nil).WithoutConversion()
	}
	return config
}

func AllSynced[T Syncer](syncers []T) bool {
	for _, h := range syncers {
		if !h.HasSynced() {
			return false
		}
	}
	return true
}

func HTTPConfigReader(req *http.Request) ([]byte, error) {
	defer req.Body.Close()
	lr := &io.LimitedReader{
		R: req.Body,
		N: MaxRequestBodyBytes + 1,
	}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if lr.N <= 0 {
		return nil, errors.NewRequestEntityTooLargeError(fmt.Sprintf("limit is %d", MaxRequestBodyBytes))
	}
	return data, nil
}

func StripPodUnusedFields(obj any) (any, error) {
	t, ok := obj.(metav1.ObjectMetaAccessor)
	if !ok {
		// shouldn't happen
		return obj, nil
	}
	// ManagedFields is large and we never use it
	t.GetObjectMeta().SetManagedFields(nil)
	// only container ports can be used
	if pod := obj.(*corev1.Pod); pod != nil {
		containers := []corev1.Container{}
		for _, c := range pod.Spec.Containers {
			if len(c.Ports) > 0 {
				containers = append(containers, corev1.Container{
					Ports: c.Ports,
				})
			}
		}
		oldSpec := pod.Spec
		newSpec := corev1.PodSpec{
			Containers:         containers,
			ServiceAccountName: oldSpec.ServiceAccountName,
			NodeName:           oldSpec.NodeName,
			HostNetwork:        oldSpec.HostNetwork,
			Hostname:           oldSpec.Hostname,
			Subdomain:          oldSpec.Subdomain,
		}
		pod.Spec = newSpec
		pod.Status.InitContainerStatuses = nil
		pod.Status.ContainerStatuses = nil
	}

	return obj, nil
}

func GetDeployMetaFromPod(pod *corev1.Pod) (types.NamespacedName, metav1.TypeMeta) {
	if pod == nil {
		return types.NamespacedName{}, metav1.TypeMeta{}
	}
	// try to capture more useful namespace/name info for deployments, etc.
	// TODO(dougreid): expand to enable lookup of OWNERs recursively a la kubernetesenv

	deployMeta := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}

	typeMetadata := metav1.TypeMeta{
		Kind:       "Pod",
		APIVersion: "v1",
	}
	if len(pod.GenerateName) > 0 {
		// if the pod name was generated (or is scheduled for generation), we can begin an investigation into the controlling reference for the pod.
		var controllerRef metav1.OwnerReference
		controllerFound := false
		for _, ref := range pod.GetOwnerReferences() {
			if ref.Controller != nil && *ref.Controller {
				controllerRef = ref
				controllerFound = true
				break
			}
		}
		if controllerFound {
			typeMetadata.APIVersion = controllerRef.APIVersion
			typeMetadata.Kind = controllerRef.Kind

			// heuristic for deployment detection
			deployMeta.Name = controllerRef.Name
			if typeMetadata.Kind == "ReplicaSet" && pod.Labels["pod-template-hash"] != "" && strings.HasSuffix(controllerRef.Name, pod.Labels["pod-template-hash"]) {
				name := strings.TrimSuffix(controllerRef.Name, "-"+pod.Labels["pod-template-hash"])
				deployMeta.Name = name
				typeMetadata.Kind = "Deployment"
			} else if typeMetadata.Kind == "ReplicaSet" && pod.Labels["rollouts-pod-template-hash"] != "" &&
				strings.HasSuffix(controllerRef.Name, pod.Labels["rollouts-pod-template-hash"]) {
				// Heuristic for ArgoCD Rollout
				name := strings.TrimSuffix(controllerRef.Name, "-"+pod.Labels["rollouts-pod-template-hash"])
				deployMeta.Name = name
				typeMetadata.Kind = "Rollout"
				typeMetadata.APIVersion = "v1alpha1"
			} else if typeMetadata.Kind == "ReplicationController" && pod.Labels["deploymentconfig"] != "" {
				// If the pod is controlled by the replication controller, which is created by the DeploymentConfig resource in
				// Openshift platform, set the deploy name to the deployment config's name, and the kind to 'DeploymentConfig'.
				//
				// nolint: lll
				// For DeploymentConfig details, refer to
				// https://docs.openshift.com/container-platform/4.1/applications/deployments/what-deployments-are.html#deployments-and-deploymentconfigs_what-deployments-are
				//
				// For the reference to the pod label 'deploymentconfig', refer to
				// https://github.com/openshift/library-go/blob/7a65fdb398e28782ee1650959a5e0419121e97ae/pkg/apps/appsutil/const.go#L25
				deployMeta.Name = pod.Labels["deploymentconfig"]
				typeMetadata.Kind = "DeploymentConfig"
			} else if typeMetadata.Kind == "Job" {
				// If job name suffixed with `-<digit-timestamp>`, where the length of digit timestamp is 8~10,
				// trim the suffix and set kind to cron job.
				if jn := cronJobNameRegexp.FindStringSubmatch(controllerRef.Name); len(jn) == 2 {
					deployMeta.Name = jn[1]
					typeMetadata.Kind = "CronJob"
					// heuristically set cron job api version to v1 as it cannot be derived from pod metadata.
					typeMetadata.APIVersion = "batch/v1"
				}
			}
		}
	}

	if deployMeta.Name == "" {
		// if we haven't been able to extract a deployment name, then just give it the pod name
		deployMeta.Name = pod.Name
	}

	return deployMeta, typeMetadata
}
