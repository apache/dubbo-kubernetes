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
	"fmt"
	"github.com/apache/dubbo-kubernetes/app/horus/base/db"
	"github.com/apache/dubbo-kubernetes/app/horus/core/alerter"
	"github.com/gammazero/workerpool"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sync"
	"time"
)

const (
	ModuleName = "podStagnationCleaner"
	Reason     = "StagnationCleanup"
)

func (h *Horuser) PodStagnationCleanManager(ctx context.Context) error {
	go wait.UntilWithContext(ctx, h.PodStagnationClean, time.Duration(h.cc.PodStagnationCleaner.IntervalSecond)*time.Second)
	<-ctx.Done()
	return nil
}

func (h *Horuser) PodStagnationClean(ctx context.Context) {
	var wg sync.WaitGroup
	for cn := range h.cc.PodStagnationCleaner.KubeMultiple {
		cn := cn
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.PodsOnCluster(cn)
		}()
	}
	wg.Wait()
}

func (h *Horuser) PodsOnCluster(clusterName string) {
	pods, err := h.Retrieve(clusterName, h.cc.PodStagnationCleaner.FieldSelector)
	if err != nil {
		klog.Errorf("Failed to retrieve pods on err:%v", err)
		klog.Infof("clusterName:%v\n", clusterName)
		return
	}
	count := len(pods)
	if count == 0 {
		klog.Infof("PodsOnCluster no abnomal clusterName:%v\n", clusterName)
		return
	}
	wp := workerpool.New(10)
	for index, pod := range pods {
		pod := pod
		if pod.Status.Phase == corev1.PodRunning {
			continue
		}
		msg := fmt.Sprintf("\n【集群：%v】\n【停滞：%d/%d】\n【PodName:%v】\n【Namespace:%v】\n【Phase:%v】\n【节点：%v】\n", clusterName, index+1, count, pod.Name, pod.Namespace, pod.Status.Phase, pod.Spec.NodeName)
		klog.Infof(msg)

		wp.Submit(func() {
			h.PodSingle(pod, clusterName)
		})
	}
	wp.StopWait()
}

func (h *Horuser) PodSingle(pod corev1.Pod, clusterName string) {
	var err error
	if !pod.DeletionTimestamp.IsZero() {
		if len(pod.Finalizers) > 0 {
			time.Sleep(time.Duration(h.cc.PodStagnationCleaner.DoubleSecond) * time.Second)
			if !h.Terminating(clusterName, &pod) {
				klog.Infof("Pod %s is still terminating skipping.", pod.Name)
				return
			}
			err := h.Finalizer(clusterName, pod.Name, pod.Namespace)
			if err != nil {
				klog.Errorf("Failed to patch finalizer for pod %s: %v", pod.Name, err)
				return
			}
			klog.Infof("Successfully patched finalizer for pod %s", pod.Name)
		}
		return
	}

	if len(pod.Finalizers) == 0 && pod.Name != "" {
		err := h.Evict(pod.Name, pod.Namespace, clusterName)
		if err != nil {
			klog.Errorf("Failed to evict pod %s err:%v", pod.Name, err)
			return
		}
		klog.Infof("Evicted pod %s successfully", pod.Name)
	}
	res := "Success"
	if err != nil {
		res = fmt.Sprintf("result failed:%v", err)
	}
	today := time.Now().Format("2006-01-02")
	msg := fmt.Sprintf("\n【集群：%v】\n【Pod：%v】\n【Namespace：%v】\n【清除 finalizer:%v】\n", clusterName, pod.Name, pod.Namespace, res)
	alerter.DingTalkSend(h.cc.PodStagnationCleaner.DingTalk, msg)
	write := db.PodDataInfo{
		PodName:     pod.Name,
		PodIP:       pod.Status.PodIP,
		NodeName:    pod.Spec.NodeName,
		ClusterName: clusterName,
		ModuleName:  ModuleName,
		Reason:      Reason,
		FirstDate:   today,
	}
	_, err = write.AddOrGet()
	klog.Errorf("write AddOrGet err:%v", err)
	return
}
