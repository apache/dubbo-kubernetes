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
	"github.com/prometheus/client_golang/prometheus"
	"strings"
)

var (
	MultipleInfo = prometheus.NewDesc(
		"horus_multiple_info",
		"horus_multiple_info",
		[]string{
			"cluster_name",
			"prometheus_multiple_address",
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
	buttons := map[bool]string{}
	modularKey := buttons[h.cc.CustomModular.Enabled]
	info[modularKey] = kFunc(h.cc.CustomModular.KubeMultiple)

	for clusterName, address := range h.cc.PromMultiple {
		p := prometheus.MustNewConstMetric(MultipleInfo,
			prometheus.GaugeValue, 1,
			clusterName,
			address,
		)
		ch <- p
	}
}

func (h *Horuser) Describe(ch chan<- *prometheus.Desc) {
	ch <- MultipleInfo
}
