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
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/admin/util/prometheusclient"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"strconv"
	"time"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func BannerSearchInstances(rt core_runtime.Runtime, req *model.SearchReq) (*model.SearchPaginationResult, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.Keywords), store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
		return nil, err
	}

	res := model.NewSearchPaginationResult()
	list := make([]*model.SearchInstanceResp, len(dataplaneList.Items))
	for i, item := range dataplaneList.Items {
		list[i] = model.NewSearchInstanceResp()
		list[i] = list[i].FromDataplaneResource(item)
	}

	res.List = list
	res.PageInfo = &dataplaneList.Pagination

	return res, nil
}

func SearchInstances(rt core_runtime.Runtime, req *model.SearchInstanceReq) (*model.SearchPaginationResult, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if req.Keywords == "" {
		if err := manager.List(rt.AppContext(), dataplaneList, store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
			return nil, err
		}
	} else {
		if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.Keywords), store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
			return nil, err
		}
	}

	res := model.NewSearchPaginationResult()
	list := make([]*model.SearchInstanceResp, len(dataplaneList.Items))
	for i, item := range dataplaneList.Items {
		list[i] = model.NewSearchInstanceResp()
		list[i] = list[i].FromDataplaneResource(item)
	}

	res.List = list
	res.PageInfo = &dataplaneList.Pagination

	return res, nil
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

func GetInstanceMetrics(rt core_runtime.Runtime, req *model.InstanceMetricsReq) (*model.InstanceMetricsResp, error) {
	if req.InstanceID == "" {
		return nil, errors.New("InstanceID is required")
	}
	if len(req.MetricNames) == 0 {
		return nil, errors.New("MetricNames is required")
	}
	prometheusClient, err := prometheusclient.NewPrometheusClient(rt)
	if err != nil {
		return nil, err
	}

	prometheusAPI := promv1.NewAPI(prometheusClient)
	metricsData := make([]model.InstanceMetricsData, 0)
	for _, metricName := range req.MetricNames {
		query := fmt.Sprintf(`%s{instance_id="%s"}`, metricName, req.InstanceID)
		startTime := req.StartTime
		endTime := req.EndTime
		if startTime.IsZero() {
			startTime = time.Now().Add(-time.Hour)
		}
		if endTime.IsZero() {
			endTime = time.Now()
		}

		values, err := prometheusclient.QueryPrometheusRange(prometheusAPI, query, startTime, endTime)
		if err != nil {
			return nil, fmt.Errorf("error querying metric %s: %v", metricName, err)
		}
		metricData := model.InstanceMetricsData{
			MetricName: metricName,
			Values:     values,
		}
		metricsData = append(metricsData, metricData)
	}
	response := &model.InstanceMetricsResp{
		InstanceID: req.InstanceID,
		Metrics:    metricsData,
	}

	return response, nil
}

func GetInstanceHealthStatus(rt core_runtime.Runtime, req *model.InstanceHealthStatusReq) (*model.InstanceHealthStatusResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}
	// Get all Dataplane resources
	if err := manager.List(rt.AppContext(), dataplaneList); err != nil {
		return nil, err
	}
	instances := make([]model.InstanceHealthInfo, 0)
	for _, dataplane := range dataplaneList.Items {
		// Get the list of inbound services
		inbounds := dataplane.Spec.Networking.GetInbound()
		if len(inbounds) == 0 {
			continue
		}
		for _, inbound := range inbounds {
			// Get health status
			healthStatus := "unknown"
			if inbound.Health != nil {
				if inbound.Health.Ready {
					healthStatus = "ready"
				} else {
					healthStatus = "notReady"
				}
			}
			instanceInfo := model.InstanceHealthInfo{
				InstanceID:   dataplane.GetMeta().GetName(),
				Application:  inbound.Tags["application"],
				IPAddress:    inbound.GetAddress(),
				Port:         inbound.GetPort(),
				HealthStatus: healthStatus,
			}
			instances = append(instances, instanceInfo)
		}
	}
	response := &model.InstanceHealthStatusResp{
		Instances: instances,
	}
	return response, nil
}
