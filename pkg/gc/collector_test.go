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

package gc_test

import (
	"context"
	"fmt"
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/gc"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/resources/memory"
	"github.com/apache/dubbo-kubernetes/pkg/test/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/test/resources/samples"
	"github.com/apache/dubbo-kubernetes/pkg/util/proto"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
)

var _ = Describe("Dataplane Collector", func() {
	var rm manager.ResourceManager
	createDpAndDpInsight := func(name, mesh string, disconnectTime time.Time) {
		dp := samples.DataplaneBackendBuilder().
			WithName(name).
			WithMesh(mesh).
			Build()
		dpInsight := &core_mesh.DataplaneInsightResource{
			Meta: &model.ResourceMeta{Name: name, Mesh: mesh},
			Spec: &mesh_proto.DataplaneInsight{
				Subscriptions: []*mesh_proto.DiscoverySubscription{
					{
						DisconnectTime: proto.MustTimestampProto(disconnectTime),
					},
				},
			},
		}
		err := rm.Create(context.Background(), dp, store.CreateByKey(name, mesh))
		Expect(err).ToNot(HaveOccurred())
		err = rm.Create(context.Background(), dpInsight, store.CreateByKey(name, mesh))
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		rm = manager.NewResourceManager(memory.NewStore())
		err := rm.Create(context.Background(), core_mesh.NewMeshResource(), store.CreateByKey(core_model.DefaultMesh, core_model.NoMesh))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should cleanup old dataplanes", func() {
		now := time.Now()
		ticks := make(chan time.Time)
		defer close(ticks)
		// given 5 dataplanes now
		for i := 0; i < 5; i++ {
			createDpAndDpInsight(fmt.Sprintf("dp-%d", i), "default", now)
		}
		// given 5 dataplanes after an hour
		for i := 5; i < 10; i++ {
			createDpAndDpInsight(fmt.Sprintf("dp-%d", i), "default", now.Add(time.Hour))
		}

		collector, err := gc.NewCollector(rm, func() *time.Ticker {
			return &time.Ticker{C: ticks}
		}, 1*time.Hour)
		Expect(err).ToNot(HaveOccurred())

		stop := make(chan struct{})
		defer close(stop)
		go func() {
			_ = collector.Start(stop)
		}()

		// Run a first call to gc after 30 mins nothing happens (just disconnected)
		ticks <- now.Add(30 * time.Minute)
		Consistently(func(g Gomega) {
			dataplanes := &core_mesh.DataplaneResourceList{}
			g.Expect(rm.List(context.Background(), dataplanes)).To(Succeed())
			g.Expect(dataplanes.Items).To(HaveLen(10))
		}).Should(Succeed())

		// after 61 then first 5 dataplanes that are offline for more than 1 hour are deleted
		ticks <- now.Add(61 * time.Minute)
		Eventually(func(g Gomega) {
			dataplanes := &core_mesh.DataplaneResourceList{}
			g.Expect(rm.List(context.Background(), dataplanes)).To(Succeed())
			g.Expect(dataplanes.Items).To(HaveLen(5))
			g.Expect(dataplanes).To(WithTransform(func(actual *core_mesh.DataplaneResourceList) []string {
				var names []string
				for _, dp := range actual.Items {
					names = append(names, dp.Meta.GetName())
				}
				return names
			}, Equal([]string{"dp-5", "dp-6", "dp-7", "dp-8", "dp-9"})))
		}).Should(Succeed())
	})
})
