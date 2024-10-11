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
	"os/exec"
	"strings"
	"time"
)

func (h *Horuser) DowntimeRestartManager(ctx context.Context) error {
	go wait.UntilWithContext(ctx, h.RestartOrRepair, time.Duration(h.cc.NodeDownTime.IntervalSecond)*time.Second)
	<-ctx.Done()
	return nil
}

func (h *Horuser) RestartOrRepair(ctx context.Context) {
	nodes, err := db.GetRestartNodeDataInfoDate()
	if err != nil {
		klog.Errorf("RestartOrRepair GetRestartNodeDataInfoDate err:%v", err)
		return
	}
	if len(nodes) == 0 {
		klog.Info("RestartOrRepair to zero.")
	}
	klog.Infof("RestartOrRepair Count:%v", len(nodes))
	wp := workerpool.New(10)
	for _, n := range nodes {
		n := n
		wp.Submit(func() {
			h.TryRestart(n)
		})
	}
}

func (h *Horuser) TryRestart(node db.NodeDataInfo) {
	err := h.Drain(node.NodeName, node.ClusterName)
	if err != nil {
		klog.Errorf("Drain node err:%v", err)
		return
	}
	klog.Info("Drain node success.")
	klog.Infof("clusterName:%v\n nodeName:%v\n", node.ClusterName, node.NodeName)

	success, err := node.RestartMarker()
	if err != nil {
		klog.Errorf("Error getting RestartMarker for nodeName:%v: err:%v", node.NodeName, err)
		return
	}
	klog.Infof("RestartMarker result success:%v", success)

	if success {
		msg := fmt.Sprintf("\n【宕机节点等待腾空后重启】\n【节点:%v】\n【日期:%v】\n【集群:%v】\n", node.NodeName, node.FirstDate, node.ClusterName)
		alerter.DingTalkSend(h.cc.NodeDownTime.DingTalk, msg)

		cmd := exec.Command("/bin/bash", "core/horuser/restart.sh", node.NodeIP, h.cc.NodeDownTime.AllSystemUser, h.cc.NodeDownTime.AllSystemPassword)
		output, err := cmd.CombinedOutput()
		if err != nil {
			klog.Errorf("Failed restart for Output:%v node:%v: err:%v", string(output), node.NodeName, err)
			return
		}
		klog.Infof("Successfully restarted node %v.", node.NodeName)

		rq := len(h.cc.NodeDownTime.AbnormalRecoveryQL)
		ql := strings.Join(h.cc.NodeDownTime.AbnormalRecoveryQL, " or ")
		vecs, err := h.InstantQuery(h.cc.PromMultiple[node.ClusterName], ql, node.ClusterName, h.cc.NodeDownTime.PromQueryTimeSecond)
		if err != nil {
			klog.Errorf("Failed to query Prometheus for recovery threshold after restart: %v", err)
			return
		}

		if len(vecs) == rq {
			klog.Infof("Node %v has reached recovery threshold after restart.", node.NodeName)

			err = h.UnCordon(node.NodeName, node.ClusterName)
			if err != nil {
				klog.Errorf("Uncordon node failed after restart: %v", err)
				return
			}

			msg = fmt.Sprintf("\n【集群: %v】\n【宕机节点已恢复】\n【恢复节点: %v】\n【处理结果：成功】\n【日期: %v】\n", node.ClusterName, node.NodeName, node.FirstDate)
			alerter.DingTalkSend(h.cc.NodeDownTime.DingTalk, msg)
			alerter.SlackSend(h.cc.NodeDownTime.Slack, msg)

		} else {
			klog.Infof("Node %v has not reached recovery threshold after restart.", node.NodeName)
		}

	} else {
		klog.Infof("RestartMarker did not success for node %v", node.NodeName)
	}

	if node.Restart > 2 {
		klog.Error("The node has already been rebooted more than twice.")
		return
	}
}
