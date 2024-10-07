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
	"fmt"
	"github.com/apache/dubbo-kubernetes/app/horus/base/db"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
	"strings"
)

var (
	FeatureInfo = prometheus.NewDesc(
		"horus_feature_info",
		"Indicates the enabled status of specific features in different clusters, with each feature represented by its name, enabled state, and associated cluster name.",
		[]string{
			"feature",
			"enabled",
			"cluster_name",
		},
		nil)

	MultipleInfo = prometheus.NewDesc(
		"horus_multiple_info",
		"Tracks the Prometheus multiple addresses associated with different clusters, providing visibility into the Prometheus endpoints used by each cluster.",
		[]string{
			"cluster_name",
			"prometheus_multiple_address",
		},
		nil)

	NodeInfo = prometheus.NewDesc(
		"horus_node_info",
		"Provides detailed information about the nodes within a cluster, including node identity, operational state, maintenance details, and recovery status.",
		[]string{
			"node_name",
			"node_ip",
			"sn",
			"cluster_name",
			"module_name",
			"reason",
			"restart",
			"repair",
			"repair_ticket_url",
			"first_date",
			"create_time",
			"update_time",
			"recovery_ql",
			"recovery_mark",
		},
		nil)

	PodStagnativeInfo = prometheus.NewDesc(
		"pod_stagnative_info",
		"pod_stagnative_info",
		[]string{
			"pod_name",
			"pod_ip",
			"node_name",
			"cluster_name",
			"module_name",
			"reason",
			"first_date",
			"create_time",
			"update_time",
		},
		nil)
)

func (h *Horuser) Collect(ch chan<- prometheus.Metric) {
	kFunc := func(m map[string]string) string {
		s := []string{}
		for k := range m {
			s = append(s, k)
		}
		return strings.Join(s, ",")
	}
	info := map[string]string{}
	buttons := map[bool]string{true: "Open", false: "Close"}

	modularKey := fmt.Sprintf("custom modular,%s", buttons[h.cc.CustomModular.Enabled])
	info[modularKey] = kFunc(h.cc.CustomModular.KubeMultiple)
	downtimeKey := fmt.Sprintf("node downtime,%s", buttons[h.cc.NodeDownTime.Enabled])
	info[downtimeKey] = kFunc(h.cc.NodeDownTime.KubeMultiple)

	nodeFunc := func() {
		nodes, err := db.GetNode()
		if err != nil {
			klog.Errorf("horus metrics collect db get node err:%v", err)
			return
		}
		if len(nodes) == 0 {
			klog.Infof("horus metrics collect db zero err:%v", err)
			return
		}
		for _, v := range nodes {
			v := v
			ct := v.CreateTime.Local().Format("2006-01-02 15:04:05")
			ut := v.UpdateTime.Local().Format("2006-01-02 15:04:05")
			p := prometheus.MustNewConstMetric(NodeInfo,
				prometheus.GaugeValue, 1,
				v.NodeName,
				v.NodeIP,
				v.Sn,
				v.ClusterName,
				v.ModuleName,
				v.Reason,
				fmt.Sprintf("%d", v.Restart),
				fmt.Sprintf("%d", v.Repair),
				v.RepairTicketUrl,
				v.FirstDate,
				ct,
				ut,
				v.RecoveryQL,
				fmt.Sprintf("%d", v.RecoveryMark),
			)
			ch <- p
		}
	}
	nodeFunc()

	podFunc := func() {
		pods, err := db.GetPod()
		if err != nil {
			klog.Errorf("horus metrics collect db get pod err:%v", err)
			return
		}
		klog.Info("horus metrics collect db get pod success.")
		if len(pods) == 0 {
			klog.Infof("horus metrics collect db zero err:%v", err)
			return
		}
		for _, v := range pods {
			v := v
			ct := v.CreateTime.Local().Format("2006-01-02 15:04:05")
			ut := v.UpdateTime.Local().Format("2006-01-02 15:04:05")
			p := prometheus.MustNewConstMetric(PodStagnativeInfo,
				prometheus.GaugeValue, 1,
				v.PodName,
				v.PodIP,
				v.NodeName,
				v.ClusterName,
				v.ModuleName,
				v.Reason,
				v.FirstDate,
				ct,
				ut,
			)
			ch <- p
		}
	}
	podFunc()

	for k, clusterName := range info {
		s := strings.Split(k, ",")
		feature, enabled := s[0], s[1]
		p := prometheus.MustNewConstMetric(
			FeatureInfo,
			prometheus.GaugeValue, 1,
			feature,
			enabled,
			clusterName,
		)
		ch <- p
	}

	for clusterName, address := range h.cc.PromMultiple {
		p := prometheus.MustNewConstMetric(
			MultipleInfo,
			prometheus.GaugeValue, 1,
			clusterName,
			address,
		)
		ch <- p
	}
}

func (h *Horuser) Describe(ch chan<- *prometheus.Desc) {
	ch <- FeatureInfo
	ch <- MultipleInfo
	ch <- NodeInfo
}
