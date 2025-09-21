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

package controller

import (
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/aggregate"
)

type Options struct {
	KubernetesAPIQPS   float32
	KubernetesAPIBurst int
	DomainSuffix       string
	// XDSUpdater will push changes to the xDS server.
	XDSUpdater model.XDSUpdater
	// MeshNetworksWatcher observes changes to the mesh networks config.
	MeshNetworksWatcher mesh.NetworksWatcher
	// MeshWatcher observes changes to the mesh config
	MeshWatcher           meshwatcher.WatcherCollection
	ClusterID             cluster.ID
	ClusterAliases        map[string]string
	SystemNamespace       string
	MeshServiceController *aggregate.Controller
	KrtDebugger           *krt.DebugHandler
}
