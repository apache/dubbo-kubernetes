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

package model

import "sync"

type Controller interface {
	Run(stop <-chan struct{})
	HasSynced() bool
}

type ServiceHandler func(*Service, *Service, Event)

type ControllerHandlers struct {
	mutex           sync.RWMutex
	serviceHandlers []ServiceHandler
}

type AggregateController interface {
	Controller
}

type Event int

const (
	EventAdd Event = iota
	EventUpdate
	EventDelete
)

func (event Event) String() string {
	out := "unknown"
	switch event {
	case EventAdd:
		out = "add"
	case EventUpdate:
		out = "update"
	case EventDelete:
		out = "delete"
	}
	return out
}

func (c *ControllerHandlers) NotifyServiceHandlers(prev, curr *Service, event Event) {
	for _, f := range c.GetServiceHandlers() {
		f(prev, curr, event)
	}
}

func (c *ControllerHandlers) GetServiceHandlers() []ServiceHandler {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	// Return a shallow copy of the array
	return c.serviceHandlers
}
