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

package mux

import (
	"context"
	"github.com/apache/dubbo-kubernetes/pkg/config/multizone"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/dds/service"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	dubbo_log "github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/go-logr/logr"
	"time"
)

type zone struct {
	zone string
}

type ZoneWatch struct {
	log        logr.Logger
	poll       time.Duration
	timeout    time.Duration
	bus        events.EventBus
	extensions context.Context
	rm         manager.ReadOnlyResourceManager
	zones      map[zone]time.Time
}

func NewZoneWatch(
	log logr.Logger,
	cfg multizone.ZoneHealthCheckConfig,
	bus events.EventBus,
	rm manager.ReadOnlyResourceManager,
	extensions context.Context,
) (*ZoneWatch, error) {
	return &ZoneWatch{
		log:        log,
		poll:       cfg.PollInterval.Duration,
		timeout:    cfg.Timeout.Duration,
		bus:        bus,
		extensions: extensions,
		rm:         rm,
		zones:      map[zone]time.Time{},
	}, nil
}

func (zw *ZoneWatch) Start(stop <-chan struct{}) error {
	timer := time.NewTicker(zw.poll)
	defer timer.Stop()

	connectionWatch := zw.bus.Subscribe(func(e events.Event) bool {
		_, ok := e.(service.ZoneOpenedStream)
		return ok
	})
	defer connectionWatch.Close()

	for {
		select {
		case <-timer.C:
			for zone, lastStreamOpened := range zw.zones {
				ctx := context.Background()
				zoneInsight := system.NewZoneInsightResource()

				log := dubbo_log.AddFieldsFromCtx(zw.log, ctx, zw.extensions)
				if err := zw.rm.Get(ctx, zoneInsight, store.GetByKey(zone.zone, model.NoMesh)); err != nil {
					if store.IsResourceNotFound(err) {
						zw.bus.Send(service.ZoneWentOffline{
							Zone: zone.zone,
						})
						delete(zw.zones, zone)
					} else {
						log.Info("error getting ZoneInsight", "zone", zone.zone, "error", err)
					}
					continue
				}

				// It may be that we don't have a health check yet so we use the
				// lastSeen time because we know the zone was connected at that
				// point at least
				lastHealthCheck := zoneInsight.Spec.GetHealthCheck().GetTime().AsTime()
				if lastStreamOpened.After(lastHealthCheck) {
					lastHealthCheck = lastStreamOpened
				}
				if time.Since(lastHealthCheck) > zw.timeout {
					zw.bus.Send(service.ZoneWentOffline{
						Zone: zone.zone,
					})
					delete(zw.zones, zone)
				}
			}
		case e := <-connectionWatch.Recv():
			newStream := e.(service.ZoneOpenedStream)

			// We keep a record of the time we open a stream.
			// This is to prevent the zone from timing out on a poll
			// where the last health check is still from a previous connect, so:
			// a long time ago: zone CP disconnects, no more health checks are sent
			// now:
			//  zone CP opens streams
			//  global CP gets ZoneOpenedStream (but we don't stash the time as below)
			//  global CP runs poll and see the last health check from "a long time ago"
			//  BAD: global CP kills streams
			//  zone CP health check arrives
			zw.zones[zone{
				zone: newStream.Zone,
			}] = core.Now()
		case <-stop:
			return nil
		}
	}
}

func (zw *ZoneWatch) NeedLeaderElection() bool {
	return false
}
