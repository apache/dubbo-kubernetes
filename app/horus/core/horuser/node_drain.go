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
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func (h *Horuser) Drain(nodeName, clusterName string) (err error) {
	kubeClient := h.kubeClientMap[clusterName]
	if kubeClient == nil {
		klog.Errorf("node Drain kubeClient by clusterName empty.")
		klog.Infof("nodeName:%v,clusterName:%v", nodeName, clusterName)
		return err
	}

	ctxFirst, cancelFirst := h.GetK8sContext()
	defer cancelFirst()
	listOpts := v1.ListOptions{FieldSelector: fmt.Sprintf("nodeName=%s", nodeName)}
	var podNamespace string
	pod, err := kubeClient.CoreV1().Pods(podNamespace).List(ctxFirst, listOpts)
	if err != nil {
		klog.Errorf("node Drain err:%v nodeName:%v clusterName:%v", err, nodeName, clusterName)
		return err
	}
	if len(pod.Items) == 0 {
		klog.Errorf("Cannot find pod on node.")
		klog.Infof("nodeName:%v,clusterName:%v", nodeName, clusterName)
	}
	count := len(pod.Items)
	for items, pods := range pod.Items {
		ds := false
		for _, owner := range pods.OwnerReferences {
			if owner.Kind == "DaemonSet" {
				ds = true
				break
			}
		}
		klog.Errorf("node Drain evict pod result items:%d count:%v nodeName:%v clusterName:%v podName:%v podNamespace:%v", items+1, count, nodeName, clusterName, pods.Name, pods.Namespace)
		if ds {
			continue
		}
		err := h.Evict(pods.Name, pods.Namespace, clusterName)
		if err != nil {
			klog.Errorf("node Drain evict pod err:%v items:%d count:%v nodeName:%v clusterName:%v podName:%v podNamespace:%v", err, items+1, count, nodeName, clusterName, pods.Name, pods.Namespace)
			return err
		}
	}
	return nil
}
