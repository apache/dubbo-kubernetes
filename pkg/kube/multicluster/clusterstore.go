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
	"sync"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
)

type ClusterStore struct {
	sync.RWMutex
	clusters map[cluster.ID]*Cluster
}

func NewClusterStore() *ClusterStore {
	return &ClusterStore{
		clusters: make(map[cluster.ID]*Cluster),
	}
}

func (c *ClusterStore) Get(id cluster.ID) (*Cluster, bool) {
	c.RLock()
	defer c.RUnlock()
	cluster, ok := c.clusters[id]
	return cluster, ok
}

func (c *ClusterStore) Store(cluster *Cluster) {
	c.Lock()
	defer c.Unlock()
	c.clusters[cluster.ID] = cluster
}

func (c *ClusterStore) Delete(id cluster.ID) (*Cluster, bool) {
	c.Lock()
	defer c.Unlock()
	cluster, ok := c.clusters[id]
	if ok {
		delete(c.clusters, id)
	}
	return cluster, ok
}

func (c *ClusterStore) IDs() sets.String {
	c.RLock()
	defer c.RUnlock()
	out := sets.New[string]()
	for id := range c.clusters {
		out.Insert(string(id))
	}
	return out
}

func (c *ClusterStore) HasSynced() bool {
	c.RLock()
	defer c.RUnlock()
	for _, cluster := range c.clusters {
		if !cluster.HasSynced() {
			return false
		}
	}
	return true
}
