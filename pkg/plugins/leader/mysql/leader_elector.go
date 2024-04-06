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

package mysql

import (
	"sync/atomic"
	"time"
)

import (
	"gorm.io/gorm"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	util_channels "github.com/apache/dubbo-kubernetes/pkg/util/channels"
)

var log = core.Log.WithName("mysql-leader")

const (
	dubboLockName = "dubbo-cp-lock"
	backoffTime   = 5 * time.Second
)

type mysqlLeaderElector struct {
	leader     int32
	lockClient *MysqlLock
	callbacks  []component.LeaderCallbacks
}

func (n *mysqlLeaderElector) IsLeader() bool {
	return atomic.LoadInt32(&(n.leader)) == 1
}

func (n *mysqlLeaderElector) AddCallbacks(callbacks component.LeaderCallbacks) {
	n.callbacks = append(n.callbacks, callbacks)
}

func (n *mysqlLeaderElector) Start(stop <-chan struct{}) {
	log.Info("waiting for lock")
	retries := 0
	for {
		acquiredLock, err := n.lockClient.TryLock()
		if err != nil {
			if retries >= 3 {
				log.Error(err, "error waiting for lock", "retries", retries)
			} else {
				log.V(1).Info("error waiting for lock", "err", err, "retries", retries)
			}
			retries += 1
		} else {
			retries = 0
			if acquiredLock {
				n.leaderAcquired()
				n.lockClient.unLock()
				n.leaderLost()
			}
		}
		if util_channels.IsClosed(stop) {
			break
		}
		time.Sleep(backoffTime)
	}
	log.Info("Leader Elector stopped")
}

func NewMysqlLeaderElector(connect *gorm.DB) component.LeaderElector {
	lock := NewLock(dubboLockName, connect)
	return &mysqlLeaderElector{
		lockClient: lock,
	}
}

func (n *mysqlLeaderElector) setLeader(leader bool) {
	var value int32 = 0
	if leader {
		value = 1
	}
	atomic.StoreInt32(&n.leader, value)
}

func (n *mysqlLeaderElector) leaderAcquired() {
	n.setLeader(true)
	for _, callback := range n.callbacks {
		callback.OnStartedLeading()
	}
}

func (n *mysqlLeaderElector) leaderLost() {
	n.setLeader(false)
	for _, callback := range n.callbacks {
		callback.OnStoppedLeading()
	}
}
