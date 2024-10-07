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

func (h *Horuser) UnCordon(nodeName, clusterName string) (err error) {
	kubeClient := h.kubeClientMap[clusterName]
	if kubeClient == nil {
		klog.Error("node UnCordon kubeClient by clusterName empty.")
		klog.Infof("nodeName:%v\n,clusterName:%v\n", nodeName, clusterName)
		return err
	}

	ctxFirst, cancelFirst := h.GetK8sContext()
	defer cancelFirst()
	node, err := kubeClient.CoreV1().Nodes().Get(ctxFirst, nodeName, v1.GetOptions{})
	if err != nil {
		klog.Errorf("node UnCordon get err:%v", err)
		klog.Infof("nodeName:%v\n clusterName:%v\n", nodeName, clusterName)
		return err
	}

	node.Spec.Unschedulable = false

	ctxSecond, cancelSecond := h.GetK8sContext()
	defer cancelSecond()
	node, err = kubeClient.CoreV1().Nodes().Update(ctxSecond, node, v1.UpdateOptions{})
	if err != nil {
		klog.Errorf("node UnCordon update err:%v", err)
		klog.Infof("nodeName:%v\n clusterName:%v\n", nodeName, clusterName)
		return err
	}
	klog.Info("node UnCordon success.")
	klog.Infof("nodeName:%v\n clusterName:%v\n", nodeName, clusterName)
	return nil
}
