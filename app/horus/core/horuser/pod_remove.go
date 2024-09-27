// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package horuser

import (
	"context"
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

type RemoveJsonValue struct {
	Op   string `json:"op"`
	Path string `json:"path"`
}

func (h *Horuser) Finalizer(clusterName, podName, podNamespace string) error {
	kubeClient := h.kubeClientMap[clusterName]
	if kubeClient == nil {
		klog.Errorf("Finalizer kubeClient by clusterName empty.")
		klog.Infof("clusterName:%v podName:%v", clusterName, podName)
		return nil
	}
	finalizer := RemoveJsonValue{
		Op:   "remove",
		Path: "/metadata/finalizers",
	}
	var payload []interface{}
	payload = append(payload, finalizer)
	data, _ := json.Marshal(payload)
	ctx, cancel := h.GetK8sContext()
	defer cancel()
	_, err := kubeClient.CoreV1().Pods(podNamespace).Patch(ctx, podName, types.JSONPatchType, data, v1.PatchOptions{})
	return err
}

func (h *Horuser) Terminating(clusterName string, oldPod *corev1.Pod) bool {
	kubeClient := h.kubeClientMap[clusterName]
	if kubeClient == nil {
		return false
	}
	newPod, _ := kubeClient.CoreV1().Pods(oldPod.Namespace).Get(context.Background(), oldPod.Name, v1.GetOptions{})
	if newPod == nil {
		return false
	}
	if newPod.UID != oldPod.UID {
		return false
	}
	if newPod.DeletionTimestamp.IsZero() {
		return false
	}
	return true
}

func (h *Horuser) Fetch(clusterName, podNamespace, fieldSelector string) ([]corev1.Pod, error) {
	kubeClient := h.kubeClientMap[clusterName]
	if kubeClient == nil {
		klog.Errorf("Fetch kubeClient by clusterName empty.")
		klog.Infof("clusterName:%v", clusterName)
		return nil, nil
	}
	ctx, cancel := h.GetK8sContext()
	defer cancel()
	list := v1.ListOptions{FieldSelector: fieldSelector}
	pods, err := kubeClient.CoreV1().Pods(podNamespace).List(ctx, list)
	if err != nil {
		klog.Errorf("Fetch list pod err:%v", err)
		klog.Infof("clusterName:%v fieldSelector:%v", clusterName, fieldSelector)
	}
	return pods.Items, err
}
