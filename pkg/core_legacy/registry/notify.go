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

package registry

import (
	"context"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/events"
)

const (
	keySeparator = "-"
)

type AdminInstanceEvent struct {
	Action   remoting.EventType
	Instance registry.ServiceInstance
	Key      string // store the key for Service.Key()
}

type NotifyListener struct {
	manager.ResourceManager
	dataplaneCache *sync.Map
	eventWriter    events.Emitter
	ctx            *ApplicationContext
}

func NewNotifyListener(
	manager manager.ResourceManager,
	cache *sync.Map,
	writer events.Emitter,
	grc *ApplicationContext,
) *NotifyListener {
	return &NotifyListener{
		manager,
		cache,
		writer,
		grc,
	}
}

func (l *NotifyListener) Notify(event *registry.ServiceEvent) {}

func (l *NotifyListener) NotifyAll(events []*registry.ServiceEvent, f func()) {}

func (l *NotifyListener) NotifyInstance(event *AdminInstanceEvent) {
	switch event.Action {
	case remoting.EventTypeAdd, remoting.EventTypeUpdate:
		if err := l.createOrUpdateDataplane(context.Background(), event.Key, event.Instance); err != nil {
			return
		}
	case remoting.EventTypeDel:
		if err := l.deleteDataplane(context.Background(), event.Key, event.Instance); err != nil {
			return
		}
	}
}

func (l *NotifyListener) NotifyAllInstances(events []*AdminInstanceEvent, f func()) {
	for _, event := range events {
		l.NotifyInstance(event)
	}
}

func (l *NotifyListener) deleteDataplane(ctx context.Context, app string, instance registry.ServiceInstance) error {
	address := instance.GetAddress()
	var revision string
	if v, ok := l.ctx.allInstances.Load(app); ok {
		instances := v.([]registry.ServiceInstance)
		for _, instance := range instances {
			if instance.GetAddress() == address {
				revision = instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName]
			}
		}
	}
	key := getDataplaneKey(app, revision)

	l.dataplaneCache.Delete(key)
	if l.eventWriter != nil {
		go func() {
			l.eventWriter.Send(events.ResourceChangedEvent{
				Operation: events.Delete,
				Type:      mesh.DataplaneType,
				Key: core_model.ResourceKey{
					Name: key,
				},
			})
		}()
	}
	return nil
}

func (l *NotifyListener) createOrUpdateDataplane(ctx context.Context, app string, instance registry.ServiceInstance) error {
	address := instance.GetAddress()
	var revision string
	if v, ok := l.ctx.allInstances.Load(app); ok {
		instances := v.([]registry.ServiceInstance)
		for _, instance := range instances {
			if instance.GetAddress() == address {
				revision = instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName]
				break
			}
		}
	}
	key := getDataplaneKey(app, address)

	dataplaneResource := mesh.NewDataplaneResource()
	dataplaneResource.SetMeta(&resourceMetaObject{
		Name: key,
		Mesh: core_model.DefaultMesh,
	})
	dataplaneResource.Spec.Networking = &mesh_proto.Dataplane_Networking{}
	dataplaneResource.Spec.Extensions = map[string]string{}
	dataplaneResource.Spec.Extensions[mesh_proto.ApplicationName] = app
	dataplaneResource.Spec.Extensions[mesh_proto.Revision] = revision
	dataplaneResource.Spec.Networking.Address = address
	ifaces, err := InboundInterfacesFor(ctx, instance)
	if err != nil {
		return err
	}
	ofaces, err := OutboundInterfacesFor(ctx, instance)
	if err != nil {
		return err
	}
	dataplaneResource.Spec.Networking.Inbound = ifaces
	dataplaneResource.Spec.Networking.Outbound = ofaces
	l.dataplaneCache.Store(key, dataplaneResource)

	if l.eventWriter != nil {
		go func() {
			l.eventWriter.Send(events.ResourceChangedEvent{
				Operation: events.Update,
				Type:      mesh.DataplaneType,
				Key:       core_model.MetaToResourceKey(dataplaneResource.GetMeta()),
			})
		}()
	}
	return nil
}

func InboundInterfacesFor(ctx context.Context, instance registry.ServiceInstance) ([]*mesh_proto.Dataplane_Networking_Inbound, error) {
	var ifaces []*mesh_proto.Dataplane_Networking_Inbound
	ifaces = append(ifaces, &mesh_proto.Dataplane_Networking_Inbound{
		Port: uint32(instance.GetPort()),
	})
	return ifaces, nil
}

func OutboundInterfacesFor(ctx context.Context, instance registry.ServiceInstance) ([]*mesh_proto.Dataplane_Networking_Outbound, error) {
	var outbounds []*mesh_proto.Dataplane_Networking_Outbound

	return outbounds, nil
}

func getDataplaneKey(app string, addr string) string {
	return app + keySeparator + addr
}
