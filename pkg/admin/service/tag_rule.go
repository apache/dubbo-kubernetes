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
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	res_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func GetTagRule(rt core_runtime.Runtime, name string) (*mesh.TagRouteResource, error) {
	res := &mesh.TagRouteResource{Spec: &mesh_proto.TagRoute{}}
	err := rt.ResourceManager().Get(rt.AppContext(), res,
		// here `name` may be service name or app name, set *ByApplication(`name`) is ok.
		store.GetByApplication(name), store.GetByKey(name+consts.TagRuleSuffix, res_model.DefaultMesh))
	if err != nil {
		logger.Warnf("get tag rule %s error: %v", name, err)
		return nil, err
	}
	return res, nil
}

func UpdateTagRule(rt core_runtime.Runtime, name string, res *mesh.TagRouteResource) error {
	err := rt.ResourceManager().Update(rt.AppContext(), res,
		// here `name` may be service name or app name, set *ByApplication(`name`) is ok.
		store.UpdateByApplication(name), store.UpdateByKey(name+consts.TagRuleSuffix, res_model.DefaultMesh))
	if err != nil {
		logger.Warnf("update tag rule %s error: %v", name, err)
		return err
	}
	return nil
}

func CreateTagRule(rt core_runtime.Runtime, name string, res *mesh.TagRouteResource) error {
	err := rt.ResourceManager().Create(rt.AppContext(), res,
		// here `name` may be service name or app name, set *ByApplication(`name`) is ok.
		store.CreateByApplication(name), store.CreateByKey(name+consts.TagRuleSuffix, res_model.DefaultMesh))
	if err != nil {
		logger.Warnf("create tag rule %s error: %v", name, err)
		return err
	}
	return nil
}

func DeleteTagRule(rt core_runtime.Runtime, name string, res *mesh.TagRouteResource) error {
	err := rt.ResourceManager().Delete(rt.AppContext(), res,
		// here `name` may be service name or app name, set *ByApplication(`name`) is ok.
		store.DeleteByApplication(name), store.DeleteByKey(name+consts.TagRuleSuffix, res_model.DefaultMesh))
	if err != nil {
		logger.Warnf("delete tag rule %s error: %v", name, err)
		return err
	}
	return nil
}
