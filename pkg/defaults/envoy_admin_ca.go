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

package defaults

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-retry"

	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	"github.com/apache/dubbo-kubernetes/pkg/core/user"
	"github.com/apache/dubbo-kubernetes/pkg/envoy/admin/tls"
	dubbo_log "github.com/apache/dubbo-kubernetes/pkg/log"
)

type EnvoyAdminCaDefaultComponent struct {
	ResManager manager.ResourceManager
	Extensions context.Context
}

var _ component.Component = &EnvoyAdminCaDefaultComponent{}

func (e *EnvoyAdminCaDefaultComponent) Start(stop <-chan struct{}) error {
	ctx, cancelFn := context.WithCancel(user.Ctx(context.Background(), user.ControlPlane))
	defer cancelFn()
	logger := dubbo_log.AddFieldsFromCtx(log, ctx, e.Extensions)
	errChan := make(chan error)
	go func() {
		errChan <- retry.Do(ctx, retry.WithMaxDuration(10*time.Minute, retry.NewConstant(5*time.Second)), func(ctx context.Context) error {
			if err := EnsureEnvoyAdminCaExist(ctx, e.ResManager, e.Extensions); err != nil {
				logger.V(1).Info("could not ensure that Envoy Admin CA exists. Retrying.", "err", err)
				return retry.RetryableError(err)
			}
			return nil
		})
	}()
	select {
	case <-stop:
		return nil
	case err := <-errChan:
		return err
	}
}

func (e EnvoyAdminCaDefaultComponent) NeedLeaderElection() bool {
	return true
}

func EnsureEnvoyAdminCaExist(
	ctx context.Context,
	resManager manager.ResourceManager,
	extensions context.Context,
) error {
	logger := dubbo_log.AddFieldsFromCtx(log, ctx, extensions)
	_, err := tls.LoadCA(ctx, resManager)
	if err == nil {
		logger.V(1).Info("Envoy Admin CA already exists. Skip creating Envoy Admin CA.")
		return nil
	}
	if !store.IsResourceNotFound(err) {
		return errors.Wrap(err, "error while loading envoy admin CA")
	}
	pair, err := tls.GenerateCA()
	if err != nil {
		return errors.Wrap(err, "could not generate envoy admin CA")
	}
	if err := tls.CreateCA(ctx, *pair, resManager); err != nil {
		return errors.Wrap(err, "could not create envoy admin CA")
	}
	logger.Info("Envoy Admin CA created")
	return nil
}
