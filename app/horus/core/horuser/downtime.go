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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sync"
	"time"
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
}
