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

package test

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"strings"
)

var testServerLog = core.Log.WithName("test")

func Setup(rt core_runtime.Runtime) error {
	testServer := NewTestServer(rt)
	if err := rt.Add(testServer); err != nil {
		testServerLog.Error(err, "fail to start the test server")
	}
	return nil
}

type TestServer struct {
	rt core_runtime.Runtime
}

func NewTestServer(rt core_runtime.Runtime) *TestServer {
	return &TestServer{rt: rt}
}

func (t *TestServer) Start(stop <-chan struct{}) error {
	// 测试dataplane资源
	if err := testDataplane(t.rt); err != nil {
		return err
	}
	// 测试mapping资源
	if err := testMapping(t.rt); err != nil {
		return err
	}
	// 测试metadata资源
	if err := testMetadata(t.rt); err != nil {
		return err
	}

	return nil
}

func (a *TestServer) NeedLeaderElection() bool {
	return false
}

// dataplane资源只有get, list接口, 其余均不支持
func testDataplane(rt core_runtime.Runtime) error {
	manager := rt.ResourceManager()
	dataplaneResource := mesh.NewDataplaneResource()

	// list
	dataplaneList := &mesh.DataplaneResourceList{}
	if err := manager.List(rt.AppContext(), dataplaneList); err != nil {
		return err
	}

	if len(dataplaneList.Items) > 0 {
		// get
		if err := manager.Get(rt.AppContext(), dataplaneResource,
			store.GetBy(core_model.ResourceKey{
				Name: dataplaneList.Items[0].Meta.GetName(),
			})); err != nil {
			return err
		}
	}

	return nil
}

// metadata资源没有删除能力
func testMetadata(rt core_runtime.Runtime) error {
	manager := rt.ResourceManager()
	// create
	metadata2 := mesh.NewMetaDataResource()
	err := metadata2.SetSpec(&mesh_proto.MetaData{
		App:      "dubbo-springboot-demo-lixinyang",
		Revision: "bdc0958191bba7a0f050a32709ee1111",
		Services: map[string]*mesh_proto.ServiceInfo{
			"org.apache.dubbo.springboot.demo.DemoService:tri": {
				Name: "org.apache.dubbo.springboot.demo.DemoService",
			},
		},
	})
	if err != nil {
		return err
	}
	if err := manager.Create(rt.AppContext(), metadata2, store.CreateBy(core_model.ResourceKey{
		Name: metadata2.Spec.App + "-" + metadata2.Spec.Revision,
	})); err != nil {
		return err
	}

	metadata1 := mesh.NewMetaDataResource()
	// get
	if err := manager.Get(rt.AppContext(), metadata1, store.GetBy(core_model.ResourceKey{
		Name: metadata2.Spec.App + "-" + metadata2.Spec.Revision,
	})); err != nil {
		return err
	}

	// list
	metadataList := &mesh.MetaDataResourceList{}

	if err := manager.List(rt.AppContext(), metadataList); err != nil {
		return err
	}

	// update
	metadata3 := mesh.NewMetaDataResource()
	metadata3.SetMeta(metadata1.GetMeta())
	err = metadata3.SetSpec(&mesh_proto.MetaData{
		App:      "dubbo-springboot-demo-lixinyang",
		Revision: "bdc0958191bba7a0f050a32709ee1111",
		Services: map[string]*mesh_proto.ServiceInfo{
			"org.apache.dubbo.springboot.demo.DemoService:tri": {
				Name: "org.apache.dubbo.springboot.demo.lixinyang",
			},
		},
	})
	if err != nil {
		return err
	}
	if err := manager.Update(rt.AppContext(), metadata3); err != nil {
		return err
	}
	return nil
}

// mapping资源没有删除功能
func testMapping(rt core_runtime.Runtime) error {
	manager := rt.ResourceManager()

	mapping2 := mesh.NewMappingResource()
	err := mapping2.SetSpec(&mesh_proto.Mapping{
		Zone:          "zone1",
		InterfaceName: "org.apache.dubbo.springboot.demo.DemoService1",
		ApplicationNames: []string{
			"dubbo-springboot-demo-provider",
		},
	})
	if err != nil {
		return err
	}

	// create
	if err := manager.Create(rt.AppContext(), mapping2, store.CreateBy(core_model.ResourceKey{
		Name: strings.ToLower(strings.ReplaceAll(mapping2.Spec.InterfaceName, ".", "-")),
	})); err != nil {
		return err
	}

	// mapping test
	mapping1 := mesh.NewMappingResource()
	// get
	if err := manager.Get(rt.AppContext(), mapping1, store.GetBy(core_model.ResourceKey{
		Name: strings.ToLower(strings.ReplaceAll("org.apache.dubbo.springboot.demo.DemoService1", ".", "-")),
	})); err != nil {
		return err
	}

	mappingList := &mesh.MappingResourceList{}

	// list
	if err := manager.List(rt.AppContext(), mappingList); err != nil {
		return err
	}

	mapping3 := mesh.NewMappingResource()
	mapping3.SetMeta(mapping1.GetMeta())
	err = mapping3.SetSpec(&mesh_proto.Mapping{
		Zone:          "zone2",
		InterfaceName: "org.apache.dubbo.springboot.demo.DemoService1",
		ApplicationNames: []string{
			"dubbo-springboot-demo-provider2",
		},
	})
	if err != nil {
		return err
	}

	// update
	if err := manager.Update(rt.AppContext(), mapping3); err != nil {
		return err
	}
	return nil
}
