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
		klog.Error("node Drain kubeClient by clusterName empty.")
		klog.Infof("nodeName:%v\n,clusterName:%v\n", nodeName, clusterName)
		return err
	}

	ctxFirst, cancelFirst := h.GetK8sContext()
	defer cancelFirst()
	listOpts := v1.ListOptions{FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName)}
	pod, err := kubeClient.CoreV1().Pods("").List(ctxFirst, listOpts)
	if err != nil {
		klog.Errorf("node Drain err:%v", err)
		klog.Infof("nodeName:%v\n clusterName:%v\n", nodeName, clusterName)
		return err
	}
	if len(pod.Items) == 0 {
		klog.Error("Unable to find pod on node..")
		klog.Infof("nodeName:%v,clusterName:%v\n", nodeName, clusterName)
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
		if ds {
			continue
		}
		klog.Errorf("node Drain evict pod result items:%d count:%v nodeName:%v\n clusterName:%v\n podName:%v\n podNamespace:%v\n", items+1, count, nodeName, clusterName, pods.Name, pods.Namespace)

		err = h.Evict(pods.Name, pods.Namespace, clusterName)
		if err != nil {
			klog.Errorf("node Drain evict pod err:%v items:%d count:%v nodeName:%v\n clusterName:%v\n podName:%v\n podNamespace:%v\n", err, items+1, count, nodeName, clusterName, pods.Name, pods.Namespace)
			return err
		}
	}
	return nil
}
