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

package prometheusclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	promModel "github.com/prometheus/common/model"
	"net/http"
	"time"
)

type PrometheusResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

type PrometheusClient struct {
	URL    string
	Client *http.Client
}

func NewPrometheusClient(rt core_runtime.Runtime) (promapi.Client, error) {
	prometheusURL := rt.Config().Admin.Prometheus
	client, err := promapi.NewClient(promapi.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (pc *PrometheusClient) Query(query string) ([]PrometheusResult, error) {
	return pc.QueryWithContext(context.Background(), query)
}

func (pc *PrometheusClient) QueryWithContext(ctx context.Context, query string) ([]PrometheusResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/query", pc.URL), nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("query", query)
	req.URL.RawQuery = q.Encode()

	resp, err := pc.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string             `json:"resultType"`
			Result     []PrometheusResult `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("query failed with status: %s", result.Status)
	}
	return result.Data.Result, nil
}

func QueryPrometheusRange(api promv1.API, query string, startTime, endTime time.Time) ([]model.MetricValue, error) {
	step := time.Duration(15 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, warnings, err := api.QueryRange(ctx, query, promv1.Range{
		Start: startTime,
		End:   endTime,
		Step:  step,
	})
	if err != nil {
		return nil, err
	}
	if len(warnings) > 0 {
		fmt.Println("Warnings:", warnings)
	}
	matrix, ok := result.(promModel.Matrix)
	if !ok {
		return nil, errors.New("unexpected result type")
	}

	values := make([]model.MetricValue, 0)
	for _, sampleStream := range matrix {
		for _, sample := range sampleStream.Values {
			metricValue := model.MetricValue{
				Timestamp: int64(sample.Timestamp / 1000),
				Value:     float64(sample.Value),
			}
			values = append(values, metricValue)
		}
	}
	return values, nil
}
