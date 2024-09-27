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
	"github.com/apache/dubbo-kubernetes/app/horus/basic/db"
	"github.com/apache/dubbo-kubernetes/app/horus/core/alert"
	"github.com/gammazero/workerpool"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sync"
	"time"
)

const (
	ModuleName = "pod_abnormal_clean"
)

func (h *Horuser) PodAbnormalCleanManager(ctx context.Context) error {
	go wait.UntilWithContext(ctx, h.PodAbnormalClean, time.Duration(h.cc.PodAbnormal.IntervalSecond)*time.Second)
	<-ctx.Done()
	return nil
}

func (h *Horuser) PodAbnormalClean(ctx context.Context) {
	var wg sync.WaitGroup
	for cn := range h.cc.PodAbnormal.KubeMultiple {
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
	var podNamespace string
	pods, err := h.Fetch(clusterName, podNamespace, h.cc.PodAbnormal.FieldSelector)
	if err != nil {
		klog.Errorf("Failed to fetch pods on cluster:%v", err)
		klog.Infof("clusterName:%v podNamespace:%v", clusterName, podNamespace)
		return
	}
	count := len(pods)
	if count == 0 {
		klog.Infof("PodsOnCluster no abnomal clusterName:%v", clusterName)
		return
	}
	wp := workerpool.New(10)
	for index, pod := range pods {
		if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}
		msg := fmt.Sprintf("\n【集群：%v】\n【存活：%d/%d】\n【PodName:%v】\n【Namespace:%v】\n【Phase:%v】\n【节点：%v】\n", clusterName, index+1, count, pod.Name, pod.Namespace, pod.Status.Phase, pod.Spec.NodeName)
		klog.Infof(msg)

		wp.Submit(func() {
			h.PodSingle(pod, clusterName)
		})
	}
	wp.StopWait()
}

func (h *Horuser) PodSingle(pod corev1.Pod, clusterName string) {
	if !pod.DeletionTimestamp.IsZero() {
		var err error
		action := ""
		switch len(pod.Finalizers) {
		case 0:
			if pod.Name != "" {
				return
			}
			err = h.Evict(pod.Name, pod.Namespace, clusterName)
			action = "try patch-finalizer"
		default:
			time.Sleep(time.Duration(h.cc.PodAbnormal.DoubleSecond) * time.Second)
			pass := h.Terminating(clusterName, &pod)
			if !pass {
				return
			}
			err = h.Finalizer(clusterName, pod.Name, pod.Namespace)
			action = "try patch-finalizer"
			res := "Success"
			if err != nil {
				res = fmt.Sprintf("failed:%v", err)
			}
			today := time.Now().Format("2006-01-02")
			msg := fmt.Sprintf("\n【集群：%v】\n【Pod：%v】\n【Namespace：%v】\n【无法删除的 finalizer:%v】\n【处理结果：%v】\n", clusterName, pod.Name, pod.Namespace, err, res)
			alert.DingTalkSend(h.cc.PodAbnormal.DingTalk, msg)
			write := db.PodDataInfo{
				PodName:     pod.Name,
				PodIP:       pod.Status.PodIP,
				NodeName:    pod.Spec.NodeName,
				ClusterName: clusterName,
				ModuleName:  ModuleName,
				Reason:      action,
				FirstDate:   today,
			}
			_, err = write.AddOrGet()
			klog.Errorf("write AddOrGet err:%v", err)
			klog.Infof("podName:%v", pod.Name)
			return
		}
	}
}
