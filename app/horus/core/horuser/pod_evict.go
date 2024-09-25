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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

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
