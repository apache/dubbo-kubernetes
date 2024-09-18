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

func (h *Horuser) Cordon(nodeName, clusterName, moduleName string) (err error) {
	kubeClient := h.kubeClientMap[clusterName]
	if kubeClient == nil {
		klog.Errorf("node Cordon kubeClient by clusterName empty.")
		klog.Infof("nodeName:%v,clusterName:%v", nodeName, clusterName)
		return err
	}

	ctxFirst, cancelFirst := h.GetK8sContext()
	defer cancelFirst()
	node, err := kubeClient.CoreV1().Nodes().Get(ctxFirst, nodeName, v1.GetOptions{})
	if err != nil {
		klog.Errorf("node Cordon get err nodeName:%v clusterName:%v", nodeName, clusterName)
		return err
	}
	annotations := node.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["dubbo.apache.org/disable-by"] = "horus"

	node.Spec.Unschedulable = true
	ctxSecond, cancelSecond := h.GetK8sContext()
	defer cancelSecond()
	node, err = kubeClient.CoreV1().Nodes().Update(ctxSecond, node, v1.UpdateOptions{})
	if err != nil {
		klog.Errorf("node Cordon update err nodeName:%v clusterName:%v", nodeName, clusterName)
		return err
	}
	klog.Infof("node Cordon success nodeName:%v clusterName:%v", nodeName, clusterName)
	return nil
}

func (h *Horuser) UnCordon(nodeName, clusterName string) (err error) {
	kubeClient := h.kubeClientMap[clusterName]
	if kubeClient == nil {
		klog.Errorf("node UnCordon kubeClient by clusterName empty.")
		klog.Infof("nodeName:%v,clusterName:%v", nodeName, clusterName)
		return err
	}

	ctxFirst, cancelFirst := h.GetK8sContext()
	defer cancelFirst()
	node, err := kubeClient.CoreV1().Nodes().Get(ctxFirst, nodeName, v1.GetOptions{})
	if err != nil {
		klog.Errorf("node UnCordon get err nodeName:%v clusterName:%v", nodeName, clusterName)
		return err
	}

	node.Spec.Unschedulable = false

	ctxSecond, cancelSecond := h.GetK8sContext()
	defer cancelSecond()
	node, err = kubeClient.CoreV1().Nodes().Update(ctxSecond, node, v1.UpdateOptions{})
	if err != nil {
		klog.Errorf("node UnCordon update err nodeName:%v clusterName:%v", nodeName, clusterName)
		return err
	}
	klog.Infof("node UnCordon success nodeName:%v clusterName:%v", nodeName, clusterName)
	return nil
}

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
			if owner.Kind == "Daemonset" {
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

func (h *Horuser) Evict(podName, podNamespace, clusterName string) (err error) {
	kubeClient := h.kubeClientMap[clusterName]
	if kubeClient == nil {
		klog.Errorf("pod Evict kubeClient by clusterName empty.")
		klog.Infof("podName:%v podNamespace:%v clusterName:%v", podName, podNamespace, clusterName)
		return err
	}

	ctxFirst, cancelFirst := h.GetK8sContext()
	defer cancelFirst()
	_, err = kubeClient.CoreV1().Pods(podNamespace).Get(ctxFirst, podName, v1.GetOptions{})
	if err != nil {
		klog.Errorf("pod Evict get err clusterName:%v podName:%v podNamespace:%v", clusterName, podName, podNamespace)
		return err
	}
	ctxSecond, cancelSecond := h.GetK8sContext()
	defer cancelSecond()
	var gracePeriodSeconds int64 = -1
	propagationPolicy := v1.DeletePropagationBackground
	err = kubeClient.CoreV1().Pods(podNamespace).Delete(ctxSecond, podName, v1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
		PropagationPolicy:  &propagationPolicy,
	})
	if err != nil {
		klog.Errorf("pod Evict delete err clusterName:%v podName:%v podNamespace:%v", clusterName, podName, podNamespace)
		return err
	}
	klog.Infof("pod Evict delete success clusterName:%v podName:%v podNamespace:%v", clusterName, podName, podNamespace)
	return nil
}
