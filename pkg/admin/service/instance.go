/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package service

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func BannerSearchInstances(rt core_runtime.Runtime, req *model.SearchReq) ([]*model.SearchInstanceResp, *core_model.Pagination, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.Keywords), store.ListByPage(req.PageSize, strconv.Itoa(req.CurPage))); err != nil {
		return nil, nil, err
	}

	res := make([]*model.SearchInstanceResp, len(dataplaneList.Items))
	for i, item := range dataplaneList.Items {
		res[i] = &model.SearchInstanceResp{}
		res[i] = res[i].FromDataplaneResource(item)
	}

	return res, &dataplaneList.Pagination, nil
}

func SearchInstances(rt core_runtime.Runtime, req *model.SearchInstanceReq) ([]*model.SearchInstanceResp, *core_model.Pagination, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByPage(req.PageSize, strconv.Itoa(req.CurPage))); err != nil {
		return nil, nil, err
	}

	res := make([]*model.SearchInstanceResp, len(dataplaneList.Items))
	for i, item := range dataplaneList.Items {
		res[i] = &model.SearchInstanceResp{}
		res[i] = res[i].FromDataplaneResource(item)
	}

	return res, &dataplaneList.Pagination, nil
}

func GetInstanceDetail(rt core_runtime.Runtime, req *model.InstanceDetailReq) ([]*model.InstanceDetailResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.InstanceName)); err != nil {
		return nil, err
	}

	instMap := make(map[string]*model.InstanceDetail)
	for _, dataplane := range dataplaneList.Items {

		// instName := dataplane.Meta.GetLabels()[mesh_proto.InstanceTag]//This tag is "" in universal mode
		instName := dataplane.Meta.GetName()
		var instanceDetail *model.InstanceDetail
		if _, ok := instMap[instName]; ok {
			// found previously recorded instance detail in instMap
			// the detail should be merged with the new instance detail
			instanceDetail = instMap[instName]
		} else {
			// the instance information appears for the 1st time
			instanceDetail = model.NewInstanceDetail()
		}
		instanceDetail.Merge(dataplane) // convert dataplane info to instance detail
		instMap[instName] = instanceDetail
	}

	resp := make([]*model.InstanceDetailResp, 0, len(instMap))
	for _, instDetail := range instMap {
		respItem := &model.InstanceDetailResp{}
		resp = append(resp, respItem.FromInstanceDetail(instDetail))
	}
	return resp, nil
}

func GetInstanceMetrics(rt core_runtime.Runtime, req *model.MetricsReq) ([]*model.MetricsResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}
	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.InstanceName)); err != nil {
		return nil, err
	}
	instMap := make(map[string]*model.InstanceDetail)
	resp := make([]*model.MetricsResp, 0)
	for _, dataplane := range dataplaneList.Items {
		instName := dataplane.Meta.GetName()
		var instanceDetail *model.InstanceDetail
		if detail, ok := instMap[instName]; ok {
			instanceDetail = detail
		} else {
			instanceDetail = model.NewInstanceDetail()
		}
		instanceDetail.Merge(dataplane)
		metrics, err := fetchMetricsData(dataplane.GetIP(), 22222)
		if err != nil {
			continue
		}
		metricsResp := &model.MetricsResp{
			InstanceName: instName,
			Metrics:      metrics,
		}
		resp = append(resp, metricsResp)
	}
	return resp, nil
}

func fetchMetricsData(ip string, port int) ([]model.Metric, error) {
	url := fmt.Sprintf("http://%s:%d/metrics", ip, port)
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	metrics, err := parsePrometheusData(string(body))
	if err != nil {
		return nil, err
	}
	return metrics, nil
}

// parsePrometheusData parses Prometheus text format data and converts it to a slice of Metrics.
func parsePrometheusData(data string) ([]model.Metric, error) {
	var metrics []model.Metric
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) != 2 {
			continue
		}

		metricPart := parts[0]
		valuePart := parts[1]

		// Extract the metric name and labels
		nameAndLabels := strings.SplitN(metricPart, "{", 2)
		if len(nameAndLabels) != 2 {
			continue
		}

		name := nameAndLabels[0]
		labelsPart := strings.TrimSuffix(nameAndLabels[1], "}")

		labels := make(map[string]string)
		for _, label := range strings.Split(labelsPart, ",") {
			if label == "" {
				continue
			}
			labelParts := strings.SplitN(label, "=", 2)
			if len(labelParts) == 2 {
				labels[labelParts[0]] = strings.Trim(labelParts[1], `"`)
			}
		}

		// Parse the value
		var value float64
		fmt.Sscanf(valuePart, "%f", &value)

		metrics = append(metrics, model.Metric{
			Name:   name,
			Labels: labels,
			Value:  value,
		})
	}

	return metrics, nil
}
