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

package manager_test

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/resources/memory"
	"github.com/apache/dubbo-kubernetes/pkg/test"
	. "github.com/apache/dubbo-kubernetes/pkg/test/matchers"
)

type countingResourcesManager struct {
	store       core_store.ResourceStore
	getQueries  uint32
	listQueries uint32
}

func (c *countingResourcesManager) Get(ctx context.Context, res core_model.Resource, fn ...core_store.GetOptionsFunc) error {
	atomic.AddUint32(&c.getQueries, 1)
	return c.store.Get(ctx, res, fn...)
}

func (c *countingResourcesManager) List(ctx context.Context, list core_model.ResourceList, fn ...core_store.ListOptionsFunc) error {
	opts := core_store.NewListOptions(fn...)
	if list.GetItemType() == core_mesh.DataplaneType && opts.Mesh == "slow" {
		time.Sleep(10 * time.Second)
	}
	atomic.AddUint32(&c.listQueries, 1)
	return c.store.List(ctx, list, fn...)
}

var _ core_manager.ReadOnlyResourceManager = &countingResourcesManager{}

var _ = Describe("Cached Resource Manager", func() {
	var store core_store.ResourceStore
	var cachedManager core_manager.ReadOnlyResourceManager
	var countingManager *countingResourcesManager
	var res *core_mesh.DataplaneResource
	expiration := 500 * time.Millisecond

	BeforeEach(func() {
		// given
		var err error
		store = memory.NewStore()
		countingManager = &countingResourcesManager{
			store: store,
		}
		Expect(err).ToNot(HaveOccurred())
		cachedManager, err = core_manager.NewCachedManager(countingManager, expiration)
		Expect(err).ToNot(HaveOccurred())

		// and created resources
		res = &core_mesh.DataplaneResource{
			Spec: &mesh_proto.Dataplane{
				Networking: &mesh_proto.Dataplane_Networking{
					Address: "127.0.0.1",
					Inbound: []*mesh_proto.Dataplane_Networking_Inbound{
						{
							Port:        80,
							ServicePort: 8080,
						},
					},
				},
			},
		}
		err = store.Create(context.Background(), res, core_store.CreateByKey("dp-1", "default"))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should cache Get() queries", func() {
		// when fetched resources multiple times
		fetch := func() *core_mesh.DataplaneResource {
			fetched := core_mesh.NewDataplaneResource()
			err := cachedManager.Get(context.Background(), fetched, core_store.GetByKey("dp-1", "default"))
			Expect(err).ToNot(HaveOccurred())
			return fetched
		}

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				fetch()
				wg.Done()
			}()
		}
		wg.Wait()

		// then real manager should be called only once
		Expect(fetch().Spec).To(MatchProto(res.Spec))
		Expect(int(countingManager.getQueries)).To(Equal(1))

		// when
		time.Sleep(expiration)

		// then
		Expect(fetch().Spec).To(MatchProto(res.Spec))
		Expect(int(countingManager.getQueries)).To(Equal(2))
	})

	It("should not cache Get() not found", func() {
		// when fetched resources multiple times
		fetch := func() {
			_ = cachedManager.Get(context.Background(), core_mesh.NewDataplaneResource(), core_store.GetByKey("non-existing", "default"))
		}

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				fetch()
				wg.Done()
			}()
		}
		wg.Wait()

		// then real manager should be called every time
		Expect(int(countingManager.getQueries)).To(Equal(100))
	})

	It("should cache List() queries", func() {
		// when fetched resources multiple times
		fetch := func() core_mesh.DataplaneResourceList {
			fetched := core_mesh.DataplaneResourceList{}
			err := cachedManager.List(context.Background(), &fetched, core_store.ListByMesh("default"))
			Expect(err).ToNot(HaveOccurred())
			return fetched
		}

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				fetch()
				wg.Done()
			}()
		}
		wg.Wait()

		// then real manager should be called only once
		list := fetch()
		Expect(list.Items).To(HaveLen(1))
		Expect(list.Items[0].GetSpec()).To(MatchProto(res.Spec))
		Expect(int(countingManager.listQueries)).To(Equal(1))

		// when
		time.Sleep(expiration)

		// then
		list = fetch()
		Expect(list.Items).To(HaveLen(1))
		Expect(list.Items[0].GetSpec()).To(MatchProto(res.Spec))
		Expect(int(countingManager.listQueries)).To(Equal(2))
	})

	It("should let concurrent List() queries for different types and meshes", test.Within(15*time.Second, func() {
		// given ongoing TrafficLog from mesh slow that takes a lot of time to complete
		done := make(chan struct{})
		go func() {
			fetched := core_mesh.DataplaneResourceList{}
			err := cachedManager.List(context.Background(), &fetched, core_store.ListByMesh("slow"))
			Expect(err).ToNot(HaveOccurred())
			close(done)
		}()

		// when trying to fetch TrafficLog from different mesh that takes normal time to response
		fetched := core_mesh.DataplaneResourceList{}
		err := cachedManager.List(context.Background(), &fetched, core_store.ListByMesh("default"))

		// then first request does not block request for other mesh
		Expect(err).ToNot(HaveOccurred())

		// when trying to fetch different resource type
		fetchedTp := core_mesh.ZoneIngressInsightResourceList{}
		err = cachedManager.List(context.Background(), &fetchedTp, core_store.ListByMesh("default"))

		// then first request does not block request for other type
		Expect(err).ToNot(HaveOccurred())
		<-done
	}))

	It("should cache List() at different key when ordered", test.Within(5*time.Second, func() {
		// when fetched resources multiple times
		fetch := func(ordered bool) core_mesh.DataplaneResourceList {
			fetched := core_mesh.DataplaneResourceList{}
			var err error
			if ordered {
				err = cachedManager.List(context.Background(), &fetched, core_store.ListOrdered(), core_store.ListByMesh("default"))
			} else {
				err = cachedManager.List(context.Background(), &fetched, core_store.ListByMesh("default"))
			}
			Expect(err).ToNot(HaveOccurred())
			return fetched
		}

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				fetch(false)
				wg.Done()
			}()
		}
		wg.Wait()

		// then real manager should be called only once
		list := fetch(false)
		Expect(list.Items).To(HaveLen(1))
		Expect(list.Items[0].GetSpec()).To(MatchProto(res.Spec))
		Expect(int(countingManager.listQueries)).To(Equal(1))

		// when call for ordered data
		list = fetch(true)

		// then real manager should be called
		Expect(list.Items).To(HaveLen(1))
		Expect(list.Items[0].GetSpec()).To(MatchProto(res.Spec))
		Expect(int(countingManager.listQueries)).To(Equal(2))
	}))
})
