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
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
)

const (
	keySeparator = "-"
)

type NotifyListener struct {
	manager.ResourceManager
	dataplaneCache *sync.Map
}

func NewNotifyListener(manager manager.ResourceManager, cache *sync.Map) *NotifyListener {
	return &NotifyListener{
		manager,
		cache,
	}
}

func (l *NotifyListener) Notify(event *dubboRegistry.ServiceEvent) {
	switch event.Action {
	case remoting.EventTypeAdd, remoting.EventTypeUpdate:
		if err := l.createOrUpdateDataplane(context.Background(), event.Service); err != nil {
			return
		}
	case remoting.EventTypeDel:
		if err := l.deleteDataplane(context.Background(), event.Service); err != nil {
			return
		}
	}
}

func (l *NotifyListener) NotifyAll(events []*dubboRegistry.ServiceEvent, f func()) {
	for _, event := range events {
		l.Notify(event)
	}
}

func (l *NotifyListener) deleteDataplane(ctx context.Context, url *common.URL) error {
	app := url.GetParam(constant.ApplicationKey, "")
	// 根据该url的特征计算一个哈希值
	candidates := make([]string, 0, 8)
	iface := url.GetParam(constant.InterfaceKey, "")
	ms := url.Methods
	if len(ms) == 0 {
		candidates = append(candidates, iface)
	} else {
		for _, m := range ms {
			candidates = append(candidates, iface+constant.KeySeparator+m)
		}
	}
	sort.Strings(candidates)

	// it's nearly impossible to be overflow
	res := uint64(0)
	for _, c := range candidates {
		res += uint64(crc32.ChecksumIEEE([]byte(c)))
	}
	hash := fmt.Sprint(res)
	key := getDataplaneKey(app, hash)

	l.dataplaneCache.Delete(key)
	return nil
}

func (l *NotifyListener) createOrUpdateDataplane(ctx context.Context, url *common.URL) error {
	app := url.GetParam(constant.ApplicationKey, "")
	// 根据该url的特征计算一个哈希值
	candidates := make([]string, 0, 8)
	iface := url.GetParam(constant.InterfaceKey, "")
	ms := url.Methods
	if len(ms) == 0 {
		candidates = append(candidates, iface)
	} else {
		for _, m := range ms {
			candidates = append(candidates, iface+constant.KeySeparator+m)
		}
	}
	sort.Strings(candidates)

	// it's nearly impossible to be overflow
	res := uint64(0)
	for _, c := range candidates {
		res += uint64(crc32.ChecksumIEEE([]byte(c)))
	}
	hash := fmt.Sprint(res)
	key := getDataplaneKey(app, hash)

	dataplaneResource := mesh.NewDataplaneResource()
	dataplaneResource.Spec.Networking = &mesh_proto.Dataplane_Networking{}
	dataplaneResource.Spec.Extensions = map[string]string{}
	dataplaneResource.Spec.Extensions[mesh_proto.ApplicationName] = app
	dataplaneResource.SetMeta(&resourceMetaObject{
		Name: key,
	})
	dataplaneResource.Spec.Networking.Address = url.Address()
	ifaces, err := InboundInterfacesFor(ctx, url)
	if err != nil {
		return err
	}
	ofaces, err := OutboundInterfacesFor(ctx, url)
	if err != nil {
		return err
	}
	dataplaneResource.Spec.Networking.Inbound = ifaces
	dataplaneResource.Spec.Networking.Outbound = ofaces
	l.dataplaneCache.Store(key, dataplaneResource)
	return nil
}

func InboundInterfacesFor(ctx context.Context, url *common.URL) ([]*mesh_proto.Dataplane_Networking_Inbound, error) {
	var ifaces []*mesh_proto.Dataplane_Networking_Inbound
	num, err := strconv.ParseUint(url.Port, 10, 32)
	if err != nil {
		return nil, err
	}
	ifaces = append(ifaces, &mesh_proto.Dataplane_Networking_Inbound{
		Port: uint32(num),
	})
	return ifaces, nil
}

func OutboundInterfacesFor(ctx context.Context, url *common.URL) ([]*mesh_proto.Dataplane_Networking_Outbound, error) {
	var outbounds []*mesh_proto.Dataplane_Networking_Outbound

	return outbounds, nil
}

func getDataplaneKey(app string, hash string) string {
	return app + keySeparator + hash
}
