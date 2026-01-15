//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package status

import (
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/features"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
)

var scope = dubbolog.RegisterScope("status", "status controller")

type Manager struct {
	store   model.ConfigStore
	workers WorkerQueue
}

func NewManager(store model.ConfigStore) *Manager {
	writeFunc := func(m *config.Config) {
		scope.Debugf("writing status for resource %s/%s", m.Namespace, m.Name)
		if store == nil {
			scope.Warnf("store is nil, cannot write status for %s/%s", m.Namespace, m.Name)
			return
		}
		_, err := store.UpdateStatus(*m)
		if err != nil {
			if kerrors.IsConflict(err) {
				scope.Debugf("warning: object has changed %s/%s: %v", m.Namespace, m.Name, err)
			} else {
				scope.Errorf("Encountered unexpected error updating status for %v, will try again later: %s", m, err)
			}
			return
		}
	}
	retrieveFunc := func(resource Resource) *config.Config {
		scope.Debugf("retrieving config for status update: %s/%s", resource.Namespace, resource.Name)
		if store == nil {
			scope.Warnf("store is nil, cannot retrieve config for %s/%s", resource.Namespace, resource.Name)
			return nil
		}
		k, ok := gvk.FromGVR(resource.GroupVersionResource)
		if !ok {
			scope.Warnf("GVR %v could not be identified", resource.GroupVersionResource)
			return nil
		}

		current := store.Get(k, resource.Name, resource.Namespace)
		if current == nil {
			scope.Debugf("no current config found for %s/%s", resource.Namespace, resource.Name)
		}
		return current
	}
	return &Manager{
		store:   store,
		workers: NewWorkerPool(writeFunc, retrieveFunc, uint(features.StatusMaxWorkers)),
	}
}

func (m *Manager) Start(stop <-chan struct{}) {
	scope.Info("Starting status manager")

	ctx := NewDubboContext(stop)
	m.workers.Run(ctx)
}

func (m *Manager) CreateGenericController(fn UpdateFunc) *Controller {
	result := &Controller{
		fn:      fn,
		workers: m.workers,
	}
	return result
}

type UpdateFunc func(status Manipulator, context any)

type Queue interface {
	EnqueueStatusUpdateResource(context any, target Resource)
}

type Controller struct {
	fn      UpdateFunc
	workers WorkerQueue
}

func (c *Controller) EnqueueStatusUpdateResource(context any, target Resource) {
	c.workers.Push(target, c, context)
}

func (c *Controller) Delete(r Resource) {
	c.workers.Delete(r)
}
