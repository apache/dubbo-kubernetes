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

package catalog

import (
	"context"
	"time"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
)

var writerLog = core.Log.WithName("intercp").WithName("catalog").WithName("writer")

type catalogWriter struct {
	catalog    Catalog
	heartbeats *Heartbeats
	instance   Instance
	interval   time.Duration
}

var _ component.Component = &catalogWriter{}

func NewWriter(
	catalog Catalog,
	heartbeats *Heartbeats,
	instance Instance,
	interval time.Duration,
) (component.Component, error) {
	leaderInstance := instance
	leaderInstance.Leader = true
	return &catalogWriter{
		catalog:    catalog,
		heartbeats: heartbeats,
		instance:   leaderInstance,
		interval:   interval,
	}, nil
}

func (r *catalogWriter) Start(stop <-chan struct{}) error {
	heartbeatLog.Info("starting catalog writer")
	ctx := context.Background()
	writerLog.Info("replacing a leader in the catalog")
	if err := r.catalog.ReplaceLeader(ctx, r.instance); err != nil {
		writerLog.Error(err, "could not replace leader") // continue, it will be replaced in ticker anyways
	}
	ticker := time.NewTicker(r.interval)
	for {
		select {
		case <-ticker.C:
			instances := r.heartbeats.ResetAndCollect()
			instances = append(instances, r.instance)
			updated, err := r.catalog.Replace(ctx, instances)
			if err != nil {
				writerLog.Error(err, "could not update catalog")
				continue
			}
			if updated {
				writerLog.Info("instances catalog updated", "instances", instances)
			} else {
				writerLog.V(1).Info("no need to update instances, because the catalog is the same", "instances", instances)
			}
		case <-stop:
			return nil
		}
	}
}

func (r *catalogWriter) NeedLeaderElection() bool {
	return true
}
