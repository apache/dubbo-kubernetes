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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sync"
	"time"
)

const (
	NODE_DOWN        = "nodeDown"
	NODE_DOWN_REASON = "downtime"
)

func (h *Horuser) DownTimeManager(ctx context.Context) error {
	go wait.UntilWithContext(ctx, h.DownTimeCheck, time.Duration(h.cc.NodeDownTime.IntervalSecond)*time.Second)
	<-ctx.Done()
	return nil
}

func (h *Horuser) DownTimeCheck(ctx context.Context) {
	var wg sync.WaitGroup

	for clusterName, addr := range h.cc.PromMultiple {
		clusterName := clusterName
		if _, exist := h.cc.NodeDownTime.KubeMultiple[clusterName]; !exist {
			klog.Info("DownTimeCheck config disable.")
			klog.Infof("clusterName:%v\n", clusterName)
			continue
		}
		addr := addr
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.DownTimeNodes(clusterName, addr)
		}()
	}
	wg.Wait()
}

func (h *Horuser) DownTimeNodes(clusterName, addr string) {
	kubeClient := h.kubeClientMap[clusterName]
	if kubeClient == nil {
		klog.Errorf("DownTimeNodes kubeClient by clusterName empty.")
		return
	}
	ctxFirst, cancelFirst := h.GetK8sContext()
	defer cancelFirst()

	klog.Info("DownTimeNodes Query Start.")
	klog.Infof("clusterName:%v\n", clusterName)

	nodeDownTimeRes := make(map[string]int)
	cq := len(h.cc.NodeDownTime.AbnormalityQL)
	for _, ql := range h.cc.NodeDownTime.AbnormalityQL {
		ql := ql
		res, err := h.InstantQuery(addr, ql, clusterName, h.cc.NodeDownTime.PromQueryTimeSecond)
		if err != nil {
			klog.Errorf("downtimeNodes InstantQuery err:%v", err)
			klog.Infof("clusterName:%v\n", clusterName)
			continue
		}

		for _, v := range res {
			v := v
			nodeName := string(v.Metric["node"])
			if nodeName == "" {
				klog.Error("downtimeNodes InstantQuery nodeName empty.")
				klog.Infof("clusterName:%v\n metric:%v\n", clusterName, v.Metric)
				continue
			}
			nodeDownTimeRes[nodeName]++
		}
	}
	WithDownNodeIPs := make(map[string]string)

	for node, count := range nodeDownTimeRes {
		if count < cq {
			klog.Error("downtimeNodes node not reach threshold")
			klog.Infof("clusterName:%v\n nodeName:%v\n threshold:%v count:%v", clusterName, node, cq, count)
			continue
		}
		abnormalInfoSystemQL := fmt.Sprintf(h.cc.NodeDownTime.AbnormalInfoSystemQL, node)
		res, err := h.InstantQuery(addr, abnormalInfoSystemQL, clusterName, h.cc.NodeDownTime.PromQueryTimeSecond)
		if len(res) == 0 {
			klog.Errorf("no results returned for query:%s", abnormalInfoSystemQL)
			continue
		}
		if err != nil {
			klog.Errorf("downtimeNodes InstantQuery NodeName To IPs empty err:%v", err)
			klog.Infof("clusterName:%v\n AbnormalInfoSystemQL:%v, err:%v", clusterName, abnormalInfoSystemQL, err)
			continue
		}
		str := ""
		for _, v := range res {
			str = string(v.Metric["instance"])
		}
		WithDownNodeIPs[node] = str
	}

	msg := fmt.Sprintf("\n【%s】\n【集群：%v】\n【已达到宕机临界点：%v】", h.cc.NodeDownTime.DingTalk.Title, clusterName, len(WithDownNodeIPs))

	newfound := 0

	for nodeName, _ := range WithDownNodeIPs {
		firstDate := time.Now().Format("2006-01-02")
		err := h.Cordon(nodeName, clusterName, NODE_DOWN)
		if err != nil {
			klog.Errorf("Cordon node err:%v", err)
			return
		}
		klog.Info("Cordon node success.")
		klog.Infof("clusterName:%v\n nodeName:%v\n", clusterName, nodeName)

		node, err := kubeClient.CoreV1().Nodes().Get(ctxFirst, nodeName, metav1.GetOptions{})
		nodeIP, err := func() (string, error) {
			for _, address := range node.Status.Addresses {
				if address.Type == "InternalIP" {
					return address.Address, nil
				}
			}
			return "", nil
		}()

		write := db.NodeDataInfo{
			NodeName:    nodeName,
			NodeIP:      nodeIP,
			ClusterName: clusterName,
			ModuleName:  NODE_DOWN,
		}
		exist, _ := write.Check()
		if exist {
			continue
		}
		newfound++
		if newfound > 0 {
			klog.Infof("DownTimeNodes get WithDownNodeIPs【集群:%v】【节点:%v】【IP:%v】", clusterName, nodeName, nodeIP)
			alerter.DingTalkSend(h.cc.NodeDownTime.DingTalk, msg)
		}
		msg += fmt.Sprintf("node:%v ip:%v", nodeName, nodeIP)
		write.Reason = NODE_DOWN_REASON
		write.FirstDate = firstDate
		_, err = write.Add()
		if err != nil {
			klog.Errorf("DownTimeNodes abnormal cordonNode AddOrGet err:%v", err)
			klog.Infof("clusterName:%v nodeName:%v", clusterName, nodeName)
		}
		klog.Info("DownTimeNodes abnormal cordonNode AddOrGet success.")
		klog.Infof("clusterName:%v nodeName:%v", clusterName, nodeName)
	}
}
