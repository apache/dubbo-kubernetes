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
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
)

func DefaultDataplaneProxyBuilder(
	config dubbo_cp.Config,
	apiVersion core_xds.APIVersion,
) *DataplaneProxyBuilder {
	return &DataplaneProxyBuilder{
		Zone:       "",
		APIVersion: apiVersion,
	}
}

func DefaultIngressProxyBuilder(
	rt core_runtime.Runtime,
	apiVersion core_xds.APIVersion,
) *IngressProxyBuilder {
	return &IngressProxyBuilder{
		ResManager:        rt.ResourceManager(),
		apiVersion:        apiVersion,
		zone:              "",
		ingressTagFilters: nil,
	}
}

func DefaultDataplaneWatchdogFactory(
	rt core_runtime.Runtime,
	metadataTracker DataplaneMetadataTracker,
	dataplaneReconciler SnapshotReconciler,
	ingressReconciler SnapshotReconciler,
	egressReconciler SnapshotReconciler,
	envoyCpCtx *xds_context.ControlPlaneContext,
	apiVersion core_xds.APIVersion,
) (DataplaneWatchdogFactory, error) {
	config := rt.Config()

	dataplaneProxyBuilder := DefaultDataplaneProxyBuilder(
		config,
		apiVersion,
	)

	ingressProxyBuilder := DefaultIngressProxyBuilder(
		rt,
		apiVersion,
	)

	deps := DataplaneWatchdogDependencies{
		DataplaneProxyBuilder: dataplaneProxyBuilder,
		DataplaneReconciler:   dataplaneReconciler,
		IngressProxyBuilder:   ingressProxyBuilder,
		IngressReconciler:     ingressReconciler,
		EnvoyCpCtx:            envoyCpCtx,
		MetadataTracker:       metadataTracker,
		ResManager:            rt.ReadOnlyResourceManager(),
	}
	return NewDataplaneWatchdogFactory(
		10,
		deps,
	)
}
