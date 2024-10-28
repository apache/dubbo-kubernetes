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
	"sort"
	"strconv"
	"time"
)

import (
	"github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	_ "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func GetServiceTabDistribution(rt core_runtime.Runtime, req *model.ServiceTabDistributionReq) (*model.SearchPaginationResult, error) {
	manager := rt.ResourceManager()
	mappingList := &mesh.MappingResourceList{}

	serviceName := req.ServiceName

	if err := manager.List(rt.AppContext(), mappingList); err != nil {
		return nil, err
	}

	res := make([]*model.ServiceTabDistributionResp, 0)

	for _, mapping := range mappingList.Items {
		// 找到对应serviceName的appNames
		if mapping.Spec.InterfaceName == serviceName {
			for _, appName := range mapping.Spec.ApplicationNames {
				dataplaneList := &mesh.DataplaneResourceList{}
				// 每拿到一个appName，都将对应的实例数据填充进dataplaneList, 再通过dataplane拿到这个appName对应的所有实例
				if err := manager.List(rt.AppContext(), dataplaneList, store.ListByApplication(appName)); err != nil {
					return nil, err
				}

				// 拿到了appName，接下来从dataplane取实例信息
				for _, dataplane := range dataplaneList.Items {
					metadata := &mesh.MetaDataResource{
						Spec: &v1alpha1.MetaData{},
					}
					if err := manager.Get(rt.AppContext(), metadata, store.GetByRevision(dataplane.Spec.GetExtensions()[v1alpha1.Revision]), store.GetByType(dataplane.Spec.GetExtensions()["registry-type"])); err != nil {
						return nil, err
					}
					respItem := &model.ServiceTabDistributionResp{}
					res = append(res, respItem.FromServiceDataplaneResource(dataplane, metadata, appName, req))
				}

			}
		}
	}

	pagedRes := ToSearchPaginationResult(res, model.ByServiceInstanceName(res), req.PageReq)

	return pagedRes, nil
}

func GetSearchServices(rt core_runtime.Runtime, req *model.ServiceSearchReq) (*model.SearchPaginationResult, error) {
	if req.Keywords != "" {
		return BannerSearchServices(rt, req)
	}

	res := make([]*model.ServiceSearchResp, 0)
	serviceMap := make(map[string]*model.ServiceSearch)
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList); err != nil {
		return nil, err
	}
	// 通过dataplane extension字段获取所有revision
	revisions := make(map[string]string, 0)
	for _, dataplane := range dataplaneList.Items {
		rev, ok := dataplane.Spec.GetExtensions()[v1alpha1.Revision]
		if ok {
			revisions[rev] = dataplane.Spec.GetExtensions()["registry-type"]
		}
	}

	// 遍历 revisions
	for rev, t := range revisions {
		metadata := &mesh.MetaDataResource{
			Spec: &v1alpha1.MetaData{},
		}
		err := manager.Get(rt.AppContext(), metadata, store.GetByRevision(rev), store.GetByType(t))
		if err != nil {
			return nil, err
		}
		for _, serviceInfo := range metadata.Spec.Services {
			if _, ok := serviceMap[serviceInfo.Name]; ok {
				serviceMap[serviceInfo.Name].FromServiceInfo(serviceInfo)
			} else {
				serviceSearch := model.NewServiceSearch(serviceInfo.Name)
				serviceSearch.FromServiceInfo(serviceInfo)
				serviceMap[serviceInfo.Name] = serviceSearch
			}
		}
	}

	for _, serviceSearch := range serviceMap {
		serviceSearchResp := model.NewServiceSearchResp()
		serviceSearchResp.FromServiceSearch(serviceSearch)
		res = append(res, serviceSearchResp)
	}

	pagedRes := ToSearchPaginationResult(res, model.ByServiceName(res), req.PageReq)
	return pagedRes, nil
}

func BannerSearchServices(rt core_runtime.Runtime, req *model.ServiceSearchReq) (*model.SearchPaginationResult, error) {
	res := make([]*model.ServiceSearchResp, 0)

	manager := rt.ResourceManager()
	mappingList := &mesh.MappingResourceList{}

	if req.Keywords != "" {
		if err := manager.List(rt.AppContext(), mappingList, store.ListByNameContains(req.Keywords)); err != nil {
			return nil, err
		}

		for _, mapping := range mappingList.Items {
			serviceSearchResp := model.NewServiceSearchResp()
			serviceSearchResp.ServiceName = mapping.GetMeta().GetName()
			res = append(res, serviceSearchResp)
		}
	}

	pagedRes := ToSearchPaginationResult(res, model.ByServiceName(res), req.PageReq)

	return pagedRes, nil
}

func ToSearchPaginationResult[T any](services []T, data sort.Interface, req model.PageReq) *model.SearchPaginationResult {
	res := model.NewSearchPaginationResult()

	list := make([]T, 0)

	sort.Sort(data)
	lenFilteredItems := len(services)
	pageSize := lenFilteredItems
	offset := 0
	paginationEnabled := req.PageSize != 0
	if paginationEnabled {
		pageSize = req.PageSize
		offset = req.PageOffset
	}

	for i := offset; i < offset+pageSize && i < lenFilteredItems; i++ {
		list = append(list, services[i])
	}

	nextOffset := ""
	if paginationEnabled {
		if offset+pageSize < lenFilteredItems { // set new offset only if we did not reach the end of the collection
			nextOffset = strconv.Itoa(offset + req.PageSize)
		}
	}

	res.List = list
	res.PageInfo = &core_model.Pagination{
		Total:      uint32(lenFilteredItems),
		NextOffset: nextOffset,
	}

	return res
}

func GetServiceDependencies(rt core_runtime.Runtime, req *model.ServiceDependenciesReq) ([]*model.ServiceDependenciesResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.ServiceName)); err != nil {
		return nil, err
	}
	// Key is the caller service name, value is the list of callee service names
	dependencyMap := make(map[string]map[string]struct{})
	for _, dataplane := range dataplaneList.Items {
		sourceService := dataplane.Spec.GetIdentifyingService()
		if sourceService == "" {
			continue
		}
		if _, exists := dependencyMap[sourceService]; !exists {
			dependencyMap[sourceService] = make(map[string]struct{})
		}
		for _, outbound := range dataplane.Spec.Networking.Outbound {
			targetService := outbound.GetService()
			if targetService == "" {
				continue
			}
			dependencyMap[sourceService][targetService] = struct{}{}
		}
	}
	var dependencies []*model.ServiceDependenciesResp
	for source, targets := range dependencyMap {
		if req.ServiceName != "" && req.ServiceName != source {
			continue
		}
		var targetList []string
		for target := range targets {
			targetList = append(targetList, target)
		}
		dependencies = append(dependencies, &model.ServiceDependenciesResp{
			SourceService:     source,
			DependentServices: targetList,
		})
	}
	return dependencies, nil
}

func GetServiceMetrics(rt core_runtime.Runtime, req *model.ServiceMetricsReq) (*model.ServiceMetricsResp, error) {
	if req.ServiceName == "" {
		return nil, errors.New("ServiceName is required")
	}
	if len(req.MetricNames) == 0 {
		return nil, errors.New("MetricNames is required")
	}
	prometheusClient, err := prometheusclient.NewPrometheusClient(rt)
	if err != nil {
		return nil, err
	}
	prometheusAPI := promv1.NewAPI(prometheusClient)

	metricsData := make([]model.ServiceMetricsData, 0)

	// Iterate over the metric names specified by the user and query the data
	for _, metricName := range req.MetricNames {
		// Construct the query statement
		query := fmt.Sprintf(`%s{service_name="%s"}`, metricName, req.ServiceName)

		// Specify the time range
		startTime := req.StartTime
		endTime := req.EndTime
		if startTime.IsZero() {
			startTime = time.Now().Add(-time.Hour) // Default to past hour data
		}
		if endTime.IsZero() {
			endTime = time.Now()
		}

		values, err := prometheusclient.QueryPrometheusRange(prometheusAPI, query, startTime, endTime)
		if err != nil {
			continue
		}
		metricData := model.ServiceMetricsData{
			MetricName: metricName,
			Values:     values,
		}
		metricsData = append(metricsData, metricData)
	}

	response := &model.ServiceMetricsResp{
		ServiceName: req.ServiceName,
		Metrics:     metricsData,
	}
	return response, nil
}

func GetServiceHealthStatus(rt core_runtime.Runtime, req *model.ServiceHealthStatusReq) (*model.ServiceHealthStatusResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList); err != nil {
		return nil, err
	}
	// Create a map to aggregate instance health status by service name
	serviceInstanceHealth := make(map[string]*serviceHealthCounts)
	for _, dataplane := range dataplaneList.Items {
		inbounds := dataplane.Spec.Networking.GetInbound()
		if len(inbounds) == 0 {
			continue
		}
		for _, inbound := range inbounds {
			serviceName, ok := inbound.Tags["dubbo.io/service"]
			if !ok {
				continue
			}
			// Initialize health statistics for the service
			if _, exists := serviceInstanceHealth[serviceName]; !exists {
				serviceInstanceHealth[serviceName] = &serviceHealthCounts{}
			}

			// Update service instance counts
			counts := serviceInstanceHealth[serviceName]
			counts.Total++

			// Get instance health status
			isHealthy := false
			if inbound.Health != nil {
				isHealthy = inbound.Health.Ready
			}

			if isHealthy {
				counts.Healthy++
			} else {
				counts.Unhealthy++
			}
		}
	}
	services := make([]model.ServiceHealthInfo, 0)
	for serviceName, counts := range serviceInstanceHealth {
		healthStatus := "healthy"
		if counts.Unhealthy > 0 {
			healthStatus = "unhealthy"
		}
		serviceInfo := model.ServiceHealthInfo{
			ServiceName:          serviceName,
			HealthyInstanceNum:   counts.Healthy,
			UnhealthyInstanceNum: counts.Unhealthy,
			TotalInstanceNum:     counts.Total,
			HealthStatus:         healthStatus,
		}
		services = append(services, serviceInfo)
	}
	response := &model.ServiceHealthStatusResp{
		Services: services,
	}
	return response, nil
}

// Helper struct for tracking the health status of service instances
type serviceHealthCounts struct {
	Healthy   int
	Unhealthy int
	Total     int
}

func GetServiceDependencyTopology(rt core_runtime.Runtime, req *model.ServiceDependencyTopologyReq) ([]*model.ServiceDependencyTopologyResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}
	// Retrieve all Dataplane resources
	if err := manager.List(rt.AppContext(), dataplaneList); err != nil {
		return nil, err
	}
	//Create a map to store services and their dependencies
	serviceDependencies := make(map[string]*model.ServiceNode)
	for _, dataplane := range dataplaneList.Items {
		// Get the list of inbound services
		inbounds := dataplane.Spec.Networking.GetInbound()
		if len(inbounds) == 0 {
			continue
		}
		// Get the list of outbound services
		outbounds := dataplane.Spec.Networking.GetOutbound()
		for _, inbound := range inbounds {
			sourceServiceName, ok := inbound.Tags["service"]
			if !ok {
				continue
			}
			// Initialize source service node
			sourceNode, exists := serviceDependencies[sourceServiceName]
			if !exists {
				sourceNode = &model.ServiceNode{
					ServiceName:  sourceServiceName,
					Dependencies: []string{},
				}
				serviceDependencies[sourceServiceName] = sourceNode
			}
			for _, outbound := range outbounds {
				targetServiceName, ok := outbound.Tags["service"]
				if !ok {
					continue
				}
				sourceNode.Dependencies = append(sourceNode.Dependencies, targetServiceName)
			}
		}
	}
	res := make([]*model.ServiceDependencyTopologyResp, 0)
	for _, node := range serviceDependencies {
		respItem := &model.ServiceDependencyTopologyResp{
			ServiceName:  node.ServiceName,
			Dependencies: node.Dependencies,
		}
		res = append(res, respItem)
	}
	return res, nil
}
