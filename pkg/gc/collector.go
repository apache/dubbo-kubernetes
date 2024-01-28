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

package gc

import (
	"context"
	"fmt"
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	"time"
)

var gcLog = core.Log.WithName("dataplane-gc")

type collector struct {
	rm         manager.ResourceManager
	cleanupAge time.Duration
	newTicker  func() *time.Ticker
}

func NewCollector(
	rm manager.ResourceManager,
	newTicker func() *time.Ticker,
	cleanupAge time.Duration,
) (component.Component, error) {
	return &collector{
		rm:         rm,
		cleanupAge: cleanupAge,
		newTicker:  newTicker,
	}, nil
}

func (d *collector) Start(stop <-chan struct{}) error {
	ticker := d.newTicker()
	defer ticker.Stop()
	gcLog.Info("started")
	ctx := context.Background()
	for {
		select {
		case now := <-ticker.C:
			if err := d.cleanup(ctx, now); err != nil {
				gcLog.Error(err, "unable to cleanup")
				continue
			}
		case <-stop:
			gcLog.Info("stopped")
			return nil
		}
	}
}

func (d *collector) cleanup(ctx context.Context, now time.Time) error {
	dataplaneInsights := &core_mesh.DataplaneInsightResourceList{}
	if err := d.rm.List(ctx, dataplaneInsights); err != nil {
		return err
	}
	onDelete := []model.ResourceKey{}
	for _, di := range dataplaneInsights.Items {
		if di.Spec.IsOnline() {
			continue
		}
		if s := di.Spec.GetLastSubscription().(*mesh_proto.DiscoverySubscription); s != nil {
			if err := s.GetDisconnectTime().CheckValid(); err != nil {
				gcLog.Error(err, "unable to parse DisconnectTime", "disconnect time", s.GetDisconnectTime(), "mesh", di.GetMeta().GetMesh(), "dataplane", di.GetMeta().GetName())
				continue
			}
			if now.Sub(s.GetDisconnectTime().AsTime()) > d.cleanupAge {
				onDelete = append(onDelete, model.ResourceKey{Name: di.GetMeta().GetName(), Mesh: di.GetMeta().GetMesh()})
			}
		}
	}
	for _, rk := range onDelete {
		gcLog.Info(fmt.Sprintf("deleting dataplane which is offline for %v", d.cleanupAge), "name", rk.Name, "mesh", rk.Mesh)
		if err := d.rm.Delete(ctx, core_mesh.NewDataplaneResource(), store.DeleteBy(rk)); err != nil {
			gcLog.Error(err, "unable to delete dataplane", "name", rk.Name, "mesh", rk.Mesh)
			continue
		}
	}
	return nil
}

func (d *collector) NeedLeaderElection() bool {
	return true
}
