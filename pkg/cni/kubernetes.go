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

package cni

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubernetesPodInfoProvider struct {
	client kubernetes.Interface
}

func NewKubernetesPodInfoProvider(kubeconfig string) (*KubernetesPodInfoProvider, error) {
	var (
		cfg *rest.Config
		err error
	)
	if kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("build Kubernetes client config: %w", err)
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client: %w", err)
	}
	return &KubernetesPodInfoProvider{client: client}, nil
}

func (p *KubernetesPodInfoProvider) PodInfo(ctx context.Context, ref PodRef) (PodInfo, error) {
	pod, err := p.client.CoreV1().Pods(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		return PodInfo{}, fmt.Errorf("get pod %s/%s: %w", ref.Namespace, ref.Name, err)
	}
	ips := make([]string, 0, len(pod.Status.PodIPs))
	for _, item := range pod.Status.PodIPs {
		if item.IP != "" {
			ips = append(ips, item.IP)
		}
	}
	if pod.Status.PodIP != "" && len(ips) == 0 {
		ips = append(ips, pod.Status.PodIP)
	}
	return PodInfo{
		Namespace: pod.Namespace,
		Name:      pod.Name,
		Labels:    pod.Labels,
		IPs:       ips,
	}, nil
}
