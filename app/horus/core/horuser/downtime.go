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
	NODE_DOWN_REASON = "node_down"
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
	klog.Infof("DownTimeNodes Query Start clusterName:%v", clusterName)
	nodeDownTimeRes := make(map[string]int)
	cq := len(h.cc.NodeDownTime.AbnormalityQL)
	for _, ql := range h.cc.NodeDownTime.AbnormalityQL {
		ql := ql
		res, err := h.InstantQuery(addr, ql, clusterName, h.cc.NodeDownTime.PromQueryTimeSecond)
		if err != nil {
			klog.Errorf("downtimeNodes Instant Query err:%v", err)
			klog.Infof("clusterName:%v", clusterName)
			continue
		}

		for _, v := range res {
			v := v
			nodeName := string(v.Metric["node"])
			if nodeName == "" {
				klog.Errorf("downtimeNodes Instant Query nodeName empty.")
				klog.Infof("clusterName:%v metrics:%v", clusterName, v.Metric)
				continue
			}
			nodeDownTimeRes[nodeName]++
		}
	}
	WithDownNodeIPs := make(map[string]string)

	for node, count := range nodeDownTimeRes {
		if count < cq {
			klog.Errorf("downtimeNodes node not reach threshold")
			klog.Infof("clusterName:%v nodeName:%v threshold:%v count:%v", clusterName, node, cq, count)
			continue
		}
		nnti := fmt.Sprintf(h.cc.NodeDownTime.NodeNameToIPs, node)
		res, err := h.InstantQuery(addr, nnti, clusterName, h.cc.NodeDownTime.PromQueryTimeSecond)
		if len(res) == 0 {
			klog.Errorf("No results returned for query: %s", nnti)
			continue
		}
		if err != nil {
			klog.Errorf("downtimeNodes InstantQuery NodeName To IPs empty err:%v", err)
			klog.Infof("clusterName:%v nodeNameToIPs: %v, err:%v", clusterName, nnti, err)
			continue
		}
		str := ""
		for _, v := range res {
			str = string(v.Metric["instance"])
		}
		WithDownNodeIPs[node] = str
	}

	WithDownNodeIPsMsg := fmt.Sprintf("\n【%s】\n【集群：%v】\n【宕机：%v】", h.cc.NodeDownTime.DingTalk.Title, clusterName, len(WithDownNodeIPs))
	newfound := 0
	for nodeName, nodeIP := range WithDownNodeIPs {
		today := time.Now().Format("2006-01-02")
		err := h.Cordon(nodeName, clusterName, NODE_DOWN)
		if err != nil {
			klog.Errorf("Cordon node err:%v", err)
			klog.Infof("clusterName:%v nodeName:%v", clusterName, nodeName)
			return
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
		if newfound > 0 {
			klog.Infof("NodeDownTimeCheckOnCluster get toNodeNameips \n【集群:%v】\n 【总数:%v】\n 【细节:%v】\n", clusterName, len(nodeIP), nodeName)
			alert.DingTalkSend(h.cc.NodeDownTime.DingTalk, WithDownNodeIPsMsg)
		}
		WithDownNodeIPsMsg += fmt.Sprintf("node:%v ip:%v", nodeName, nodeIP)
		write.Reason = NODE_DOWN_REASON
		write.FirstDate = today
		_, err = write.Add()
		if err != nil {
			klog.Errorf("DownTimeNodes abnormal cordonNode AddOrGetOne err:%v", err)
			klog.Infof("cluster:%v node:%v", clusterName, nodeName)
		}
		klog.Infof("DownTimeNodes abnormal cordonNode AddOrGetOne cluster:%v node:%v", clusterName, nodeName)
	}
}
