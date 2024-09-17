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
	"k8s.io/klog/v2"
	"sync"
	"time"
)

func (h *Horuser) CustomizeModularManager(ctx context.Context) error {
	go wait.UntilWithContext(ctx, h.CustomizeModular, time.Duration(h.cc.CustomModular.CheckIntervalSecond)*time.Second)
	<-ctx.Done()
	return nil
}

func (h *Horuser) CustomizeModular(ctx context.Context) {
	var wg sync.WaitGroup
	for clusterName, addr := range h.cc.PromMultiple {
		if _, exists := h.cc.CustomModular.KubeMultiple[clusterName]; !exists {
			klog.Infof("CustomizeModular config disable clusterName: %v", clusterName)
			continue
		}
		wg.Add(1)
		go func(clusterName, addr string) {
			defer wg.Done()
			h.CustomizeModularOnCluster(clusterName, addr)
		}(clusterName, addr)
	}
	wg.Wait()
}

func (h *Horuser) CustomizeModularOnCluster(clusterName, addr string) {
	klog.Infof("CustomizeModularOnCluster Start clusterName:%v", clusterName)
	for moduleName, ql := range h.cc.CustomModular.CheckQL {
		vecs, err := h.InstantQuery(addr, ql, clusterName, h.cc.CustomModular.PromQueryTimeSecond)
		if err != nil {
			klog.Errorf("CustomizeModularOnCluster InstantQuery err:%v", err)
			klog.Infof("clusterName:%v ql: %v", clusterName, ql)
			return
		}
		count := len(vecs)
		for index, vec := range vecs {
			vec := vec
			labelMap := vec.Metric
			nodeName := string(labelMap["node"])
			if nodeName == "" {
				klog.Warningf("CustomizeModularOnCluster empty.")
				klog.Infof("clusterName:%v moduleName:%v index:%d", clusterName, moduleName, index)
				continue
			}
			ip := string(labelMap["instance"])
			value := vec.Value.String()
			klog.Infof("CustomizeModularOnCluster Query result clusterName:%v moduleName:%v %d nodeName:%v value:%v count:%v",
				clusterName, moduleName, index+1, nodeName, value, count)
			h.CustomizeModularNodes(clusterName, moduleName, nodeName, ip)
		}
	}
}

func (h *Horuser) CustomizeModularNodes(clusterName, moduleName, nodeName, ip string) {
	today := time.Now().Format("2006-01-02")

	recoveryQL := h.cc.CustomModular.RecoveryQL[moduleName]

	data, err := db.GetDailyLimitNodeDataInfoDate(today, moduleName, clusterName)
	if err != nil {
		klog.Errorf("CustomizeModularNodes GetDailyLimitNodeDataInfoDate err:%v", err)
		return
	}
	klog.Infof("%v", data)

	dailyLimit := h.cc.CustomModular.CordonDailyLimit[moduleName]
	if len(data) > dailyLimit {
		msg := fmt.Sprintf("【日期:%v】 【集群:%v\n】 【模块今日 Cordon 节点数: %v】\n 【已达到今日上限: %v】\n [节点:%v]",
			data, clusterName, moduleName, dailyLimit, nodeName)
		alert.DingTalkSend(h.cc.CustomModular.DingTalk, msg)
		return
	}

	write := db.NodeDataInfo{
		NodeName:    nodeName,
		NodeIP:      ip,
		ClusterName: clusterName,
		ModuleName:  moduleName,
		Reason:      moduleName,
		FirstDate:   today,
		RecoveryQL:  recoveryQL,
	}
	pass, _ := write.Check()
	if pass {
		klog.Infof("CustomizeModularNodes already existing clusterName:%v nodeName:%v moduleName:%v", clusterName, nodeName, moduleName)
		return
	}
	err = h.Cordon(nodeName, clusterName)
	res := "Success"
	if err != nil {
		res = fmt.Sprintf("failed:%v", err)
		klog.Errorf("Cordon failed:%v", res)
	}

	msg := fmt.Sprintf("【集群:%v】\n 【%s 插件 Cordon 节点:%v】\n 【结果: %v】\n 【今日操作次数:%v",
		clusterName, moduleName, nodeName, res, len(data)+1)

	klog.Infof("Attempting to send DingTalk message: %s", msg)
	alert.DingTalkSend(h.cc.CustomModular.DingTalk, msg)
	klog.Infof("DingTalk message sent")

	_, err = write.AddOrGet()
	if err != nil {
		klog.Errorf("CustomizeModularNodes AddOrGet err:%v", err)
		klog.Infof("moduleName:%v nodeName:%v", moduleName, write.NodeName)
	}
	klog.Infof("CustomizeModularNodes AddOrGet success moduleName:%v nodeName:%v", moduleName, write.NodeName)
}
