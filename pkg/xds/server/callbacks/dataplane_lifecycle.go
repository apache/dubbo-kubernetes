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

package callbacks

import (
	"context"
	"github.com/apache/dubbo-kubernetes/api/generic"
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/util/maps"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sync"
	"time"
)

var lifecycleLog = core.Log.WithName("xds").WithName("dp-lifecycle")

type DataplaneLifecycle struct {
	resManager          manager.ResourceManager
	proxyInfos          maps.Sync[core_model.ResourceKey, *proxyInfo]
	appCtx              context.Context
	deregistrationDelay time.Duration
	cpInstanceID        string
}

type proxyInfo struct {
	mtx       sync.Mutex
	proxyType mesh_proto.ProxyType
	connected bool
	deleted   bool
}

var _ DataplaneCallbacks = &DataplaneLifecycle{}

func NewDataplaneLifecycle(
	appCtx context.Context,
	resManager manager.ResourceManager,
	deregistrationDelay time.Duration,
	cpInstanceID string,
) *DataplaneLifecycle {
	return &DataplaneLifecycle{
		resManager:          resManager,
		proxyInfos:          maps.Sync[core_model.ResourceKey, *proxyInfo]{},
		appCtx:              appCtx,
		deregistrationDelay: deregistrationDelay,
		cpInstanceID:        cpInstanceID,
	}
}

func (d *DataplaneLifecycle) OnProxyConnected(streamID core_xds.StreamID, proxyKey core_model.ResourceKey, ctx context.Context, md core_xds.DataplaneMetadata) error {
	if md.Resource == nil {
		return nil
	}
	if err := d.validateProxyKey(proxyKey, md.Resource); err != nil {
		return err
	}
	return d.register(ctx, streamID, proxyKey, md)
}

func (d *DataplaneLifecycle) OnProxyReconnected(streamID core_xds.StreamID, proxyKey core_model.ResourceKey, ctx context.Context, md core_xds.DataplaneMetadata) error {
	if md.Resource == nil {
		return nil
	}
	if err := d.validateProxyKey(proxyKey, md.Resource); err != nil {
		return err
	}
	return d.register(ctx, streamID, proxyKey, md)
}

func (d *DataplaneLifecycle) OnProxyDisconnected(ctx context.Context, streamID core_xds.StreamID, proxyKey core_model.ResourceKey) {
	// OnStreamClosed method could be called either in case data plane proxy is down or
	// Kuma CP is gracefully shutting down. If Kuma CP is gracefully shutting down we
	// must not delete Dataplane resource, data plane proxy will be reconnected to another
	// instance of Kuma CP.
	select {
	case <-d.appCtx.Done():
		lifecycleLog.Info("graceful shutdown, don't delete Dataplane resource")
		return
	default:
	}

	d.deregister(ctx, streamID, proxyKey)
}

func (d *DataplaneLifecycle) register(
	ctx context.Context,
	streamID core_xds.StreamID,
	proxyKey core_model.ResourceKey,
	md core_xds.DataplaneMetadata,
) error {
	log := lifecycleLog.
		WithValues("proxyType", md.GetProxyType()).
		WithValues("proxyKey", proxyKey).
		WithValues("streamID", streamID).
		WithValues("resource", md.Resource)

	info, loaded := d.proxyInfos.LoadOrStore(proxyKey, &proxyInfo{
		proxyType: md.GetProxyType(),
	})

	info.mtx.Lock()
	defer info.mtx.Unlock()

	if info.deleted {
		// we took info object that was deleted from proxyInfo map by other goroutine, return err so DPP retry registration
		return errors.Errorf("attempt to concurently register deleted DPP resource, needs retry")
	}

	log.Info("register proxy")

	err := manager.Upsert(ctx, d.resManager, core_model.MetaToResourceKey(md.Resource.GetMeta()), proxyResource(md.GetProxyType()), func(existing core_model.Resource) error {
		return existing.SetSpec(md.Resource.GetSpec())
	})
	if err != nil {
		log.Info("cannot register proxy", "reason", err.Error())
		if !loaded {
			info.deleted = true
			d.proxyInfos.Delete(proxyKey)
		}
		return errors.Wrap(err, "could not register proxy passed in kuma-dp run")
	}

	info.connected = true

	return nil
}

func (d *DataplaneLifecycle) deregister(
	ctx context.Context,
	streamID core_xds.StreamID,
	proxyKey core_model.ResourceKey,
) {
	info, ok := d.proxyInfos.Load(proxyKey)
	if !ok {
		// proxy was not registered with this callback
		return
	}

	info.mtx.Lock()
	if info.deleted {
		info.mtx.Unlock()
		return
	}

	info.connected = false
	proxyType := info.proxyType
	info.mtx.Unlock()

	log := lifecycleLog.
		WithValues("proxyType", proxyType).
		WithValues("proxyKey", proxyKey).
		WithValues("streamID", streamID)

	// if delete immediately we're more likely to have a race condition
	// when DPP is connected to another CP but proxy resource in the store is deleted
	log.Info("waiting for deregister proxy", "waitFor", d.deregistrationDelay)
	<-time.After(d.deregistrationDelay)

	info.mtx.Lock()
	defer info.mtx.Unlock()

	if info.deleted {
		return
	}

	if info.connected {
		log.Info("no need to deregister proxy. It has already connected to this instance")
		return
	}

	if connected, err := d.proxyConnectedToAnotherCP(ctx, proxyType, proxyKey, log); err != nil {
		log.Error(err, "could not check if proxy connected to another CP")
		return
	} else if connected {
		return
	}

	log.Info("deregister proxy")
	if err := d.resManager.Delete(ctx, proxyResource(proxyType), store.DeleteBy(proxyKey)); err != nil {
		log.Error(err, "could not unregister proxy")
	}

	d.proxyInfos.Delete(proxyKey)
	info.deleted = true
}

func (d *DataplaneLifecycle) validateProxyKey(proxyKey core_model.ResourceKey, proxyResource core_model.Resource) error {
	if core_model.MetaToResourceKey(proxyResource.GetMeta()) != proxyKey {
		return errors.Errorf("proxyId %s does not match proxy resource %s", proxyKey, proxyResource.GetMeta())
	}
	return nil
}

func (d *DataplaneLifecycle) proxyConnectedToAnotherCP(
	ctx context.Context,
	pt mesh_proto.ProxyType,
	key core_model.ResourceKey,
	log logr.Logger,
) (bool, error) {
	insight := proxyInsight(pt)

	err := d.resManager.Get(ctx, insight, store.GetBy(key))
	switch {
	case store.IsResourceNotFound(err):
		// If insight is missing it most likely means that it was not yet created, so DP just connected and now leaving the mesh.
		log.Info("insight is missing. Safe to deregister the proxy")
		return false, nil
	case err != nil:
		return false, errors.Wrap(err, "could not get insight to determine if we can delete proxy object")
	}

	subs := insight.GetSpec().(generic.Insight).AllSubscriptions()
	if len(subs) == 0 {
		return false, nil
	}

	if sub := subs[len(subs)-1].(*mesh_proto.DiscoverySubscription); sub.ControlPlaneInstanceId != d.cpInstanceID {
		log.Info("no need to deregister proxy. It has already connected to another instance", "newCPInstanceID", sub.ControlPlaneInstanceId)
		return true, nil
	}

	return false, nil
}

func proxyResource(pt mesh_proto.ProxyType) core_model.Resource {
	switch pt {
	case mesh_proto.DataplaneProxyType:
		return core_mesh.NewDataplaneResource()
	case mesh_proto.IngressProxyType:
		return core_mesh.NewZoneIngressResource()
	default:
		return nil
	}
}

func proxyInsight(pt mesh_proto.ProxyType) core_model.Resource {
	switch pt {
	case mesh_proto.DataplaneProxyType:
		return core_mesh.NewDataplaneInsightResource()
	case mesh_proto.IngressProxyType:
		return core_mesh.NewZoneIngressInsightResource()
	default:
		return nil
	}
}
