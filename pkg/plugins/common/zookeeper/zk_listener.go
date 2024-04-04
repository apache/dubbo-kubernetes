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

package zookeeper

import (
	"sync"
)

import (
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

	"github.com/go-logr/logr"

	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/plugins/resources/zookeeper"
)

type zkListener struct {
	Client        *gxzookeeper.ZookeeperClient
	pathMapLock   sync.Mutex
	pathMap       map[string]*atomic.Int32
	wg            sync.WaitGroup
	err           chan error
	notifications chan *Notification
	stop          chan struct{}
}

func NewListener(cfg zookeeper.ZookeeperStoreConfig, log logr.Logger) (Listener, error) {
	return nil, nil
}

func (z *zkListener) Error() <-chan error {
	return z.err
}

func (z *zkListener) Notify() chan *Notification {
	return z.notifications
}

func (z *zkListener) Close() error {
	close(z.stop)
	z.wg.Wait()
	return nil
}
