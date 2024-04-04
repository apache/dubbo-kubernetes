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

package servicemapping

import (
	"fmt"
)

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

type RegisterRequest struct {
	ConfigsUpdated map[core_model.ResourceReq]map[string]struct{}
}

func (q *RegisterRequest) merge(req *RegisterRequest) *RegisterRequest {
	if q == nil {
		return req
	}
	for key, newApps := range req.ConfigsUpdated {
		if _, ok := q.ConfigsUpdated[key]; !ok {
			q.ConfigsUpdated[key] = make(map[string]struct{})
		}
		for app := range newApps {
			q.ConfigsUpdated[key][app] = struct{}{}
		}
	}
	return q
}

func configsUpdated(req *RegisterRequest) string {
	configs := ""
	for key := range req.ConfigsUpdated {
		configs += key.Name + "." + key.Mesh
		break
	}
	if len(req.ConfigsUpdated) > 1 {
		more := fmt.Sprintf(" and %d more configs", len(req.ConfigsUpdated)-1)
		configs += more
	}
	return configs
}
