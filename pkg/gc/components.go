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

package gc

import (
	"time"
)

import (
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func Setup(rt runtime.Runtime) error {
	if err := setupCollector(rt); err != nil {
		return err
	}
	return nil
}

func setupCollector(rt runtime.Runtime) error {
	if rt.Config().Environment != config_core.UniversalEnvironment || rt.Config().Mode == config_core.Global {
		// Dataplane GC is run only on Universal because on Kubernetes Dataplanes are bounded by ownership to Pods.
		// Therefore, on K8S offline dataplanes are cleaned up quickly enough to not run this.
		return nil
	}
	collector, err := NewCollector(
		rt.ResourceManager(),
		func() *time.Ticker { return time.NewTicker(1 * time.Minute) },
		rt.Config().Runtime.Universal.DataplaneCleanupAge.Duration,
	)
	if err != nil {
		return err
	}
	return rt.Add(collector)
}
