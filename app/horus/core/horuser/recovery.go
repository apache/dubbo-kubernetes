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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"time"
)

func (h *Horuser) RecoveryManager(ctx context.Context) error {
	go wait.UntilWithContext(ctx, h.recoveryCheck, time.Duration(h.cc.NodeRecovery.CheckIntervalSecond)*time.Second)
	<-ctx.Done()
	return nil
}

func (h *Horuser) recoveryCheck(ctx context.Context) {
	data, err := db.GetRecoveryNodeDataInfoDate(h.cc.NodeRecovery.DayNumber)
	if err != nil {
		klog.Errorf("recovery check GetRecoveryNodeDataInfoDate err:%v", err)
		return
	}
	if len(data) == 0 {
		klog.Errorf("recovery check GetRecoveryNodeDataInfoDate zero.")
		return
	}
	wp := workerpool.New(5)
	for _, d := range data {
		d := d
		wp.Submit(func() {
			h.recoveryNodes(d)
		})

	}
	wp.StopWait()
}

func (h *Horuser) recoveryNodes(n *db.NodeDataInfo) {
	addr := h.cc.PromMultiple[n.ClusterName]
	if addr == "" {
		klog.Errorf("recoveryNodes PromMultiple get addr empty.")
		klog.Infof("clusterName:%v nodeName:%v", n.ClusterName, n.NodeName)
		return
	}
	ql := fmt.Sprintf(n.RecoveryQL, n.NodeName)
	vecs, err := h.InstantQuery(addr, ql, n.ClusterName, h.cc.NodeRecovery.PromQueryTimeSecond)
	if err != nil {
		klog.Errorf("recoveryNodes instantQuery err:%v ql:%v", err, ql)
		return
	}
	if len(vecs) != 2 {
		return
	}
	klog.Infof("recoveryNodes check success.")

	err = h.UnCordon(n.NodeName, n.ClusterName)
	res := "success"
	if err != nil {
		res = fmt.Sprintf("failed:%v", err)
	}
	msg := fmt.Sprintf("【自愈检查 %v: 恢复节点调度】【集群: %v】\n【节点: %v】【日期: %v】\n"+
		"【自愈检查 QL: %v】", res, n.ClusterName, n.NodeName, n.CreateTime, ql)
	alert.DingTalkSend(h.cc.NodeRecovery.DingTalk, msg)

	pass, err := n.RecoveryMarker()
	klog.Infof("RecoveryMarker result pass:%v err:%v", pass, err)
}
