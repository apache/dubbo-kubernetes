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
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	_ "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/admin/util/prometheus/prometheusclient"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"sync"
	"time"
)

func GetServiceTabDistribution(rt core_runtime.Runtime, req *model.ServiceTabDistributionReq) ([]*model.ServiceTabDistributionResp, error) {
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
	return res, nil
}

func GetSearchServices(rt core_runtime.Runtime) ([]*model.ServiceSearchResp, error) {
	res := make([]*model.ServiceSearchResp, 0)
	serviceMap := make(map[string]*model.ServiceSearch)
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}
	if err := manager.List(rt.AppContext(), dataplaneList); err != nil {
		return nil, err
	}
	// 通过dataplane extension字段获取所有revision
	revisions := make(map[string]struct{}, 0)
	for _, dataplane := range dataplaneList.Items {
		rev, ok := dataplane.Spec.GetExtensions()[v1alpha1.Revision]
		if ok {
			revisions[rev] = struct{}{}
		}
	}
	// 遍历 revisions
	for rev := range revisions {
		metadata := &mesh.MetaDataResource{
			Spec: &v1alpha1.MetaData{},
		}
		err := manager.Get(rt.AppContext(), metadata, store.GetByRevision(rev))
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
	return res, nil
}

func BannerSearchServices(rt core_runtime.Runtime, req *model.SearchReq) ([]*model.ServiceSearchResp, error) {
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

	return res, nil
}

func GetServiceDependencies(rt core_runtime.Runtime, req *model.ServiceDependenciesReq) ([]*model.ServiceDependenciesResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.ServiceName)); err != nil {
		return nil, err
	}
	// 键为调用方服务名称，值为被调用方服务名称列表
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

// GetServiceMetrics 获取 Dubbo 微服务集群中各服务的关键指标
func GetServiceMetrics(rt core_runtime.Runtime, req *model.ServiceMetricsReq) ([]*model.ServiceMetricsResp, error) {
	var serviceMetricsList []*model.ServiceMetricsResp
	pc := prometheusclient.NewPrometheusClient(string(*rt.Config().Admin.Prometheus))

	services, err := getAllServices(rt, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for _, service := range services {
		wg.Add(1)
		go func(svc string) {
			defer wg.Done()
			metrics, err := fetchMetricsForService(pc, svc)
			if err != nil {
				// 记录错误并继续
				fmt.Printf("Error fetching metrics for service %s: %v\n", svc, err)
				mu.Lock()
				errs = append(errs, fmt.Errorf("service %s: %v", svc, err))
				mu.Unlock()
				return
			}
			mu.Lock()
			serviceMetricsList = append(serviceMetricsList, metrics)
			mu.Unlock()
		}(service)
	}

	wg.Wait()

	if len(errs) > 0 {
		return serviceMetricsList, fmt.Errorf("encountered errors: %v", errs)
	}

	if req.Page > 0 && req.PageSize > 0 {
		start := (req.Page - 1) * req.PageSize
		end := start + req.PageSize
		if start >= len(serviceMetricsList) {
			return []*model.ServiceMetricsResp{}, nil
		}
		if end > len(serviceMetricsList) {
			end = len(serviceMetricsList)
		}
		serviceMetricsList = serviceMetricsList[start:end]
	}

	return serviceMetricsList, nil
}

func getAllServices(rt core_runtime.Runtime, req *model.ServiceMetricsReq) ([]string, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}
	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.Namespace)); err != nil {
		return nil, err
	}
	serviceSet := make(map[string]struct{})
	for _, dataplane := range dataplaneList.Items {
		for _, inbound := range dataplane.Spec.Networking.GetInbound() {
			if inbound.Tags != nil {
				if service, exists := inbound.Tags["dubbo.io/service"]; exists && service != "" {
					serviceSet[service] = struct{}{}
				}
			}
		}
	}
	var services []string
	for service := range serviceSet {
		if req.ServiceName != "" && req.ServiceName != service {
			continue
		}
		services = append(services, service)
	}

	return services, nil
}

func fetchMetricsForService(pc *prometheusclient.PrometheusClient, service string) (*model.ServiceMetricsResp, error) {
	var metrics model.ServiceMetricsResp
	metrics.ServiceName = service
	metrics.LastUpdated = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 请求速率
	reqRateQuery := fmt.Sprintf(`sum(rate(http_requests_total{service="%s"}[5m]))`, service)
	requestRate, err := queryPrometheusMetric(ctx, pc, reqRateQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query request rate: %v", err)
	}
	metrics.RequestRate = requestRate

	// 错误率
	errRateQuery := fmt.Sprintf(`sum(rate(http_requests_errors_total{service="%s"}[5m]))`, service)
	errorRate, err := queryPrometheusMetric(ctx, pc, errRateQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query error rate: %v", err)
	}
	metrics.ErrorRate = errorRate

	// 平均延迟
	latencyQuery := fmt.Sprintf(`
        avg(rate(http_request_duration_seconds_sum{service="%s"}[5m]) / rate(http_request_duration_seconds_count{service="%s"}[5m]))`, service, service)
	avgLatency, err := queryPrometheusMetric(ctx, pc, latencyQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query average latency: %v", err)
	}
	metrics.AvgLatency = avgLatency

	return &metrics, nil
}

func queryPrometheusMetric(ctx context.Context, pc *prometheusclient.PrometheusClient, query string) (float64, error) {
	results, err := pc.QueryWithContext(ctx, query)
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, fmt.Errorf("no results for query: %s", query)
	}
	if valueStr, ok := results[0].Value[1].(string); ok {
		var value float64
		_, err := fmt.Sscanf(valueStr, "%f", &value)
		if err != nil {
			return 0, fmt.Errorf("failed to parse float value: %v", err)
		}
		return value, nil
	}
	return 0, fmt.Errorf("unexpected result format")
}
