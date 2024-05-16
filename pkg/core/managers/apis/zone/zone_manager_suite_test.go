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

package zone_test

import (
	"context"
	"time"
)

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

import (
	"github.com/apache/dubbo-kubernetes/api/system/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/zone"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/resources/memory"
	"github.com/apache/dubbo-kubernetes/pkg/util/proto"
)

var _ = Describe("Zone Manager", func() {
	var validator zone.Validator
	var resStore store.ResourceStore

	BeforeEach(func() {
		resStore = memory.NewStore()
		validator = zone.Validator{Store: resStore}
	})

	It("should not delete zone if it's online", func() {
		// given zone and zoneInsight
		err := resStore.Create(context.Background(), system.NewZoneResource(), store.CreateByKey("zone-1", model.NoMesh))
		Expect(err).ToNot(HaveOccurred())

		err = resStore.Create(context.Background(), &system.ZoneInsightResource{
			Spec: &v1alpha1.ZoneInsight{
				Subscriptions: []*v1alpha1.DDSSubscription{
					{
						ConnectTime: proto.MustTimestampProto(time.Now()),
					},
				},
			},
		}, store.CreateByKey("zone-1", model.NoMesh))
		Expect(err).ToNot(HaveOccurred())
		zoneManager := zone.NewZoneManager(resStore, validator, false)

		zone := system.NewZoneResource()
		err = resStore.Get(context.Background(), zone, store.GetByKey("zone-1", model.NoMesh))
		Expect(err).ToNot(HaveOccurred())

		// when
		err = zoneManager.Delete(context.Background(), zone, store.DeleteByKey("zone-1", model.NoMesh))

		// then
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("zone: unable to delete Zone, Zone CP is still connected, please shut it down first"))
	})

	It("should delete if zone is online when unsafe delete is enabled", func() {
		// given zone and zoneInsight
		err := resStore.Create(context.Background(), system.NewZoneResource(), store.CreateByKey("zone-1", model.NoMesh))
		Expect(err).ToNot(HaveOccurred())

		err = resStore.Create(context.Background(), &system.ZoneInsightResource{
			Spec: &v1alpha1.ZoneInsight{
				Subscriptions: []*v1alpha1.DDSSubscription{
					{
						ConnectTime: proto.MustTimestampProto(time.Now()),
					},
				},
			},
		}, store.CreateByKey("zone-1", model.NoMesh))
		Expect(err).ToNot(HaveOccurred())
		zoneManager := zone.NewZoneManager(resStore, validator, true)

		zone := system.NewZoneResource()
		err = resStore.Get(context.Background(), zone, store.GetByKey("zone-1", model.NoMesh))
		Expect(err).ToNot(HaveOccurred())

		// when
		err = zoneManager.Delete(context.Background(), zone, store.DeleteByKey("zone-1", model.NoMesh))

		// then
		Expect(err).ToNot(HaveOccurred())
	})
})
