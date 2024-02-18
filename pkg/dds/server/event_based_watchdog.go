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

package server

import (
	"context"
	"errors"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/dds/reconcile"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	util_maps "github.com/apache/dubbo-kubernetes/pkg/util/maps"
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/go-logr/logr"
	"golang.org/x/exp/maps"
	"strings"
	"time"
)

type EventBasedWatchdog struct {
	Ctx                 context.Context
	Node                *envoy_core.Node
	EventBus            events.EventBus
	Reconciler          reconcile.Reconciler
	ProvidedTypes       map[model.ResourceType]struct{}
	Log                 logr.Logger
	NewFlushTicker      func() *time.Ticker
	NewFullResyncTicker func() *time.Ticker
}

func (e *EventBasedWatchdog) Start(stop <-chan struct{}) {
	listener := e.EventBus.Subscribe(func(event events.Event) bool {
		resChange, ok := event.(events.ResourceChangedEvent)
		if !ok {
			return false
		}
		if _, ok := e.ProvidedTypes[resChange.Type]; !ok {
			return false
		}
		return true
	})
	flushTicker := e.NewFlushTicker()
	defer flushTicker.Stop()
	fullResyncTicker := e.NewFullResyncTicker()
	defer fullResyncTicker.Stop()

	// for the first reconcile assign all types
	changedTypes := maps.Clone(e.ProvidedTypes)
	reasons := map[string]struct{}{
		ReasonResync: {},
	}

	for {
		select {
		case <-stop:
			if err := e.Reconciler.Clear(e.Ctx, e.Node); err != nil {
				e.Log.Error(err, "reconcile clear failed")
			}
			listener.Close()
			return
		case <-flushTicker.C:
			if len(changedTypes) == 0 {
				continue
			}
			reason := strings.Join(util_maps.SortedKeys(reasons), "_and_")
			e.Log.V(1).Info("reconcile", "changedTypes", changedTypes, "reason", reason)
			err, _ := e.Reconciler.Reconcile(e.Ctx, e.Node, changedTypes, e.Log)
			if err != nil && errors.Is(err, context.Canceled) {
				e.Log.Error(err, "reconcile failed", "changedTypes", changedTypes, "reason", reason)
			} else {
				changedTypes = map[model.ResourceType]struct{}{}
				reasons = map[string]struct{}{}
			}
		case <-fullResyncTicker.C:
			e.Log.V(1).Info("schedule full resync")
			changedTypes = maps.Clone(e.ProvidedTypes)
			reasons[ReasonResync] = struct{}{}
		case event := <-listener.Recv():
			resChange := event.(events.ResourceChangedEvent)
			e.Log.V(1).Info("schedule sync for type", "typ", resChange.Type)
			changedTypes[resChange.Type] = struct{}{}
			reasons[ReasonEvent] = struct{}{}
		}
	}
}
