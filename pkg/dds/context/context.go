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

package context

import (
	"context"
	"github.com/apache/dubbo-kubernetes/pkg/dds/service"
	"reflect"
)

import (
	"github.com/pkg/errors"

	"google.golang.org/grpc"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/reflect/protoreflect"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	config_manager "github.com/apache/dubbo-kubernetes/pkg/core/config/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/dds"
	"github.com/apache/dubbo-kubernetes/pkg/dds/hash"
	"github.com/apache/dubbo-kubernetes/pkg/dds/mux"
	"github.com/apache/dubbo-kubernetes/pkg/dds/reconcile"
	"github.com/apache/dubbo-kubernetes/pkg/dds/util"
)

var log = core.Log.WithName("dds")

type Context struct {
	ZoneClientCtx        context.Context
	GlobalProvidedFilter reconcile.ResourceFilter
	ZoneProvidedFilter   reconcile.ResourceFilter
	GlobalServerFilters  []mux.Filter
	// Configs contains the names of system.ConfigResource that will be transferred from Global to Zone
	Configs map[string]bool

	GlobalResourceMapper reconcile.ResourceMapper
	ZoneResourceMapper   reconcile.ResourceMapper

	ServerStreamInterceptors []grpc.StreamServerInterceptor
	ServerUnaryInterceptor   []grpc.UnaryServerInterceptor
	EnvoyAdminRPCs           service.EnvoyAdminRPCs
}

func DefaultContext(
	ctx context.Context,
	manager manager.ResourceManager,
	cfg dubbo_cp.Config,
) *Context {
	configs := map[string]bool{
		config_manager.ClusterIdConfigKey: true,
	}

	globalMappers := []reconcile.ResourceMapper{
		UpdateResourceMeta(util.WithLabel(mesh_proto.ResourceOriginLabel, string(mesh_proto.GlobalResourceOrigin))),
		reconcile.If(
			reconcile.IsKubernetes(cfg.Store.Type),
			RemoveK8sSystemNamespaceSuffixMapper(cfg.Store.Kubernetes.SystemNamespace)),
		reconcile.If(
			reconcile.And(
				reconcile.ScopeIs(core_model.ScopeMesh),
				// secrets already named with mesh prefix for uniqueness on k8s, also Zone CP expects secret names to be in
				// particular format to be able to reference them
				reconcile.Not(reconcile.TypeIs(system.SecretType)),
			),
			HashSuffixMapper(true)),
	}

	zoneMappers := []reconcile.ResourceMapper{
		UpdateResourceMeta(
			util.WithLabel(mesh_proto.ResourceOriginLabel, string(mesh_proto.ZoneResourceOrigin)),
			util.WithLabel(mesh_proto.ZoneTag, cfg.Multizone.Zone.Name),
		),
		MapInsightResourcesZeroGeneration,
		reconcile.If(
			reconcile.IsKubernetes(cfg.Store.Type),
			RemoveK8sSystemNamespaceSuffixMapper(cfg.Store.Kubernetes.SystemNamespace)),
		HashSuffixMapper(false, mesh_proto.ZoneTag, mesh_proto.KubeNamespaceTag),
	}

	return &Context{
		ZoneClientCtx:        ctx,
		GlobalProvidedFilter: GlobalProvidedFilter(manager, configs),
		ZoneProvidedFilter:   ZoneProvidedFilter,
		Configs:              configs,
		GlobalResourceMapper: CompositeResourceMapper(globalMappers...),
		ZoneResourceMapper:   CompositeResourceMapper(zoneMappers...),
	}
}

// CompositeResourceMapper combines the given ResourceMappers into
// a single ResourceMapper which calls each in order. If an error
// occurs, the first one is returned and no further mappers are executed.
func CompositeResourceMapper(mappers ...reconcile.ResourceMapper) reconcile.ResourceMapper {
	return func(features dds.Features, r core_model.Resource) (core_model.Resource, error) {
		var err error
		for _, mapper := range mappers {
			if mapper == nil {
				continue
			}

			r, err = mapper(features, r)
			if err != nil {
				return r, err
			}
		}
		return r, nil
	}
}

type specWithDiscoverySubscriptions interface {
	GetSubscriptions() []*mesh_proto.DiscoverySubscription
	ProtoReflect() protoreflect.Message
}

// MapInsightResourcesZeroGeneration zeros "generation" field in resources for which
// the field has only local relevance. This prevents reconciliation from unnecessarily
// deeming the object to have changed.
func MapInsightResourcesZeroGeneration(_ dds.Features, r core_model.Resource) (core_model.Resource, error) {
	if spec, ok := r.GetSpec().(specWithDiscoverySubscriptions); ok {
		spec = proto.Clone(spec).(specWithDiscoverySubscriptions)
		for _, sub := range spec.GetSubscriptions() {
			sub.Generation = 0
		}

		meta := r.GetMeta()
		resType := reflect.TypeOf(r).Elem()

		newR := reflect.New(resType).Interface().(core_model.Resource)
		newR.SetMeta(meta)
		if err := newR.SetSpec(spec.(core_model.ResourceSpec)); err != nil {
			panic(any(errors.Wrap(err, "error setting spec on resource")))
		}

		return newR, nil
	}

	return r, nil
}

// RemoveK8sSystemNamespaceSuffixMapper is a mapper responsible for removing control plane system namespace suffixes
// from names of resources if resources are stored in kubernetes.
func RemoveK8sSystemNamespaceSuffixMapper(k8sSystemNamespace string) reconcile.ResourceMapper {
	return func(_ dds.Features, r core_model.Resource) (core_model.Resource, error) {
		util.TrimSuffixFromName(r, k8sSystemNamespace)
		return r, nil
	}
}

// HashSuffixMapper returns mapper that adds a hash suffix to the name during DDS sync
func HashSuffixMapper(checkDDSFeature bool, labelsToUse ...string) reconcile.ResourceMapper {
	return func(features dds.Features, r core_model.Resource) (core_model.Resource, error) {
		if checkDDSFeature && !features.HasFeature(dds.FeatureHashSuffix) {
			return r, nil
		}

		name := core_model.GetDisplayName(r)
		values := make([]string, 0, len(labelsToUse))
		for _, lbl := range labelsToUse {
			values = append(values, r.GetMeta().GetLabels()[lbl])
		}

		newObj := r.Descriptor().NewObject()
		newMeta := util.CloneResourceMeta(r.GetMeta(), util.WithName(hash.HashedName(r.GetMeta().GetMesh(), name, values...)))
		newObj.SetMeta(newMeta)
		_ = newObj.SetSpec(r.GetSpec())

		return newObj, nil
	}
}

func UpdateResourceMeta(fs ...util.CloneResourceMetaOpt) reconcile.ResourceMapper {
	return func(_ dds.Features, r core_model.Resource) (core_model.Resource, error) {
		r.SetMeta(util.CloneResourceMeta(r.GetMeta(), fs...))
		return r, nil
	}
}

func GlobalProvidedFilter(rm manager.ResourceManager, configs map[string]bool) reconcile.ResourceFilter {
	return func(ctx context.Context, clusterID string, features dds.Features, r core_model.Resource) bool {
		resName := r.GetMeta().GetName()

		switch {
		case r.Descriptor().Name == system.ConfigType:
			return configs[resName]
		case r.Descriptor().DDSFlags.Has(core_model.GlobalToAllButOriginalZoneFlag):
			zoneTag := util.ZoneTag(r)

			if clusterID == zoneTag {
				// don't need to sync resource to the zone where resource is originated from
				return false
			}

			zone := system.NewZoneResource()
			if err := rm.Get(ctx, zone, store.GetByKey(zoneTag, core_model.NoMesh)); err != nil {
				log.Error(err, "failed to get zone", "zone", zoneTag)
				// since there is no explicit 'enabled: false' then we don't
				// make any strong decisions which might affect connectivity
				return true
			}

			return zone.Spec.IsEnabled()
		default:
			return core_model.IsLocallyOriginated(config_core.Global, r)
		}
	}
}

func ZoneProvidedFilter(_ context.Context, _ string, _ dds.Features, r core_model.Resource) bool {
	return core_model.IsLocallyOriginated(config_core.Zone, r)
}
