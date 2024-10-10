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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"strings"
	"time"
)

func (h *Horuser) RecoveryManager(ctx context.Context) error {
	go wait.UntilWithContext(ctx, h.recoveryCheck, time.Duration(h.cc.NodeRecovery.IntervalSecond)*time.Second)
	<-ctx.Done()
	return nil
}

func (h *Horuser) DownTimeRecoveryManager(ctx context.Context) error {
	go wait.UntilWithContext(ctx, h.recoveryCheck, time.Duration(h.cc.NodeRecovery.IntervalSecond)*time.Second)
	<-ctx.Done()
	return nil
}

func (h *Horuser) downTimeRecoveryCheck(ctx context.Context) {
	data, err := db.GetRecoveryNodeDataInfoDate(h.cc.NodeRecovery.DayNumber)
	if err != nil {
		klog.Errorf("recovery check GetRecoveryNodeDataInfoDate err:%v", err)
		return
	}
	if len(data) == 0 {
		klog.Info("recovery check GetRecoveryNodeDataInfoDate zero.")
		return
	}
	wp := workerpool.New(50)
	for _, d := range data {
		d := d
		wp.Submit(func() {
			h.recoveryNodes(d)
			h.downTimeRecoveryNodes(d)
		})

	}
	wp.StopWait()
}

func (h *Horuser) recoveryCheck(ctx context.Context) {
	data, err := db.GetRecoveryNodeDataInfoDate(h.cc.NodeRecovery.DayNumber)
	if err != nil {
		klog.Errorf("recovery check GetRecoveryNodeDataInfoDate err:%v", err)
		return
	}
	if len(data) == 0 {
		klog.Info("recovery check GetRecoveryNodeDataInfoDate zero.")
		return
	}
	wp := workerpool.New(50)
	for _, d := range data {
		d := d
		wp.Submit(func() {
			h.downTimeRecoveryNodes(d)
		})

	}
	wp.StopWait()
}

func (h *Horuser) recoveryNodes(n db.NodeDataInfo) {
	promAddr := h.cc.PromMultiple[n.ClusterName]
	if promAddr == "" {
		klog.Error("recoveryNodes promAddr by clusterName empty.")
		klog.Infof("clusterName:%v nodeName:%v", n.ClusterName, n.NodeName)
		return
	}
	vecs, err := h.InstantQuery(promAddr, n.RecoveryQL, n.ClusterName, h.cc.NodeRecovery.PromQueryTimeSecond)
	if err != nil {
		klog.Errorf("recoveryNodes InstantQuery err:%v", err)
		klog.Infof("recoveryQL:%v", n.RecoveryQL)
		return
	}
	if len(vecs) != 1 {
		klog.Infof("Expected 1 result, but got:%d", len(vecs))
		return
	}
	if err != nil {
		klog.Errorf("recoveryNodes InstantQuery err:%v", err)
		klog.Infof("recoveryQL:%v", n.RecoveryQL)
		return
	}
	klog.Info("recoveryNodes InstantQuery success.")

	err = h.UnCordon(n.NodeName, n.ClusterName)
	res := "Success"
	if err != nil {
		res = fmt.Sprintf("result failed:%v", err)
	}
	msg := fmt.Sprintf("\n【集群: %v】\n【封锁节点恢复调度】\n【已恢复调度节点: %v】\n【处理结果：%v】\n【日期: %v】\n", n.ClusterName, n.NodeName, res, n.CreateTime)
	alerter.DingTalkSend(h.cc.NodeRecovery.DingTalk, msg)
	alerter.SlackSend(h.cc.CustomModular.Slack, msg)

	success, err := n.RecoveryMarker()
	if err != nil {
		klog.Errorf("RecoveryMarker result failed err:%v", err)
		return
	}
	klog.Infof("RecoveryMarker result success:%v", success)
}

func (h *Horuser) downTimeRecoveryNodes(n db.NodeDataInfo) {
	promAddr := h.cc.PromMultiple[n.ClusterName]
	if promAddr == "" {
		klog.Error("recoveryNodes promAddr by clusterName empty.")
		klog.Infof("clusterName:%v nodeName:%v", n.ClusterName, n.NodeName)
		return
	}
	vecs, err := h.InstantQuery(promAddr, strings.Join(n.DownTimeRecoveryQL, " "), n.ClusterName, h.cc.NodeDownTime.PromQueryTimeSecond)
	if err != nil {
		klog.Errorf("recoveryNodes InstantQuery err:%v", err)
		klog.Infof("downTimeRecoveryQL:%v", n.DownTimeRecoveryQL)
		return
	}
	if len(vecs) != 1 {
		klog.Infof("Expected 1 result, but got:%d", len(vecs))
		return
	}
	if err != nil {
		klog.Errorf("recoveryNodes InstantQuery err:%v", err)
		klog.Infof("recoveryQL:%v", n.DownTimeRecoveryQL)
		return
	}
	klog.Info("recoveryNodes InstantQuery success.")

	err = h.UnCordon(n.NodeName, n.ClusterName)
	res := "Success"
	if err != nil {
		res = fmt.Sprintf("result failed:%v", err)
	}
	msg := fmt.Sprintf("\n【集群: %v】\n【宕机节点已达到恢复临界点】\n【已恢复调度节点: %v】\n【处理结果：%v】\n【日期: %v】\n", n.ClusterName, n.NodeName, res, n.CreateTime)
	alerter.DingTalkSend(h.cc.NodeDownTime.DingTalk, msg)
	alerter.SlackSend(h.cc.NodeDownTime.Slack, msg)

	success, err := n.DownTimeRecoveryMarker()
	if err != nil {
		klog.Errorf("DownTimeRecoveryMarker result failed err:%v", err)
		return
	}
	klog.Infof("DownTimeRecoveryMarker result success:%v", success)
}
