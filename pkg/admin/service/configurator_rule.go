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

func GetConfigurator(rt core_runtime.Runtime, name string) (*mesh.DynamicConfigResource, error) {
	res := &mesh.DynamicConfigResource{Spec: &mesh_proto.DynamicConfig{}}
	if err := rt.ResourceManager().Get(rt.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.GetByApplication(name), store.GetByKey(name+consts.ConfiguratorRuleSuffix, res_model.DefaultMesh)); err != nil {
		logger.Warnf("get %s configurator failed with error: %s", name, err.Error())
		return nil, err
	}
	return res, nil
}

func UpdateConfigurator(rt core_runtime.Runtime, name string, res *mesh.DynamicConfigResource) error {
	if err := rt.ResourceManager().Update(rt.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.UpdateByApplication(name), store.UpdateByKey(name+consts.ConfiguratorRuleSuffix, res_model.DefaultMesh)); err != nil {
		logger.Warnf("update %s configurator failed with error: %s", name, err.Error())
		return err
	}
	return nil
}

func CreateConfigurator(rt core_runtime.Runtime, name string, res *mesh.DynamicConfigResource) error {
	if err := rt.ResourceManager().Create(rt.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.CreateByApplication(name), store.CreateByKey(name+consts.ConfiguratorRuleSuffix, res_model.DefaultMesh)); err != nil {
		logger.Warnf("create %s configurator failed with error: %s", name, err.Error())
		return err
	}
	return nil
}

func DeleteConfigurator(rt core_runtime.Runtime, name string, res *mesh.DynamicConfigResource) error {
	if err := rt.ResourceManager().Delete(rt.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.DeleteByApplication(name), store.DeleteByKey(name+consts.ConfiguratorRuleSuffix, res_model.DefaultMesh)); err != nil {
		logger.Warnf("delete %s configurator failed with error: %s", name, err.Error())
		return err
	}
	return nil
}
