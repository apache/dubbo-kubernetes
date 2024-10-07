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
	apiV1 "github.com/prometheus/client_golang/api"
	prometheusV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"k8s.io/klog/v2"
	"time"
)

func (h *Horuser) InstantQuery(address, ql, clusterName string, timeWindowsSecond int64) (model.Vector, error) {
	client, err := apiV1.NewClient(apiV1.Config{Address: address})
	if err != nil {
		klog.Errorf("InstantQuery creating NewClient err:%v", err)
		return nil, err
	}
	promClient := h.cc.PromMultiple[clusterName]
	if promClient == "" && address == "" {
		klog.Error("PromMultiple empty.")
		klog.Infof("clusterName:%v\n ql:%v", clusterName, ql)
		return nil, err
	}

	apiV1 := prometheusV1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeWindowsSecond)*time.Second)
	defer cancel()

	result, _, err := apiV1.Query(ctx, ql, time.Now())
	if err != nil {
		klog.Errorf("prometheus Query error:%v url:%+v ql:%v", err, address, ql)
		return nil, err
	}

	vector, has := result.(model.Vector)
	if !has {
		klog.Errorf("prometheus Query result vector error %+v", result)
		return nil, err
	}
	return vector, nil
}
