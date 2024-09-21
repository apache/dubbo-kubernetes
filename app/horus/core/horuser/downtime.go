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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sync"
	"time"
)

const (
	NODE_DOWN        = "node_down"
	NODE_DOWN_REASON = "node_down_aegis"
	POWER_OFF        = "POWER_OFF"
	POWER_ON         = "POWER_ON"
)

func (h *Horuser) DownTimeManager(ctx context.Context) error {
	go wait.UntilWithContext(ctx, h.DownTimeCheck, time.Duration(h.cc.NodeDownTime.CheckIntervalSecond)*time.Second)
	<-ctx.Done()
	return nil
}

func (h *Horuser) DownTimeCheck(ctx context.Context) {
	var wg sync.WaitGroup

	for clusterName, addr := range h.cc.PromMultiple {
		clusterName := clusterName
		if _, exist := h.cc.NodeDownTime.KubeMultiple[clusterName]; !exist {
			klog.Infof("DownTimeCheck config disable")
			klog.Infof("clusterName: %v", clusterName)
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
	klog.Infof("DownTimeNodes QueryStart clusterName:%v", clusterName)
	resMap := map[string]int{}
	checkQl := len(h.cc.NodeDownTime.CheckQL)
	for _, ql := range h.cc.NodeDownTime.CheckQL {
		ql := ql
		res, err := h.InstantQuery(addr, ql, clusterName, h.cc.NodeDownTime.PromQueryTimeSecond)
		if err != nil {
			klog.Errorf("downtimeNodes InstantQuery err:%v", err)
			klog.Infof("clusterName:%v", clusterName)
			continue
		}

		for _, v := range res {
			v := v
			nodeName := string(v.Metric["node"])
			if nodeName == "" {
				klog.Errorf("downtimeNodes InstantQuery nodeName empty")
				klog.Infof("clusterName:%v metrics:%v", clusterName, v.Metric)
				continue
			}
			resMap[nodeName]++
		}
	}
	for node, count := range resMap {
		if count < checkQl {
			klog.Errorf("downtimeNodes node not reach threshold")
			klog.Infof("clusterName:%v node:%v threshold:%v count:%v", clusterName, node, checkQl, count)
			continue
		}
	}

	WithDownNodeIPs := map[string]string{}
	for node, count := range resMap {
		if count < checkQl {
			klog.Errorf("downtimeNodes node not reach threshold")
			klog.Infof("clusterName:%v nodeName:%v threshold:%v count:%v", clusterName, node, checkQl, count)
			continue
		}
		toNodeNameips := fmt.Sprintf(h.cc.NodeDownTime.NodeNameToIPs, node)
		res, err := h.InstantQuery(toNodeNameips, addr, clusterName, h.cc.NodeDownTime.PromQueryTimeSecond)
		if err != nil {
			klog.Errorf("downtimeNodes InstantQuery NodeName To IPs empty err:%v", err)
			klog.Infof("clusterName:%v toNodeNameips:%v err:%v", clusterName, toNodeNameips, err)
			continue
		}
		str := ""
		for _, v := range res {
			v := v
			str = string(v.Metric["instance"])
		}
		WithDownNodeIPs[node] = str
	}
	WithDownNodeIPsMsg := fmt.Sprintf("【%s】\n【集群：%v】\n【宕机：%v】\n", h.cc.NodeDownTime.DingTalk.Title, clusterName, len(WithDownNodeIPs))
	newfound := 0
	for nodeName, nodeIP := range WithDownNodeIPs {
		today := time.Now().Format("2006-01-02")
		err := h.Cordon(nodeName, clusterName, NODE_DOWN)
		if err != nil {
			klog.Errorf("Cordon node err:%v", err)
			klog.Infof("clusterName:%v nodeName:%v", clusterName, nodeName)
		}
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
		WithDownNodeIPsMsg += fmt.Sprintf("node:%v ip:%v", nodeName, nodeIP)
		write.Reason = NODE_DOWN_REASON
		write.FirstDate = today
		_, err = write.Add()
		if err != nil {
			klog.Errorf("NodeDownTimeCheckOnCluster abnormal cordonNode AddOrGetOne err:%v", err)
			klog.Infof("cluster:%v node:%v", clusterName, nodeName)
		}
		klog.Infof("NodeDownTimeCheckOnCluster abnormal cordonNode AddOrGetOne cluster:%v node:%v", clusterName, nodeName)
		if newfound > 0 {
			klog.Infof("NodeDownTimeCheckOnCluster get toNodeNameips result msg:%v clusterName:%v count:%v detail:%v", WithDownNodeIPsMsg, clusterName, len(nodeIP), nodeName)
			alert.DingTalkSend(h.cc.NodeDownTime.DingTalk, WithDownNodeIPsMsg)
		}
	}
}
