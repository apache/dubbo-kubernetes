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

package sync

import (
	"context"
)

import (
	"github.com/go-logr/logr"

	"github.com/pkg/errors"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/xds/cache/mesh"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
)

type DataplaneWatchdogDependencies struct {
	DataplaneProxyBuilder *DataplaneProxyBuilder
	DataplaneReconciler   SnapshotReconciler
	IngressProxyBuilder   *IngressProxyBuilder
	IngressReconciler     SnapshotReconciler
	EgressProxyBuilder    *EgressProxyBuilder
	EgressReconciler      SnapshotReconciler
	EnvoyCpCtx            *xds_context.ControlPlaneContext
	MetadataTracker       DataplaneMetadataTracker
	ResManager            core_manager.ReadOnlyResourceManager
	MeshCache             *mesh.Cache
}

type Status string

var (
	SkipStatus      Status = "skip"
	GeneratedStatus Status = "generated"
	ChangedStatus   Status = "changed"
)

type SyncResult struct {
	ProxyType mesh_proto.ProxyType
	Status    Status
}

type DataplaneWatchdog struct {
	DataplaneWatchdogDependencies
	key core_model.ResourceKey
	log logr.Logger

	// state of watchdog
	lastHash         string // last Mesh hash that was used to **successfully** generate Reconcile Envoy config
	dpType           mesh_proto.ProxyType
	proxyTypeSettled bool
	dpAddress        string
}

func NewDataplaneWatchdog(deps DataplaneWatchdogDependencies, dpKey core_model.ResourceKey) *DataplaneWatchdog {
	return &DataplaneWatchdog{
		DataplaneWatchdogDependencies: deps,
		key:                           dpKey,
		log:                           core.Log.WithValues("key", dpKey),
		proxyTypeSettled:              false,
	}
}

func (d *DataplaneWatchdog) Sync(ctx context.Context) (SyncResult, error) {
	metadata := d.MetadataTracker.Metadata(d.key)
	if metadata == nil {
		return SyncResult{}, errors.New("metadata cannot be nil")
	}

	if d.dpType == "" {
		d.dpType = metadata.GetProxyType()
	}
	switch d.dpType {
	case mesh_proto.DataplaneProxyType:
		return d.syncDataplane(ctx, metadata)
	case mesh_proto.IngressProxyType:
		return d.syncIngress(ctx, metadata)
	case mesh_proto.EgressProxyType:
		return d.syncEgress(ctx, metadata)
	default:
		return SyncResult{}, nil
	}
}

func (d *DataplaneWatchdog) Cleanup() error {
	proxyID := core_xds.FromResourceKey(d.key)
	switch d.dpType {
	case mesh_proto.DataplaneProxyType:
		return d.DataplaneReconciler.Clear(&proxyID)
	case mesh_proto.IngressProxyType:
		return d.IngressReconciler.Clear(&proxyID)
	case mesh_proto.EgressProxyType:
		return d.EgressReconciler.Clear(&proxyID)
	default:
		return nil
	}
}

func (d *DataplaneWatchdog) syncIngress(ctx context.Context, metadata *core_xds.DataplaneMetadata) (SyncResult, error) {
	return SyncResult{}, nil
}

func (d *DataplaneWatchdog) syncEgress(ctx context.Context, metadata *core_xds.DataplaneMetadata) (SyncResult, error) {
	return SyncResult{}, nil
}

func (d *DataplaneWatchdog) syncDataplane(ctx context.Context, metadata *core_xds.DataplaneMetadata) (SyncResult, error) {
	return SyncResult{}, nil
}
