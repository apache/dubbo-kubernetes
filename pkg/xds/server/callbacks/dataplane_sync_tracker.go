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

package callbacks

import (
	"context"
	stdsync "sync"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	util_watchdog "github.com/apache/dubbo-kubernetes/pkg/util/watchdog"
)

var dataplaneSyncTrackerLog = core.Log.WithName("xds-server").WithName("dataplane-sync-tracker")

type NewDataplaneWatchdogFunc func(key core_model.ResourceKey) util_watchdog.Watchdog

func NewDataplaneSyncTracker(factoryFunc NewDataplaneWatchdogFunc) DataplaneCallbacks {
	return &dataplaneSyncTracker{
		newDataplaneWatchdog: factoryFunc,
		watchdogs:            map[core_model.ResourceKey]context.CancelFunc{},
	}
}

var _ DataplaneCallbacks = &dataplaneSyncTracker{}

// dataplaneSyncTracker tracks XDS streams that are connected to the CP and fire up a watchdog.
// Watchdog should be run only once for given dataplane regardless of the number of streams.
// For ADS there is only one stream for DP.
//
// Node info can be (but does not have to be) carried only on the first XDS request. That's why need streamsAssociation map
// that indicates that the stream was already associated
type dataplaneSyncTracker struct {
	NoopDataplaneCallbacks

	newDataplaneWatchdog NewDataplaneWatchdogFunc

	stdsync.RWMutex // protects access to the fields below
	watchdogs       map[core_model.ResourceKey]context.CancelFunc
}

func (t *dataplaneSyncTracker) OnProxyConnected(streamID core_xds.StreamID, dpKey core_model.ResourceKey, _ context.Context, _ core_xds.DataplaneMetadata) error {
	// We use OnProxyConnected because there should be only one watchdog for given dataplane.
	t.Lock()
	defer t.Unlock()

	stopCh := make(chan struct{})

	t.watchdogs[dpKey] = func() {
		dataplaneSyncTrackerLog.V(1).Info("stopping Watchdog for a Dataplane", "dpKey", dpKey, "streamID", streamID)
		close(stopCh)
	}
	dataplaneSyncTrackerLog.V(1).Info("starting Watchdog for a Dataplane", "dpKey", dpKey, "streamID", streamID)
	go t.newDataplaneWatchdog(dpKey).Start(stopCh)
	return nil
}

func (t *dataplaneSyncTracker) OnProxyDisconnected(_ context.Context, _ core_xds.StreamID, dpKey core_model.ResourceKey) {
	t.Lock()
	defer t.Unlock()
	if cancelFn := t.watchdogs[dpKey]; cancelFn != nil {
		cancelFn()
	}
	delete(t.watchdogs, dpKey)
}
