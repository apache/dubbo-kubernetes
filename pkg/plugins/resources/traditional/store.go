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

package traditional

import (
	"context"
	"fmt"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	dubbo_identifier "dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"

	"github.com/dubbogo/go-zookeeper/zk"

	"github.com/dubbogo/gost/encoding/yaml"

	"github.com/pkg/errors"

	"golang.org/x/exp/maps"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"github.com/apache/dubbo-kubernetes/pkg/core/governance"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"github.com/apache/dubbo-kubernetes/pkg/core/reg_client"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	util_k8s "github.com/apache/dubbo-kubernetes/pkg/util/k8s"
)

const (
	dubboGroup    = "dubbo"
	mappingGroup  = "mapping"
	dubboConfig   = "config"
	metadataGroup = "metadata"
	cpGroup       = "dubbo-cp"
	pathSeparator = "/"
)

type traditionalStore struct {
	configCenter   config_center.DynamicConfiguration
	metadataReport report.MetadataReport
	registryCenter dubboRegistry.Registry
	governance     governance.GovernanceConfig
	dCache         *sync.Map
	regClient      reg_client.RegClient
	eventWriter    events.Emitter
	mu             sync.RWMutex
}

func NewStore(
	configCenter config_center.DynamicConfiguration,
	metadataReport report.MetadataReport,
	registryCenter dubboRegistry.Registry,
	governance governance.GovernanceConfig,
	dCache *sync.Map,
	regClient reg_client.RegClient,
) store.ResourceStore {
	return &traditionalStore{
		configCenter:   configCenter,
		metadataReport: metadataReport,
		registryCenter: registryCenter,
		governance:     governance,
		dCache:         dCache,
		regClient:      regClient,
	}
}

func (t *traditionalStore) SetEventWriter(writer events.Emitter) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.eventWriter = writer
}

func (t *traditionalStore) Create(_ context.Context, resource core_model.Resource, fs ...store.CreateOptionsFunc) error {
	var err error
	opts := store.NewCreateOptions(fs...)
	name, _, err := util_k8s.CoreNameToK8sName(opts.Name)
	if err != nil {
		return err
	}
	switch resource.Descriptor().Name {
	case mesh.MappingType:
		spec := resource.GetSpec()
		mapping := spec.(*mesh_proto.Mapping)
		appNames := mapping.ApplicationNames
		serviceInterface := mapping.InterfaceName
		for _, app := range appNames {
			err = t.metadataReport.RegisterServiceAppMapping(serviceInterface, mappingGroup, app)
			if err != nil {
				return err
			}
		}
	case mesh.MetaDataType:
		spec := resource.GetSpec()
		metadata := spec.(*mesh_proto.MetaData)
		identifier := &dubbo_identifier.SubscriberMetadataIdentifier{
			Revision: metadata.GetRevision(),
			BaseApplicationMetadataIdentifier: dubbo_identifier.BaseApplicationMetadataIdentifier{
				Application: metadata.GetApp(),
				Group:       dubboGroup,
			},
		}
		services := map[string]*common.ServiceInfo{}
		// 把metadata赋值到services中
		for key, serviceInfo := range metadata.GetServices() {
			services[key] = &common.ServiceInfo{
				Name:     serviceInfo.GetName(),
				Group:    serviceInfo.GetGroup(),
				Version:  serviceInfo.GetVersion(),
				Protocol: serviceInfo.GetProtocol(),
				Path:     serviceInfo.GetPath(),
				Params:   serviceInfo.GetParams(),
			}
		}
		info := &common.MetadataInfo{
			App:      metadata.GetApp(),
			Revision: metadata.GetRevision(),
			Services: services,
		}
		err = t.metadataReport.PublishAppMetadata(identifier, info)
		if err != nil {
			return err
		}
	case mesh.DataplaneType:
		// Dataplane无法Create, 只能Get和List
	case mesh.TagRouteType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		id := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetRoutePath(id, consts.TagRoute)
		bytes, err := core_model.ToYAML(resource.GetSpec())
		if err != nil {
			return err
		}

		err = t.governance.SetConfig(path, string(bytes))
		if err != nil {
			return err
		}
	case mesh.ConditionRouteType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		id := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetRoutePath(id, consts.ConditionRoute)

		bytes, err := core_model.ToYAML(resource.GetSpec())
		if err != nil {
			return err
		}

		err = t.governance.SetConfig(path, string(bytes))
		if err != nil {
			return err
		}
	case mesh.DynamicConfigType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		id := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetOverridePath(id)
		bytes, err := core_model.ToYAML(resource.GetSpec())
		if err != nil {
			return err
		}

		err = t.governance.SetConfig(path, string(bytes))
		if err != nil {
			return err
		}
	default:
		bytes, err := core_model.ToYAML(resource.GetSpec())
		if err != nil {
			return err
		}

		path := GenerateCpGroupPath(string(resource.Descriptor().Name), name)
		// 使用RegClient
		err = t.regClient.SetContent(path, bytes)
		if err != nil {
			return err
		}
	}

	resource.SetMeta(&resourceMetaObject{
		Name:             name,
		Mesh:             opts.Mesh,
		CreationTime:     opts.CreationTime,
		ModificationTime: opts.CreationTime,
		Labels:           maps.Clone(opts.Labels),
	})

	if t.eventWriter != nil {
		go func() {
			t.eventWriter.Send(events.ResourceChangedEvent{
				Operation: events.Create,
				Type:      resource.Descriptor().Name,
				Key: core_model.MetaToResourceKey(&resourceMetaObject{
					Name: name,
					Mesh: opts.Mesh,
				}),
			})
		}()
	}
	return nil
}

func (t *traditionalStore) Update(ctx context.Context, resource core_model.Resource, fs ...store.UpdateOptionsFunc) error {
	opts := store.NewUpdateOptions(fs...)
	name, _, err := util_k8s.CoreNameToK8sName(opts.Name)
	if err != nil {
		return err
	}
	switch resource.Descriptor().Name {
	case mesh.DataplaneType:
		// Dataplane资源无法更新, 只能获取和删除
	case mesh.TagRouteType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		id := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetRoutePath(id, consts.TagRoute)
		cfg, err := t.governance.GetConfig(path)
		if err != nil {
			return err
		}
		if cfg == "" {
			return fmt.Errorf("tag route %s not found", id)
		}
		bytes, err := core_model.ToYAML(resource.GetSpec())
		if err != nil {
			return err
		}
		err = t.governance.SetConfig(path, string(bytes))
		if err != nil {
			return err
		}
	case mesh.ConditionRouteType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		id := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetRoutePath(id, consts.ConditionRoute)
		cfg, err := t.governance.GetConfig(path)
		if err != nil {
			return err
		}
		if cfg == "" {
			if cfg == "" {
				return fmt.Errorf("no existing condition route for path: %s", path)
			}
		}

		bytes, err := core_model.ToYAML(resource.GetSpec())
		if err != nil {
			return err
		}
		err = t.governance.SetConfig(path, string(bytes))
		if err != nil {
			return err
		}
	case mesh.DynamicConfigType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		id := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetOverridePath(id)
		existConfig, err := t.governance.GetConfig(path)
		if err != nil {
			return err
		}
		override := &mesh_proto.DynamicConfig{}
		err = yaml.UnmarshalYML([]byte(existConfig), override)
		if err != nil {
			return err
		}
		configs := make([]*mesh_proto.OverrideConfig, 0)
		if len(override.Configs) > 0 {
			for _, c := range override.Configs {
				if consts.Configs.Contains(c.Type) {
					configs = append(configs, c)
				}
			}
		}
		update := resource.GetSpec().(*mesh_proto.DynamicConfig)
		configs = append(configs, update.Configs...)
		override.Configs = configs
		override.Enabled = update.Enabled
		if b, err := yaml.MarshalYML(override); err != nil {
			return err
		} else {
			err := t.governance.SetConfig(path, string(b))
			if err != nil {
				return err
			}
		}
	case mesh.MappingType:
		spec := resource.GetSpec()
		mapping := spec.(*mesh_proto.Mapping)
		appNames := mapping.ApplicationNames
		serviceInterface := mapping.InterfaceName
		for _, app := range appNames {
			path := getMappingPath(serviceInterface)
			// 先使用regClient判断是否存在, 如果存在的话就先删除再更新
			bytes, err := t.regClient.GetContent(path)
			if err != nil {
				return err
			}
			if len(bytes) != 0 {
				// 说明有内容, 需要先删除
				err := t.regClient.DeleteContent(path)
				if err != nil {
					return err
				}
			}
			err = t.metadataReport.RegisterServiceAppMapping(serviceInterface, mappingGroup, app)
			if err != nil {
				return err
			}
		}
	case mesh.MetaDataType:
		spec := resource.GetSpec()
		metadata := spec.(*mesh_proto.MetaData)
		identifier := &dubbo_identifier.SubscriberMetadataIdentifier{
			Revision: metadata.GetRevision(),
			BaseApplicationMetadataIdentifier: dubbo_identifier.BaseApplicationMetadataIdentifier{
				Application: metadata.GetApp(),
				Group:       dubboGroup,
			},
		}
		// 先判断identifier是否存在, 如果存在到话需要将其删除
		content, err := t.regClient.GetContent(getMetadataPath(metadata.GetApp(), metadata.GetRevision()))
		if err != nil {
			return err
		}
		if len(content) != 0 {
			// 如果不为空, 先删除
			err := t.regClient.DeleteContent(getMetadataPath(metadata.GetApp(), metadata.GetRevision()))
			if err != nil {
				return err
			}
		}
		services := map[string]*common.ServiceInfo{}
		// 把metadata赋值到services中
		for key, serviceInfo := range metadata.GetServices() {
			services[key] = &common.ServiceInfo{
				Name:     serviceInfo.GetName(),
				Group:    serviceInfo.GetGroup(),
				Version:  serviceInfo.GetVersion(),
				Protocol: serviceInfo.GetProtocol(),
				Path:     serviceInfo.GetPath(),
				Params:   serviceInfo.GetParams(),
			}
		}
		info := &common.MetadataInfo{
			App:      metadata.GetApp(),
			Revision: metadata.GetRevision(),
			Services: services,
		}
		err = t.metadataReport.PublishAppMetadata(identifier, info)
		if err != nil {
			return err
		}
	default:
		bytes, err := core_model.ToYAML(resource.GetSpec())
		if err != nil {
			return err
		}

		path := GenerateCpGroupPath(string(resource.Descriptor().Name), name)
		// 使用RegClient
		err = t.regClient.SetContent(path, bytes)
		if err != nil {
			return err
		}
	}
	resource.SetMeta(&resourceMetaObject{
		Name:             name,
		Mesh:             opts.Mesh,
		ModificationTime: opts.ModificationTime,
		Labels:           maps.Clone(opts.Labels),
	})

	if t.eventWriter != nil {
		go func() {
			t.eventWriter.Send(events.ResourceChangedEvent{
				Operation: events.Update,
				Type:      resource.Descriptor().Name,
				Key: core_model.MetaToResourceKey(&resourceMetaObject{
					Name: name,
					Mesh: opts.Mesh,
				}),
			})
		}()
	}
	return nil
}

func (t *traditionalStore) Delete(ctx context.Context, resource core_model.Resource, fs ...store.DeleteOptionsFunc) error {
	opts := store.NewDeleteOptions(fs...)
	name, _, err := util_k8s.CoreNameToK8sName(opts.Name)
	if err != nil {
		return err
	}
	switch resource.Descriptor().Name {
	case mesh.DataplaneType:
		// 不支持删除
	case mesh.TagRouteType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		key := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetOverridePath(key)
		err := t.governance.DeleteConfig(path)
		if err != nil {
			return err
		}
	case mesh.ConditionRouteType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		key := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetRoutePath(key, consts.ConditionRoute)
		err := t.governance.DeleteConfig(path)
		if err != nil {
			return err
		}
	case mesh.DynamicConfigType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		key := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetOverridePath(key)
		conf, err := t.governance.GetConfig(path)
		if err != nil {
			logger.Sugar().Error(err.Error())
			return err
		}
		if err := core_model.FromYAML([]byte(conf), resource.GetSpec()); err != nil {
			return err
		}
		override := resource.GetSpec().(*mesh_proto.DynamicConfig)
		if len(override.Configs) > 0 {
			newConfigs := make([]*mesh_proto.OverrideConfig, 0)
			for _, c := range override.Configs {
				if consts.Configs.Contains(c.Type) {
					newConfigs = append(newConfigs, c)
				}
			}
			if len(newConfigs) == 0 {
				err := t.governance.DeleteConfig(path)
				if err != nil {
					return err
				}
			} else {
				override.Configs = newConfigs
				if b, err := yaml.MarshalYML(override); err != nil {
					return err
				} else {
					err := t.governance.SetConfig(path, string(b))
					if err != nil {
						return err
					}
				}
			}
		} else {
			err := t.governance.DeleteConfig(path)
			if err != nil {
				return err
			}
		}
	case mesh.MappingType:
		// 无法删除
	case mesh.MetaDataType:
		// 无法删除
	default:
		path := GenerateCpGroupPath(string(resource.Descriptor().Name), name)
		err = t.regClient.DeleteContent(path)
		if err != nil {
			return err
		}
	}

	if t.eventWriter != nil {
		go func() {
			t.eventWriter.Send(events.ResourceChangedEvent{
				Operation: events.Delete,
				Type:      resource.Descriptor().Name,
				Key: core_model.ResourceKey{
					Mesh: opts.Mesh,
					Name: name,
				},
			})
		}()
	}
	return nil
}

func (c *traditionalStore) Get(_ context.Context, resource core_model.Resource, fs ...store.GetOptionsFunc) error {
	opts := store.NewGetOptions(fs...)

	name, _, err := util_k8s.CoreNameToK8sName(opts.Name)
	if err != nil {
		return err
	}

	switch resource.Descriptor().Name {
	case mesh.DataplaneType:
		value, ok := c.dCache.Load(name)
		if !ok {
			return nil
		}
		r := value.(core_model.Resource)
		resource.SetMeta(r.GetMeta())
		err = resource.SetSpec(r.GetSpec())
		if err != nil {
			return err
		}
		resource.SetMeta(&resourceMetaObject{
			Name: name,
			Mesh: opts.Mesh,
		})
	case mesh.TagRouteType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		id := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetRoutePath(id, consts.TagRoute)
		cfg, err := c.governance.GetConfig(path)
		if err != nil {
			return err
		}
		if cfg != "" {
			if err := core_model.FromYAML([]byte(cfg), resource.GetSpec()); err != nil {
				return errors.Wrap(err, "failed to convert json to spec")
			}
		}
		resource.SetMeta(&resourceMetaObject{
			Name: name,
			Mesh: opts.Mesh,
		})
	case mesh.ConditionRouteType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		id := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetRoutePath(id, consts.ConditionRoute)
		cfg, err := c.governance.GetConfig(path)
		if err != nil {
			return err
		}
		if cfg != "" {
			if err := core_model.FromYAML([]byte(cfg), resource.GetSpec()); err != nil {
				return errors.Wrap(err, "failed to convert json to spec")
			}
		}
		resource.SetMeta(&resourceMetaObject{
			Name: name,
			Mesh: opts.Mesh,
		})
	case mesh.DynamicConfigType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		id := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetOverridePath(id)
		cfg, err := c.governance.GetConfig(path)
		if err != nil {
			return err
		}
		if cfg != "" {
			if err := core_model.FromYAML([]byte(cfg), resource.GetSpec()); err != nil {
				return errors.Wrap(err, "failed to convert json to spec")
			}
		}
		resource.SetMeta(&resourceMetaObject{
			Name: name,
			Mesh: opts.Mesh,
		})
	case mesh.MappingType:
		// Get通过Key获取, 不设置listener
		set, err := c.metadataReport.GetServiceAppMapping(name, mappingGroup, nil)
		if err != nil {
			if errors.Is(err, zk.ErrNoNode) {
				return nil
			}
			return err
		}

		meta := &resourceMetaObject{
			Name: name,
			Mesh: opts.Mesh,
		}
		resource.SetMeta(meta)
		mapping := resource.GetSpec().(*mesh_proto.Mapping)
		mapping.Zone = "default"
		mapping.InterfaceName = name
		var items []string
		for k := range set.Items {
			items = append(items, fmt.Sprintf("%v", k))
		}
		mapping.ApplicationNames = items
		resource.SetMeta(&resourceMetaObject{
			Name: name,
			Mesh: opts.Mesh,
		})
	case mesh.MetaDataType:
		// 拆分name得到revision和app
		app, revision := splitAppAndRevision(name)
		if revision == "" {
			children, err := c.regClient.GetChildren(getMetadataPath(app))
			if err != nil {
				return err
			}
			revision = children[0]
		}
		id := dubbo_identifier.NewSubscriberMetadataIdentifier(app, revision)
		appMetadata, err := c.metadataReport.GetAppMetadata(id)
		if err != nil {
			return err
		}
		metaData := resource.GetSpec().(*mesh_proto.MetaData)
		metaData.App = appMetadata.App
		metaData.Revision = appMetadata.Revision
		service := map[string]*mesh_proto.ServiceInfo{}
		for key, serviceInfo := range appMetadata.Services {
			service[key] = &mesh_proto.ServiceInfo{
				Name:     serviceInfo.Name,
				Group:    serviceInfo.Group,
				Version:  serviceInfo.Version,
				Protocol: serviceInfo.Protocol,
				Path:     serviceInfo.Path,
				Params:   serviceInfo.Params,
			}
		}
		metaData.Services = service
		resource.SetMeta(&resourceMetaObject{
			Name: name,
			Mesh: opts.Mesh,
		})
	default:
		path := GenerateCpGroupPath(string(resource.Descriptor().Name), name)
		value, err := c.regClient.GetContent(path)
		if err != nil {
			return err
		}
		if err := core_model.FromYAML(value, resource.GetSpec()); err != nil {
			return err
		}
		resource.SetMeta(&resourceMetaObject{
			Name: name,
			Mesh: opts.Mesh,
		})
	}
	return nil
}

func (c *traditionalStore) List(_ context.Context, resources core_model.ResourceList, fs ...store.ListOptionsFunc) error {
	opts := store.NewListOptions(fs...)

	switch resources.GetItemType() {
	case mesh.DataplaneType:
		// iterator services key set
		c.dCache.Range(func(key, value any) bool {
			item := resources.NewItem()
			r := value.(core_model.Resource)
			item.SetMeta(&resourceMetaObject{
				Name: key.(string),
			})
			err := item.SetSpec(r.GetSpec())
			if err != nil {
				return false
			}
			if err := resources.AddItem(item); err != nil {
				return false
			}
			return true
		})
	case mesh.MappingType:
		// 1. 首先获取到所有到key
		keys, err := c.metadataReport.GetConfigKeysByGroup(mappingGroup)
		if err != nil {
			return err
		}
		for _, key := range keys.Values() {
			key := key.(string)
			// 通过key得到所有的mapping映射关系
			set, err := c.metadataReport.GetServiceAppMapping(key, mappingGroup, nil)
			if err != nil {
				return err
			}
			meta := &resourceMetaObject{
				Name: key,
			}
			item := resources.NewItem()
			item.SetMeta(meta)
			mapping := item.GetSpec().(*mesh_proto.Mapping)
			mapping.Zone = "default"
			mapping.InterfaceName = key
			var items []string
			for k := range set.Items {
				items = append(items, fmt.Sprintf("%v", k))
			}
			mapping.ApplicationNames = items
			err = resources.AddItem(item)
			if err != nil {
				return err
			}
		}
	case mesh.MetaDataType:
		// 1. 获取到所有的key, key是application(应用名)
		rootDir := getMetadataPath()
		appNames, err := c.regClient.GetChildren(rootDir)
		if err != nil {
			return err
		}
		for _, app := range appNames {
			// 2. 获取到该应用名下所有的revision
			path := getMetadataPath(app)
			revisions, err := c.regClient.GetChildren(path)
			if err != nil {
				return err
			}
			if revisions[0] == "provider" ||
				revisions[0] == "consumer" {
				continue
			}
			for _, revision := range revisions {
				id := dubbo_identifier.NewSubscriberMetadataIdentifier(app, revision)
				appMetadata, err := c.metadataReport.GetAppMetadata(id)
				if err != nil {
					return err
				}
				item := resources.NewItem()
				metaData := item.GetSpec().(*mesh_proto.MetaData)
				metaData.App = appMetadata.App
				metaData.Revision = appMetadata.Revision
				service := map[string]*mesh_proto.ServiceInfo{}
				for key, serviceInfo := range appMetadata.Services {
					service[key] = &mesh_proto.ServiceInfo{
						Name:     serviceInfo.Name,
						Group:    serviceInfo.Group,
						Version:  serviceInfo.Version,
						Protocol: serviceInfo.Protocol,
						Path:     serviceInfo.Path,
						Params:   serviceInfo.Params,
					}
				}
				metaData.Services = service
				item.SetMeta(&resourceMetaObject{
					Name:    app,
					Version: revision,
				})
				err = resources.AddItem(item)
				if err != nil {
					return err
				}
			}
		}

	case mesh.DynamicConfigType:
		// 不支持List
	case mesh.TagRouteType:
		// 不支持List
	case mesh.ConditionRouteType:
		// 不支持List
	default:
		rootDir := getDubboCpPath(string(resources.GetItemType()))
		names, err := c.regClient.GetChildren(rootDir)
		if err != nil {
			return err
		}
		for _, name := range names {
			path := getDubboCpPath(string(resources.GetItemType()), name)
			bytes, err := c.regClient.GetContent(path)
			if err != nil {
				return err
			}
			item := resources.NewItem()
			if err = core_model.FromYAML(bytes, item.GetSpec()); err != nil {
				return err
			}
			item.SetMeta(&resourceMetaObject{
				Name:   name,
				Labels: maps.Clone(opts.Labels),
			})
			err = resources.AddItem(item)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
