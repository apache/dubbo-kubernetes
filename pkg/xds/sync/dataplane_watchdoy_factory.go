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

package sync

import (
	"context"
	"time"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	util_watchdog "github.com/apache/dubbo-kubernetes/pkg/util/watchdog"
)

var xdsServerLog = core.Log.WithName("xds-server")

type dataplaneWatchdogFactory struct {
	refreshInternal time.Duration

	deps DataplaneWatchdogDependencies
}

func NewDataplaneWatchdogFactory(
	refreshInternal time.Duration,
	deps DataplaneWatchdogDependencies,
) (DataplaneWatchdogFactory, error) {
	return &dataplaneWatchdogFactory{
		refreshInternal: refreshInternal,
		deps:            deps,
	}, nil
}

func (d *dataplaneWatchdogFactory) New(dpKey model.ResourceKey) util_watchdog.Watchdog {
	log := xdsServerLog.WithName("dataplane-sync-watchdog").WithValues("dataplaneKey", dpKey)
	dataplaneWatchdog := NewDataplaneWatchdog(d.deps, dpKey)
	return &util_watchdog.SimpleWatchdog{
		NewTicker: func() *time.Ticker {
			return time.NewTicker(d.refreshInternal)
		},
		OnTick: func(ctx context.Context) error {
			_, err := dataplaneWatchdog.Sync(ctx)
			if err != nil {
				return err
			}
			return nil
		},
		OnError: func(err error) {
			log.Error(err, "OnTick() failed")
		},
		OnStop: func() {
			if err := dataplaneWatchdog.Cleanup(); err != nil {
				log.Error(err, "OnTick() failed")
			}
		},
	}
}
