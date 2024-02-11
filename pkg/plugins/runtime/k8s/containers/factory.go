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

package containers

import (
	runtime_k8s "github.com/apache/dubbo-kubernetes/pkg/config/plugins/runtime/k8s"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/runtime/k8s/metadata"
	kube_core "k8s.io/api/core/v1"
	kube_intstr "k8s.io/apimachinery/pkg/util/intstr"
	kube_client "sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
	"time"
)

type EnvVarsByName []kube_core.EnvVar

func (a EnvVarsByName) Len() int      { return len(a) }
func (a EnvVarsByName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a EnvVarsByName) Less(i, j int) bool {
	return a[i].Name < a[j].Name
}

type DataplaneProxyFactory struct {
	ControlPlaneURL    string
	ControlPlaneCACert string
	DefaultAdminPort   uint32
	ContainerConfig    runtime_k8s.DataplaneContainer
}

func NewDataplaneProxyFactory(
	controlPlaneURL string,
	controlPlaneCACert string,
	defaultAdminPort uint32,
	containerConfig runtime_k8s.DataplaneContainer,
	waitForDataplane bool,
) *DataplaneProxyFactory {
	return &DataplaneProxyFactory{
		ControlPlaneURL:    controlPlaneURL,
		ControlPlaneCACert: controlPlaneCACert,
		DefaultAdminPort:   defaultAdminPort,
		ContainerConfig:    containerConfig,
	}
}

func (i *DataplaneProxyFactory) envoyAdminPort(annotations map[string]string) (uint32, error) {
	adminPort, _, err := metadata.Annotations(annotations).GetUint32(metadata.DubboIngressAnnotation)
	return adminPort, err
}

func (i *DataplaneProxyFactory) drainTime(annotations map[string]string) (time.Duration, error) {
	r, _, err := metadata.Annotations(annotations).GetDurationWithDefault(i.ContainerConfig.DrainTime.Duration, metadata.DubboSidecarDrainTime)
	return r, err
}

func (i *DataplaneProxyFactory) sidecarEnvVars(mesh string, podAnnotations map[string]string) ([]kube_core.EnvVar, error) {
	drainTime, err := i.drainTime(podAnnotations)
	if err != nil {
		return nil, err
	}

	envVars := map[string]kube_core.EnvVar{
		"POD_NAME": {
			Name: "POD_NAME",
			ValueFrom: &kube_core.EnvVarSource{
				FieldRef: &kube_core.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.name",
				},
			},
		},
		"POD_NAMESPACE": {
			Name: "POD_NAMESPACE",
			ValueFrom: &kube_core.EnvVarSource{
				FieldRef: &kube_core.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.namespace",
				},
			},
		},
		"INSTANCE_IP": {
			Name: "INSTANCE_IP",
			ValueFrom: &kube_core.EnvVarSource{
				FieldRef: &kube_core.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "status.podIP",
				},
			},
		},
		"DUBBO_CONTROL_PLANE_URL": {
			Name:  "DUBBO_CONTROL_PLANE_URL",
			Value: i.ControlPlaneURL,
		},
		"DUBBO_DATAPLANE_MESH": {
			Name:  "DUBBO_DATAPLANE_MESH",
			Value: mesh,
		},
		"DUBBO_DATAPLANE_DRAIN_TIME": {
			Name:  "DUBBO_DATAPLANE_DRAIN_TIME",
			Value: drainTime.String(),
		},
		"DUBBO_DATAPLANE_RUNTIME_TOKEN_PATH": {
			Name:  "DUBBO_DATAPLANE_RUNTIME_TOKEN_PATH",
			Value: "/var/run/secrets/kubernetes.io/serviceaccount/token",
		},
		"DUBBO_CONTROL_PLANE_CA_CERT": {
			Name:  "DUBBO_CONTROL_PLANE_CA_CERT",
			Value: i.ControlPlaneCACert,
		},
	}

	// override defaults with cfg env vars
	for envName, envVal := range i.ContainerConfig.EnvVars {
		envVars[envName] = kube_core.EnvVar{
			Name:  envName,
			Value: envVal,
		}
	}

	// override defaults and cfg env vars with annotations
	annotationEnvVars, _, err := metadata.Annotations(podAnnotations).GetMap(metadata.DUBBOSidecarEnvVarsAnnotation)
	if err != nil {
		return nil, err
	}
	for envName, envVal := range annotationEnvVars {
		envVars[envName] = kube_core.EnvVar{
			Name:  envName,
			Value: envVal,
		}
	}

	var result []kube_core.EnvVar
	for _, v := range envVars {
		result = append(result, v)
	}
	sort.Stable(EnvVarsByName(result))

	return result, nil
}

func (i *DataplaneProxyFactory) NewContainer(
	owner kube_client.Object,
	mesh string,
) (kube_core.Container, error) {
	annnotations := owner.GetAnnotations()

	env, err := i.sidecarEnvVars(mesh, annnotations)
	if err != nil {
		return kube_core.Container{}, err
	}

	adminPort, err := i.envoyAdminPort(annnotations)
	if err != nil {
		return kube_core.Container{}, err
	}
	if adminPort == 0 {
		adminPort = i.DefaultAdminPort
	}

	container := kube_core.Container{
		Env: env,
		LivenessProbe: &kube_core.Probe{
			ProbeHandler: kube_core.ProbeHandler{
				HTTPGet: &kube_core.HTTPGetAction{
					Path: "/ready",
					Port: kube_intstr.IntOrString{
						IntVal: int32(adminPort),
					},
				},
			},
			InitialDelaySeconds: i.ContainerConfig.LivenessProbe.InitialDelaySeconds,
			TimeoutSeconds:      i.ContainerConfig.LivenessProbe.TimeoutSeconds,
			PeriodSeconds:       i.ContainerConfig.LivenessProbe.PeriodSeconds,
			SuccessThreshold:    1,
			FailureThreshold:    i.ContainerConfig.LivenessProbe.FailureThreshold,
		},
		ReadinessProbe: &kube_core.Probe{
			ProbeHandler: kube_core.ProbeHandler{
				HTTPGet: &kube_core.HTTPGetAction{
					Path: "/ready",
					Port: kube_intstr.IntOrString{
						IntVal: int32(adminPort),
					},
				},
			},
			InitialDelaySeconds: i.ContainerConfig.ReadinessProbe.InitialDelaySeconds,
			TimeoutSeconds:      i.ContainerConfig.ReadinessProbe.TimeoutSeconds,
			PeriodSeconds:       i.ContainerConfig.ReadinessProbe.PeriodSeconds,
			SuccessThreshold:    i.ContainerConfig.ReadinessProbe.SuccessThreshold,
			FailureThreshold:    i.ContainerConfig.ReadinessProbe.FailureThreshold,
		},
	}
	return container, nil
}
