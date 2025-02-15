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
	envoy_admin_tls "github.com/apache/dubbo-kubernetes/pkg/envoy/admin/tls"
	util_tls "github.com/apache/dubbo-kubernetes/pkg/tls"
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
	envoyAdminMTLS   *core_xds.ServerSideMTLSCerts
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
	default:
		return nil
	}
}

func (d *DataplaneWatchdog) syncIngress(ctx context.Context, metadata *core_xds.DataplaneMetadata) (SyncResult, error) {
	envoyCtx := &xds_context.Context{
		ControlPlane: d.EnvoyCpCtx,
		Mesh:         xds_context.MeshContext{}, // ZoneIngress does not have a mesh!
	}

	aggregatedMeshCtxs, err := xds_context.AggregateMeshContexts(ctx, d.ResManager, d.MeshCache.GetMeshContext)
	if err != nil {
		return SyncResult{}, errors.Wrap(err, "could not aggregate mesh contexts")
	}

	result := SyncResult{
		ProxyType: mesh_proto.IngressProxyType,
	}
	syncForConfig := aggregatedMeshCtxs.Hash != d.lastHash
	var syncForCert bool
	for _, mesh := range aggregatedMeshCtxs.Meshes {
		certInfo := d.EnvoyCpCtx.Secrets.Info(
			mesh_proto.IngressProxyType,
			core_model.ResourceKey{Mesh: mesh.GetMeta().GetName(), Name: d.key.Name},
		)
		syncForCert = syncForCert || (certInfo != nil && certInfo.ExpiringSoon()) // check if we need to regenerate config because identity cert is expiring soon.
	}
	if !syncForConfig && !syncForCert {
		result.Status = SkipStatus
		return result, nil
	}

	d.log.V(1).Info("snapshot hash updated, reconcile", "prev", d.lastHash, "current", aggregatedMeshCtxs.Hash)
	d.lastHash = aggregatedMeshCtxs.Hash

	if syncForCert {
		d.log.V(1).Info("certs expiring soon, reconcile")
	}

	proxy, err := d.IngressProxyBuilder.Build(ctx, d.key, aggregatedMeshCtxs)
	if err != nil {
		return SyncResult{}, errors.Wrap(err, "could not build ingress proxy")
	}

	networking := proxy.ZoneIngressProxy.ZoneIngressResource.Spec.GetNetworking()
	envoyAdminMTLS, err := d.getEnvoyAdminMTLS(ctx, networking.GetAddress(), networking.GetAdvertisedAddress())
	if err != nil {
		return SyncResult{}, errors.Wrap(err, "could not get Envoy Admin mTLS certs")
	}
	proxy.EnvoyAdminMTLSCerts = envoyAdminMTLS

	proxy.Metadata = metadata
	changed, err := d.IngressReconciler.Reconcile(ctx, *envoyCtx, proxy)
	if err != nil {
		return SyncResult{}, errors.Wrap(err, "could not reconcile")
	}
	if changed {
		result.Status = ChangedStatus
	} else {
		result.Status = GeneratedStatus
	}
	return result, nil
}

func (d *DataplaneWatchdog) syncEgress(ctx context.Context, metadata *core_xds.DataplaneMetadata) (SyncResult, error) {
	return SyncResult{}, nil
}

// syncDataplane syncs state of the Dataplane.
// It uses Mesh Hash to decide if we need to regenerate configuration or not.
func (d *DataplaneWatchdog) syncDataplane(ctx context.Context, metadata *core_xds.DataplaneMetadata) (SyncResult, error) {
	meshCtx, err := d.MeshCache.GetMeshContext(ctx, d.key.Mesh)
	if err != nil {
		return SyncResult{}, errors.Wrap(err, "could not get mesh context")
	}

	certInfo := d.EnvoyCpCtx.Secrets.Info(mesh_proto.DataplaneProxyType, d.key)
	// check if we need to regenerate config because identity cert is expiring soon.
	syncForCert := certInfo != nil && certInfo.ExpiringSoon()
	// check if we need to regenerate config because Dubbo policies has changed.
	syncForConfig := meshCtx.Hash != d.lastHash
	result := SyncResult{
		ProxyType: mesh_proto.DataplaneProxyType,
	}
	if !syncForConfig && !syncForCert {
		result.Status = SkipStatus
		return result, nil
	}
	if syncForConfig {
		d.log.V(1).Info("snapshot hash updated, reconcile", "prev", d.lastHash, "current", meshCtx.Hash)
	}
	if syncForCert {
		d.log.V(1).Info("certs expiring soon, reconcile")
	}

	envoyCtx := &xds_context.Context{
		ControlPlane: d.EnvoyCpCtx,
		Mesh:         meshCtx,
	}

	if _, found := meshCtx.DataplanesByName[d.key.Name]; !found {
		d.log.Info("Dataplane object not found. Can't regenerate XDS configuration. It's expected during Kubernetes namespace termination. " +
			"If it persists it's a bug.")
		result.Status = SkipStatus
		return result, nil
	}

	proxy, err := d.DataplaneProxyBuilder.Build(ctx, d.key, meshCtx)
	if err != nil {
		return SyncResult{}, errors.Wrap(err, "could not build dataplane proxy")
	}
	networking := proxy.Dataplane.Spec.Networking
	envoyAdminMTLS, err := d.getEnvoyAdminMTLS(ctx, networking.Address, networking.AdvertisedAddress)
	proxy.EnvoyAdminMTLSCerts = envoyAdminMTLS
	if !envoyCtx.Mesh.Resource.MTLSEnabled() {
		d.EnvoyCpCtx.Secrets.Cleanup(mesh_proto.DataplaneProxyType, d.key) // we need to cleanup secrets if mtls is disabled
	}
	proxy.Metadata = metadata
	changed, err := d.DataplaneReconciler.Reconcile(ctx, *envoyCtx, proxy)
	if err != nil {
		return SyncResult{}, errors.Wrap(err, "could not reconcile")
	}
	d.lastHash = meshCtx.Hash

	if changed {
		result.Status = ChangedStatus
	} else {
		result.Status = GeneratedStatus
	}
	return result, nil
}

func (d *DataplaneWatchdog) getEnvoyAdminMTLS(ctx context.Context, address string, advertisedAddress string) (core_xds.ServerSideMTLSCerts, error) {
	if d.envoyAdminMTLS == nil || d.dpAddress != address {
		ca, err := envoy_admin_tls.LoadCA(ctx, d.ResManager)
		if err != nil {
			return core_xds.ServerSideMTLSCerts{}, errors.Wrap(err, "could not load the CA")
		}
		caPair, err := util_tls.ToKeyPair(ca.PrivateKey, ca.Certificate[0])
		if err != nil {
			return core_xds.ServerSideMTLSCerts{}, err
		}
		ips := []string{address}
		if advertisedAddress != "" && advertisedAddress != address {
			ips = append(ips, advertisedAddress)
		}
		serverPair, err := envoy_admin_tls.GenerateServerCert(ca, ips...)
		if err != nil {
			return core_xds.ServerSideMTLSCerts{}, errors.Wrap(err, "could not generate server certificate")
		}

		envoyAdminMTLS := core_xds.ServerSideMTLSCerts{
			CaPEM:      caPair.CertPEM,
			ServerPair: serverPair,
		}
		// cache the Envoy Admin MTLS and dp address, so we
		// 1) don't have to do I/O on every sync
		// 2) have a stable certs = stable Envoy config
		// This means that if we want to change Envoy Admin CA, we need to restart all CP instances.
		// Additionally, we need to trigger cert generation when DP address has changed without DP reconnection.
		d.envoyAdminMTLS = &envoyAdminMTLS
		d.dpAddress = address
	}
	return *d.envoyAdminMTLS, nil
}
