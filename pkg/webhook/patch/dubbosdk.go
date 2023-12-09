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

package patch

import (
	"fmt"
	"strconv"

	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-kubernetes/pkg/core/client/webhook"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type DubboSdk struct {
	options       *dubbo_cp.Config
	webhookClient webhook.Client
	kubeClient    kubernetes.Interface
}

func NewDubboSdk(options *dubbo_cp.Config, webhookClient webhook.Client, kubeClient kubernetes.Interface) *DubboSdk {
	return &DubboSdk{
		options:       options,
		webhookClient: webhookClient,
		kubeClient:    kubeClient,
	}
}

const (
	ExpireSeconds           = 1800
	Labeled                 = "true"
	EnvDubboRegistryAddress = "DUBBO_REGISTRY_ADDRESS"
)

const (
	RegistryInjectZookeeperLabel = "registry-zookeeper-inject"
	RegistryInjectNacosLabel     = "registry-nacos-inject"
	RegistryInjectK8sLabel       = "registry-k8s-inject"

	DefaultK8sRegistryAddress = "kubernetes://DEFAULT_MASTER_HOST"
)

// the priority of registry
// default is zk > nacos
var (
	registryInjectLabelPriorities = []string{
		RegistryInjectZookeeperLabel,
		RegistryInjectNacosLabel,
		RegistryInjectK8sLabel,
	}
	registrySchemas = map[string]string{
		RegistryInjectZookeeperLabel: "zookeeper",
		RegistryInjectNacosLabel:     "nacos",
	}
)

func (s *DubboSdk) injectAnnotations(target *v1.Pod, annotations map[string]string) {
	if target.Annotations == nil {
		target.Annotations = make(map[string]string)
	}

	for k, v := range annotations {
		if _, ok := target.Annotations[k]; !ok {
			target.Annotations[k] = v
		}
	}
}

func (s *DubboSdk) NewPodWithDubboRegistryInject(origin *v1.Pod) (*v1.Pod, error) {
	target := origin.DeepCopy()

	// find specific registry inject label (such as zookeeper-registry-inject)
	// in pod labels and namespace labels
	var registryInjects []string
	// 1. find in pod labels
	for _, registryInject := range registryInjectLabelPriorities {
		if target.Labels[registryInject] == Labeled { // find in pod labels
			registryInjects = []string{registryInject}
			break
		}
	}
	// 2. find in namespace labels
	if len(registryInjects) == 0 {
		for _, registryInject := range registryInjectLabelPriorities {
			if s.webhookClient.GetNamespaceLabels(target.Namespace)[registryInject] == Labeled {
				// find in namespace labels
				registryInjects = []string{registryInject}
				break
			}
		}
	}

	// default is zk > nacos > k8s
	if len(registryInjects) == 0 {
		registryInjects = registryInjectLabelPriorities
	}

	// find registry service in k8s
	var registryAddress string
	for _, registryInject := range registryInjects {
		if registryInject == RegistryInjectK8sLabel { // k8s registry
			registryAddress = DefaultK8sRegistryAddress
			continue
		}

		// other registry
		serviceList := s.webhookClient.ListServices(target.Namespace, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", registryInject, Labeled),
		})

		if serviceList == nil || len(serviceList.Items) < 1 {
			continue
		}

		schema := registrySchemas[registryInject]
		registryAddress = fmt.Sprintf("%s://%s.%s.svc", schema, serviceList.Items[0].Name, serviceList.Items[0].Namespace)
		break
	}

	var found bool
	if len(registryAddress) > 0 {
		// inject into env
		var targetContainers []v1.Container
		for _, c := range target.Spec.Containers {
			if !found { // found DUBBO_REGISTRY_ADDRESS ENV, stop inject
				found = s.injectEnv(&c, EnvDubboRegistryAddress, registryAddress)
			}

			targetContainers = append(targetContainers, c)
		}
		target.Spec.Containers = targetContainers
	}

	return target, nil
}

func (s *DubboSdk) injectEnv(container *v1.Container, name, value string) (found bool) {
	for j, env := range container.Env {
		if env.Name == name {
			found = true
			// env is not empty, inject into env
			if len(env.Value) > 0 {
				break
			}

			container.Env[j].Value = value
			break
		}
	}
	if found { // found registry env in pod, stop inject
		return
	}

	container.Env = append(container.Env, v1.EnvVar{
		Name:  name,
		Value: value,
	})

	return
}

func (s *DubboSdk) NewPodWithDubboCa(origin *v1.Pod) (*v1.Pod, error) {
	target := origin.DeepCopy()
	expireSeconds := int64(ExpireSeconds)

	shouldInject := false

	if target.Labels["dubbo-ca.inject"] == Labeled {
		shouldInject = true
	}

	if !shouldInject && s.webhookClient.GetNamespaceLabels(target.Namespace)["dubbo-ca.inject"] == Labeled {
		shouldInject = true
	}

	if shouldInject {
		shouldInject = s.checkVolume(target, shouldInject)

		for _, c := range target.Spec.Containers {
			shouldInject = s.checkContainers(c, shouldInject)
		}
	}

	if shouldInject {
		s.injectVolumes(target, expireSeconds)

		var targetContainers []v1.Container
		for _, c := range target.Spec.Containers {
			s.injectContainers(&c)

			targetContainers = append(targetContainers, c)
		}
		target.Spec.Containers = targetContainers
	}

	return target, nil
}

func (s *DubboSdk) injectContainers(c *v1.Container) {
	c.Env = append(c.Env, v1.EnvVar{
		Name:  "DUBBO_CA_ADDRESS",
		Value: s.options.KubeConfig.ServiceName + "." + s.options.KubeConfig.Namespace + ".svc:" + strconv.Itoa(s.options.GrpcServer.SecureServerPort),
	})
	c.Env = append(c.Env, v1.EnvVar{
		Name:  "DUBBO_CA_CERT_PATH",
		Value: "/var/run/secrets/dubbo-ca-cert/ca.crt",
	})
	c.Env = append(c.Env, v1.EnvVar{
		Name:  "DUBBO_OIDC_TOKEN",
		Value: "/var/run/secrets/dubbo-ca-token/token",
	})
	c.Env = append(c.Env, v1.EnvVar{
		Name:  "DUBBO_OIDC_TOKEN_TYPE",
		Value: "dubbo-ca-token",
	})

	c.VolumeMounts = append(c.VolumeMounts, v1.VolumeMount{
		Name:      "dubbo-ca-token",
		MountPath: "/var/run/secrets/dubbo-ca-token",
		ReadOnly:  true,
	})
	c.VolumeMounts = append(c.VolumeMounts, v1.VolumeMount{
		Name:      "dubbo-ca-cert",
		MountPath: "/var/run/secrets/dubbo-ca-cert",
		ReadOnly:  true,
	})
}

func (s *DubboSdk) injectVolumes(target *v1.Pod, expireSeconds int64) {
	target.Spec.Volumes = append(target.Spec.Volumes, v1.Volume{
		Name: "dubbo-ca-token",
		VolumeSource: v1.VolumeSource{
			Projected: &v1.ProjectedVolumeSource{
				Sources: []v1.VolumeProjection{
					{
						ServiceAccountToken: &v1.ServiceAccountTokenProjection{
							Audience:          "dubbo-ca",
							ExpirationSeconds: &expireSeconds,
							Path:              "token",
						},
					},
				},
			},
		},
	})
	target.Spec.Volumes = append(target.Spec.Volumes, v1.Volume{
		Name: "dubbo-ca-cert",
		VolumeSource: v1.VolumeSource{
			Projected: &v1.ProjectedVolumeSource{
				Sources: []v1.VolumeProjection{
					{
						ConfigMap: &v1.ConfigMapProjection{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "dubbo-ca-cert",
							},
							Items: []v1.KeyToPath{
								{
									Key:  "ca.crt",
									Path: "ca.crt",
								},
							},
						},
					},
				},
			},
		},
	})
}

func (s *DubboSdk) checkContainers(c v1.Container, shouldInject bool) bool {
	for _, e := range c.Env {
		if e.Name == "DUBBO_CA_ADDRESS" {
			shouldInject = false
			break
		}
		if e.Name == "DUBBO_CA_CERT_PATH" {
			shouldInject = false
			break
		}
		if e.Name == "DUBBO_OIDC_TOKEN" {
			shouldInject = false
			break
		}
		if e.Name == "DUBBO_OIDC_TOKEN_TYPE" {
			shouldInject = false
			break
		}
	}

	for _, m := range c.VolumeMounts {
		if m.Name == "dubbo-ca-token" {
			shouldInject = false
			break
		}
		if m.Name == "dubbo-ca-cert" {
			shouldInject = false
			break
		}
	}
	return shouldInject
}

func (s *DubboSdk) checkVolume(target *v1.Pod, shouldInject bool) bool {
	for _, v := range target.Spec.Volumes {
		if v.Name == "dubbo-ca-token" {
			shouldInject = false
			break
		}
	}
	for _, v := range target.Spec.Volumes {
		if v.Name == "dubbo-ca-cert" {
			shouldInject = false
			break
		}
	}
	return shouldInject
}
