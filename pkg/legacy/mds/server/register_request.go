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
	"time"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

type RegisterRequest struct {
	MetadataConfigsUpdated map[core_model.ResourceKey]*mesh_proto.MetaData
	MappingConfigsUpdates  map[core_model.ResourceKey]map[string]struct{}
}

func (q *RegisterRequest) Merge(req *RegisterRequest) *RegisterRequest {
	if q == nil {
		return req
	}
	// merge metadata
	for key, metadata := range q.MetadataConfigsUpdated {
		q.MetadataConfigsUpdated[key] = metadata
	}

	// merge mapping
	for key, newApps := range req.MappingConfigsUpdates {
		if _, ok := q.MappingConfigsUpdates[key]; !ok {
			q.MappingConfigsUpdates[key] = make(map[string]struct{})
		}
		for app := range newApps {
			q.MappingConfigsUpdates[key][app] = struct{}{}
		}
	}

	return q
}

func (m *MdsServer) metadataRegister(req *RegisterRequest) {
	for key, metadata := range req.MetadataConfigsUpdated {
		for i := 0; i < 3; i++ {
			if err := m.tryMetadataRegister(key, metadata); err != nil {
				log.Error(err, "register failed", "key", key)
			} else {
				break
			}
		}
	}
}

func (m *MdsServer) tryMetadataRegister(key core_model.ResourceKey, newMetadata *mesh_proto.MetaData) error {
	err := core_store.InTx(m.ctx, m.transactions, func(ctx context.Context) error {
		// get Metadata Resource first,
		// if Metadata is not found, create it,
		// else update it.
		metadata := core_mesh.NewMetaDataResource()
		err := m.resourceManager.Get(m.ctx, metadata, core_store.GetBy(key))
		if err != nil && !core_store.IsResourceNotFound(err) {
			log.Error(err, "get Metadata Resource")
			return err
		}

		if core_store.IsResourceNotFound(err) {
			// create if not found
			metadata.Spec = newMetadata
			err = m.resourceManager.Create(m.ctx, metadata, core_store.CreateBy(key), core_store.CreatedAt(time.Now()))
			if err != nil {
				log.Error(err, "create Metadata Resource failed")
				return err
			}

			log.Info("create Metadata Resource success", "key", key, "metadata", newMetadata)
			return nil
		} else {
			// if found, update it
			metadata.Spec = newMetadata

			err = m.resourceManager.Update(m.ctx, metadata, core_store.ModifiedAt(time.Now()))
			if err != nil {
				log.Error(err, "update Metadata Resource failed")
				return err
			}

			log.Info("update Metadata Resource success", "key", key, "metadata", newMetadata)
			return nil
		}
	})
	if err != nil {
		log.Error(err, "transactions failed")
		return err
	}

	return nil
}

func (s *MdsServer) mappingRegister(req *RegisterRequest) {
	for key, m := range req.MappingConfigsUpdates {
		var appNames []string
		for app := range m {
			appNames = append(appNames, app)
		}
		for i := 0; i < 3; i++ {
			if err := s.tryMappingRegister(key.Mesh, key.Name, appNames); err != nil {
				log.Error(err, "register failed", "key", key)
			} else {
				break
			}
		}
	}
}

func (s *MdsServer) tryMappingRegister(mesh, interfaceName string, newApps []string) error {
	err := core_store.InTx(s.ctx, s.transactions, func(ctx context.Context) error {
		key := core_model.ResourceKey{
			Mesh: mesh,
			Name: interfaceName,
		}

		// get Mapping Resource first,
		// if Mapping is not found, create it,
		// else update it.
		mapping := core_mesh.NewMappingResource()
		err := s.resourceManager.Get(s.ctx, mapping, core_store.GetBy(key))
		if err != nil && !core_store.IsResourceNotFound(err) {
			log.Error(err, "get Mapping Resource")
			return err
		}

		if core_store.IsResourceNotFound(err) {
			// create if not found
			mapping.Spec = &mesh_proto.Mapping{
				Zone:             s.localZone,
				InterfaceName:    interfaceName,
				ApplicationNames: newApps,
			}
			err = s.resourceManager.Create(s.ctx, mapping, core_store.CreateBy(key), core_store.CreatedAt(time.Now()))
			if err != nil {
				log.Error(err, "create Mapping Resource failed")
				return err
			}

			log.Info("create Mapping Resource success", "key", key, "applicationNames", newApps)
			return nil
		} else {
			// if found, update it
			previousLen := len(mapping.Spec.ApplicationNames)
			previousAppNames := make(map[string]struct{}, previousLen)
			for _, name := range mapping.Spec.ApplicationNames {
				previousAppNames[name] = struct{}{}
			}
			for _, newApp := range newApps {
				previousAppNames[newApp] = struct{}{}
			}
			if len(previousAppNames) == previousLen {
				log.Info("Mapping not need to register", "interfaceName", interfaceName, "applicationNames", newApps)
				return nil
			}

			mergedApps := make([]string, 0, len(previousAppNames))
			for name := range previousAppNames {
				mergedApps = append(mergedApps, name)
			}
			mapping.Spec = &mesh_proto.Mapping{
				Zone:             s.localZone,
				InterfaceName:    interfaceName,
				ApplicationNames: mergedApps,
			}

			err = s.resourceManager.Update(s.ctx, mapping, core_store.ModifiedAt(time.Now()))
			if err != nil {
				log.Error(err, "update Mapping Resource failed")
				return err
			}

			log.Info("update Mapping Resource success", "key", key, "applicationNames", newApps)
			return nil
		}
	})
	if err != nil {
		log.Error(err, "transactions failed")
		return err
	}

	return nil
}
