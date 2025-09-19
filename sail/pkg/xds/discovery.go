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

package xds

import (
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"go.uber.org/atomic"
	"k8s.io/klog/v2"
	"time"
)

type DiscoveryServer struct {
	Env                *model.Environment
	serverReady        atomic.Bool
	DiscoveryStartTime time.Time
	ClusterAliases     map[cluster.ID]cluster.ID
	Cache              model.XdsCache
	pushQueue          *PushQueue
	krtDebugger        *krt.DebugHandler
	InboundUpdates     *atomic.Int64
	CommittedUpdates   *atomic.Int64
}

func NewDiscoveryServer(env *model.Environment, clusterAliases map[string]string, debugger *krt.DebugHandler) *DiscoveryServer {
	out := &DiscoveryServer{
		Env:         env,
		Cache:       env.Cache,
		krtDebugger: debugger,
	}
	out.ClusterAliases = make(map[cluster.ID]cluster.ID)
	for alias := range clusterAliases {
		out.ClusterAliases[cluster.ID(alias)] = cluster.ID(clusterAliases[alias])
	}
	return out
}

func (s *DiscoveryServer) CachesSynced() {
	klog.Infof("All caches have been synced up in %v, marking server ready", time.Since(s.DiscoveryStartTime))
	s.serverReady.Store(true)
}

func (s *DiscoveryServer) Shutdown() {
	s.pushQueue.ShutDown()
}
