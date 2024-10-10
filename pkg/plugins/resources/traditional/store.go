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
	"strings"
	"sync"

	dubboconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	"github.com/apache/dubbo-kubernetes/pkg/core/registry"
	gxset "github.com/dubbogo/gost/container/set"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	dubbo_identifier "dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"

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
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
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
	appContext     *registry.ApplicationContext
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
	appContext *registry.ApplicationContext,
) store.ResourceStore {
	return &traditionalStore{
		configCenter:   configCenter,
		metadataReport: metadataReport,
		registryCenter: registryCenter,
		governance:     governance,
		dCache:         dCache,
		regClient:      regClient,
		appContext:     appContext,
	}
}

func (t *traditionalStore) SetEventWriter(writer events.Emitter) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.eventWriter = writer
}

func (t *traditionalStore) Create(ctx context.Context, resource core_model.Resource, fs ...store.CreateOptionsFunc) error {
	var err error
	opts := store.NewCreateOptions(fs...)
	if opts.Name == core_model.DefaultMesh {
		opts.Name += ".universal"
	}
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
		key := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetRoutePath(key, consts.TagRoute)
		bytes, err := core_model.ToYAML(resource.GetSpec())
		if err != nil {
			return err
		}
		cfg, _ := t.governance.GetConfig(path)
		if cfg != "" {
			return fmt.Errorf("%s Config is exsited ", path)
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
		key := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetRoutePath(key, consts.ConditionRoute)
		bytes, err := resource.GetSpec().(*mesh_proto.ConditionRoute).ToYAML()
		if err != nil {
			return err
		}
		cfg, _ := t.governance.GetConfig(path)
		if cfg != "" {
			return fmt.Errorf("%s Config is exsited ", path)
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
		key := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetOverridePath(key)
		bytes, err := core_model.ToYAML(resource.GetSpec())
		if err != nil {
			return err
		}
		cfg, _ := t.governance.GetConfig(path)
		if cfg != "" {
			return fmt.Errorf("%s Config is exsited ", path)
		}
		err = t.governance.SetConfig(path, string(bytes))
		if err != nil {
			return err
		}
	case mesh.AffinityRouteType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		key := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetRoutePath(key, consts.AffinityRoute)
		bytes, err := core_model.ToYAML(resource.GetSpec())
		if err != nil {
			return err
		}
		cfg, _ := t.governance.GetConfig(path)
		if cfg != "" {
			return fmt.Errorf("%s Config is exsited ", path)
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
	if opts.Name == core_model.DefaultMesh {
		opts.Name += ".universal"
	}
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
			return core_store.ErrorResourceNotFound(resource.Descriptor().Name, opts.Name, opts.Mesh)
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
			return core_store.ErrorResourceNotFound(resource.Descriptor().Name, opts.Name, opts.Mesh)
		}

		bytes, err := resource.GetSpec().(*mesh_proto.ConditionRoute).ToYAML()
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
		} else if existConfig == "" {
			return core_store.ErrorResourceNotFound(resource.Descriptor().Name, opts.Name, opts.Mesh)
		}

		if b, err := core_model.ToYAML(resource.GetSpec()); err != nil {
			return err
		} else {
			err := t.governance.SetConfig(path, string(b))
			if err != nil {
				return err
			}
		}
	case mesh.AffinityRouteType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		id := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetRoutePath(id, consts.AffinityRoute)

		existConfig, err := t.governance.GetConfig(path)
		if err != nil {
			return err
		} else if existConfig == "" {
			return core_store.ErrorResourceNotFound(resource.Descriptor().Name, opts.Name, opts.Mesh)
		}

		if b, err := core_model.ToYAML(resource.GetSpec()); err != nil {
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
	if opts.Name == core_model.DefaultMesh {
		opts.Name += ".universal"
	}
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
		path := mesh_proto.GetRoutePath(key, consts.TagRoute)
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
		_, err := t.governance.GetConfig(path)
		if err != nil {
			logger.Sugar().Error(err.Error())
			return err
		}
		err = t.governance.DeleteConfig(path)
		if err != nil {
			return err
		}
	case mesh.AffinityRouteType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		key := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetRoutePath(key, consts.AffinityRoute)
		_, err := t.governance.GetConfig(path)
		if err != nil {
			logger.Sugar().Error(err.Error())
			return err
		}
		err = t.governance.DeleteConfig(path)
		if err != nil {
			return err
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
	if opts.Name == core_model.DefaultMesh {
		opts.Name += ".universal"
	}
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
			res := &mesh_proto.TagRoute{}
			if err := core_model.FromYAML([]byte(cfg), res); err != nil {
				return errors.Wrap(err, "failed to convert json to spec")
			}
			err = resource.SetSpec(res)
			if err != nil {
				panic(err)
			}
		} else {
			return core_store.ErrorResourceNotFound(resource.Descriptor().Name, opts.Name, opts.Mesh)
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
			res, err := mesh_proto.ConditionRouteDecodeFromYAML([]byte(cfg))
			if err != nil {
				return err
			}
			err = resource.SetSpec(res)
			if err != nil {
				panic(err)
			}
		} else {
			return core_store.ErrorResourceNotFound(resource.Descriptor().Name, opts.Name, opts.Mesh)
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
		key := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetOverridePath(key)
		cfg, err := c.governance.GetConfig(path)
		if err != nil {
			return err
		}
		if cfg != "" {
			data := &mesh_proto.DynamicConfig{}
			if err := core_model.FromYAML([]byte(cfg), data); err != nil {
				return errors.Wrap(err, "failed to convert json to spec")
			}
			err = resource.SetSpec(data)
			if err != nil {
				panic(err)
			}
		} else {
			return core_store.ErrorResourceNotFound(resource.Descriptor().Name, opts.Name, opts.Mesh)
		}
		resource.SetMeta(&resourceMetaObject{
			Name: name,
			Mesh: opts.Mesh,
		})
	case mesh.AffinityRouteType:
		labels := opts.Labels
		base := mesh_proto.Base{
			Application:    labels[mesh_proto.Application],
			Service:        labels[mesh_proto.Service],
			ID:             labels[mesh_proto.ID],
			ServiceVersion: labels[mesh_proto.ServiceVersion],
			ServiceGroup:   labels[mesh_proto.ServiceGroup],
		}
		key := mesh_proto.BuildServiceKey(base)
		path := mesh_proto.GetRoutePath(key, consts.AffinityRoute)
		cfg, err := c.governance.GetConfig(path)
		if err != nil {
			return err
		}
		if cfg != "" {
			data := &mesh_proto.AffinityRoute{}
			if err := core_model.FromYAML([]byte(cfg), data); err != nil {
				return errors.Wrap(err, "failed to convert json to spec")
			}
			err = resource.SetSpec(data)
			if err != nil {
				panic(err)
			}
		} else {
			return core_store.ErrorResourceNotFound(resource.Descriptor().Name, opts.Name, opts.Mesh)
		}
		resource.SetMeta(&resourceMetaObject{
			Name: name,
			Mesh: opts.Mesh,
		})
	case mesh.MappingType:
		// Get通过Key获取, 不设置listener
		mappings := make(map[string]*gxset.HashSet)
		for app, instances := range c.appContext.GetAllInstances() {
			for _, instance := range instances {
				appMetadata := c.appContext.GetRevisionToMetadata(instance.GetMetadata()[dubboconstant.ExportedServicesRevisionPropertyName])
				for s := range appMetadata.Services {
					if set, ok := mappings[s]; !ok {
						set = gxset.NewSet()
						mappings[s] = set
					} else {
						set.Add(app)
					}
				}
			}
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
		for k := range mappings[name].Items {
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
		appMetadata := c.appContext.GetRevisionToMetadata(revision)
		if appMetadata == nil {
			return nil
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
			match, err := c.copyResource(key, item, r, opts)
			if err != nil || !match {
				return false
			}
			if err := resources.AddItem(item); err != nil {
				return false
			}
			return true
		})
	case mesh.MappingType:
		mappings := make(map[string]*gxset.HashSet)
		for app, instances := range c.appContext.GetAllInstances() {
			for _, instance := range instances {
				appMetadata := c.appContext.GetRevisionToMetadata(instance.GetMetadata()[dubboconstant.ExportedServicesRevisionPropertyName])
				for s := range appMetadata.Services {
					if set, ok := mappings[s]; !ok {
						set = gxset.NewSet()
						mappings[s] = set
					} else {
						set.Add(app)
					}
				}
			}
		}
		for key, set := range mappings {
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
			err := resources.AddItem(item)
			if err != nil {
				return err
			}
		}
	case mesh.MetaDataType:
		// 1. 获取到所有的key, key是application(应用名)
		for app, instances := range c.appContext.GetAllInstances() {
			if opts.NameContains != "" && !strings.Contains(app, opts.NameContains) {
				return nil
			}
			// 2. 获取到该应用名下所有的revision
			revisions := make(map[string]struct{})
			for _, instance := range instances {
				revisions[instance.GetMetadata()[dubboconstant.ExportedServicesRevisionPropertyName]] = struct{}{}
			}
			for revision := range revisions {
				appMetadata := c.appContext.GetRevisionToMetadata(revision)
				if appMetadata == nil {
					log.Error(nil, "Err loading app metadata with id %s", dubbo_identifier.NewSubscriberMetadataIdentifier(app, revision))
					continue
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
				err := resources.AddItem(item)
				if err != nil {
					return err
				}
			}
		}
	case mesh.DynamicConfigType:
		cfg, err := c.governance.GetList(consts.ConfiguratorRuleSuffix)
		if err != nil {
			return err
		}
		for name, rule := range cfg {
			newIt := resources.NewItem()
			ConfiguratorCfg, err := parseConfiguratorConfig(rule)
			_ = newIt.SetSpec(ConfiguratorCfg)
			meta := &resourceMetaObject{
				Name:   name,
				Mesh:   opts.Mesh,
				Labels: maps.Clone(opts.Labels),
			}
			if err != nil {
				logger.Errorf("failed to parse dynamicConfig rule: %s : %s, %s", name, rule, err.Error())
				continue
			}
			newIt.SetMeta(meta)
			_ = resources.AddItem(newIt)
		}
	case mesh.TagRouteType:
		cfg, err := c.governance.GetList(consts.TagRuleSuffix)
		if err != nil {
			return err
		}
		for name, rule := range cfg {
			newIt := resources.NewItem()
			ConfiguratorCfg, err := parseTagConfig(rule)
			_ = newIt.SetSpec(ConfiguratorCfg)
			meta := &resourceMetaObject{
				Name:   name,
				Mesh:   opts.Mesh,
				Labels: maps.Clone(opts.Labels),
			}
			if err != nil {
				logger.Errorf("failed to parse tag rule: %s : %s, %s", name, rule, err.Error())
				continue
			}
			newIt.SetMeta(meta)
			_ = resources.AddItem(newIt)
		}
	case mesh.ConditionRouteType:
		cfg, err := c.governance.GetList(consts.ConditionRuleSuffix)
		if err != nil {
			return err
		}
		for name, rule := range cfg {
			newIt := resources.NewItem()
			ConfiguratorCfg, err := parseConditionConfig(rule)
			if err != nil {
				logger.Errorf("failed to parse condition rule: %s : %s, %s", name, rule, err.Error())
				continue
			} else {
				_ = newIt.SetSpec(ConfiguratorCfg)
				meta := &resourceMetaObject{
					Name:   name,
					Mesh:   opts.Mesh,
					Labels: maps.Clone(opts.Labels),
				}
				newIt.SetMeta(meta)
				_ = resources.AddItem(newIt)
			}
		}
	case mesh.AffinityRouteType:
		cfg, err := c.governance.GetList(consts.AffinityRuleSuffix)
		if err != nil {
			return err
		}
		for name, rule := range cfg {
			newIt := resources.NewItem()
			ConfiguratorCfg, err := parseAffinityConfig(rule)
			if err != nil {
				logger.Errorf("failed to parse condition rule: %s : %s, %s", name, rule, err.Error())
				continue
			} else {
				_ = newIt.SetSpec(ConfiguratorCfg)
				meta := &resourceMetaObject{
					Name:   name,
					Mesh:   opts.Mesh,
					Labels: maps.Clone(opts.Labels),
				}
				newIt.SetMeta(meta)
				_ = resources.AddItem(newIt)
			}
		}
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

// copyResource, todo is copy necessary since they are of the same type?
func (c *traditionalStore) copyResource(key any, dst core_model.Resource, src core_model.Resource, opts *store.ListOptions) (bool, error) {
	name := opts.NameContains

	if name != "" && !strings.Contains(key.(string), name) {
		return false, nil
	}

	dst.SetMeta(src.GetMeta())
	err := dst.SetSpec(src.GetSpec())
	return true, err
}
