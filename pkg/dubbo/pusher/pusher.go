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

package pusher

import (
	"context"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	"reflect"
	"time"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/events"
)

var log = core.Log.WithName("dubbo").WithName("server").WithName("pusher")

const (
	eventsChannelSize  = 10000
	requestChannelSize = 1000
)

type changedEvent struct {
	resourceType core_model.ResourceType
	event        events.Event
}

type pusher struct {
	ctx                 context.Context
	resourceManager     manager.ResourceManager
	eventBus            events.EventBus
	newFullResyncTicker func() *time.Ticker

	resourceTypes                 map[core_model.ResourceType]struct{}
	resourceRevisions             map[core_model.ResourceType]revision
	resourceLastPushed            map[core_model.ResourceType]core_model.ResourceList
	resourceChangedEventListeners map[core_model.ResourceType]events.Listener
	eventsChannel                 chan *changedEvent
	requestChannel                chan struct {
		resourceType core_model.ResourceType
		id           string
	}

	resourceChangedCallbacks *ResourceChangedCallbacks
}

func NewPusher(
	resourceManager manager.ResourceManager,
	eventBus events.EventBus,
	newFullResyncTicker func() *time.Ticker,
	resourceTypes []core_model.ResourceType,
) Pusher {
	p := &pusher{
		resourceManager:               resourceManager,
		eventBus:                      eventBus,
		newFullResyncTicker:           newFullResyncTicker,
		resourceTypes:                 make(map[core_model.ResourceType]struct{}),
		resourceLastPushed:            make(map[core_model.ResourceType]core_model.ResourceList),
		resourceRevisions:             make(map[core_model.ResourceType]revision),
		resourceChangedEventListeners: make(map[core_model.ResourceType]events.Listener),
		eventsChannel:                 make(chan *changedEvent, eventsChannelSize),
		requestChannel: make(chan struct {
			resourceType core_model.ResourceType
			id           string
		}, requestChannelSize),

		resourceChangedCallbacks: NewResourceChangedCallbacks(),
	}

	for _, resourceType := range resourceTypes {
		p.registerResourceType(resourceType)
	}

	return p
}

func (p *pusher) registerResourceType(resourceType core_model.ResourceType) {
	if _, ok := p.resourceTypes[resourceType]; ok {
		return
	}

	p.resourceTypes[resourceType] = struct{}{}
	p.resourceRevisions[resourceType] = 0

	// subscribe Resource Changed Event
	resourceChanged := p.eventBus.Subscribe(func(event events.Event) bool {
		resourceChangedEvent, ok := event.(events.ResourceChangedEvent)
		if ok {
			return resourceChangedEvent.Type == resourceType
		}

		return false
	})
	p.resourceChangedEventListeners[resourceType] = resourceChanged
}

func (p *pusher) receiveResourceChangedEvents(stop <-chan struct{}, resourceType core_model.ResourceType) {
	if _, ok := p.resourceTypes[resourceType]; !ok {
		return
	}

	for {
		select {
		case <-stop:
			p.resourceChangedEventListeners[resourceType].Close()
			return
		case event := <-p.resourceChangedEventListeners[resourceType].Recv():
			p.eventsChannel <- &changedEvent{
				resourceType: resourceType,
				event:        event,
			}
		}
	}
}

func (p *pusher) Start(stop <-chan struct{}) error {
	log.Info("pusher start")

	ctx, cancel := context.WithCancel(context.Background())

	// receive ResourceChanged Events
	for resourceType := range p.resourceTypes {
		log.Info("start receive ResourceChanged Event", "ResourceType", resourceType)
		go p.receiveResourceChangedEvents(stop, resourceType)
	}

	fullResyncTicker := p.newFullResyncTicker()
	defer fullResyncTicker.Stop()

	for {
		select {
		case <-stop:
			log.Info("pusher stopped")

			cancel()
			return nil
		case ce := <-p.eventsChannel:
			log.Info("event received", "ResourceType", ce.resourceType)
			resourceList, err := registry.Global().NewList(ce.resourceType)
			if err != nil {
				log.Info("can not get resourceList")
				continue
			}
			err = p.resourceManager.List(ctx, resourceList)
			if err != nil {
				log.Error(err, "list resource failed", "ResourceType", ce.resourceType)
				continue
			}
			if reflect.DeepEqual(p.resourceLastPushed[ce.resourceType], resourceList) {
				log.Info("resource not changed, nothing to push")
				continue
			}

			p.resourceRevisions[ce.resourceType]++
			p.resourceLastPushed[ce.resourceType] = resourceList

			log.Info("invoke callbacks", "ResourceType", ce.resourceType, "revision", p.resourceRevisions[ce.resourceType])
			// for a ResourceChangedEvent, invoke all callbacks.
			p.resourceChangedCallbacks.InvokeCallbacks(ce.resourceType, PushedItems{
				resourceList: resourceList,
				revision:     p.resourceRevisions[ce.resourceType],
			})
		case req := <-p.requestChannel:
			resourceType := req.resourceType
			id := req.id
			log.Info("received a push request", "ResourceType", resourceType, "id", id)

			cb, ok := p.resourceChangedCallbacks.GetCallBack(resourceType, id)
			if !ok {
				log.Info("not found callback", "ResourceType", resourceType, "id", id)
				continue
			}

			revision := p.resourceRevisions[resourceType]
			lastedPushed := p.resourceLastPushed[resourceType]
			if lastedPushed == nil {
				log.Info("last pushed is nil", "ResourceType", resourceType, "id", id)
				continue
			}

			cb.Invoke(PushedItems{
				resourceList: lastedPushed,
				revision:     revision,
			})
		case <-fullResyncTicker.C:
			log.Info("full resync ticker arrived, starting resync for all types", "ResourceTypes", p.resourceTypes)

			for resourceType := range p.resourceTypes {
				revision := p.resourceRevisions[resourceType]
				lastedPushed := p.resourceLastPushed[resourceType]
				if lastedPushed == nil {
					continue
				}

				// for a ResourceChangedEvent, invoke all callbacks.
				p.resourceChangedCallbacks.InvokeCallbacks(resourceType, PushedItems{
					resourceList: lastedPushed,
					revision:     revision,
				})
			}
		}
	}
}

func (p *pusher) NeedLeaderElection() bool {
	return false
}

func (p *pusher) AddCallback(resourceType core_model.ResourceType, id string, callback ResourceChangedCallbackFn, filters ...ResourceChangedEventFilter) {
	p.resourceChangedCallbacks.AddCallBack(resourceType, id, callback, filters...)
}

func (p *pusher) RemoveCallback(resourceType core_model.ResourceType, id string) {
	p.resourceChangedCallbacks.RemoveCallBack(resourceType, id)
}

func (p *pusher) InvokeCallback(resourceType core_model.ResourceType, id string) {
	p.requestChannel <- struct {
		resourceType core_model.ResourceType
		id           string
	}{resourceType: resourceType, id: id}
}
