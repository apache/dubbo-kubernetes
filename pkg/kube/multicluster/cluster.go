//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package multicluster

import (
	"crypto/sha256"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"go.uber.org/atomic"
)

type Cluster struct {
	ID     cluster.ID
	Client kube.Client

	kubeConfigSha [sha256.Size]byte

	stop chan struct{}

	initialSync        *atomic.Bool
	initialSyncTimeout *atomic.Bool
	closeOnce          sync.Once
}

func (c *Cluster) HasSynced() bool {
	// It could happen when a wrong credential provide, this cluster has no chance to run.
	// In this case, the `initialSyncTimeout` will never be set
	// In order not block dubbod start up, check close as well.
	if c.Closed() {
		return true
	}
	return c.initialSync.Load() || c.initialSyncTimeout.Load()
}

func NewRemoteCluster(id cluster.ID, client kube.Client, kubeConfigSha [sha256.Size]byte) *Cluster {
	return &Cluster{
		ID:                 id,
		Client:             client,
		kubeConfigSha:      kubeConfigSha,
		stop:               make(chan struct{}),
		initialSync:        atomic.NewBool(false),
		initialSyncTimeout: atomic.NewBool(false),
	}
}

func (c *Cluster) Start(syncTimeout time.Duration) {
	if syncTimeout > 0 {
		time.AfterFunc(syncTimeout, func() {
			if !c.initialSync.Load() && !c.Closed() {
				c.initialSyncTimeout.Store(true)
			}
		})
	}
	go func() {
		if c.Client.RunAndWait(c.stop) {
			c.initialSync.Store(true)
		}
	}()
}

func (c *Cluster) Close() {
	c.closeOnce.Do(func() {
		close(c.stop)
		if c.Client != nil {
			c.Client.Shutdown()
		}
	})
}

func (c *Cluster) Closed() bool {
	select {
	case <-c.stop:
		return true
	default:
		return false
	}
}
