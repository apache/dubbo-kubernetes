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
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-retry"
	"go.uber.org/multierr"
	"sync"
	"time"
)

var log = core.Log.WithName("defaults")

func Setup(runtime runtime.Runtime) error {
	if !runtime.Config().IsFederatedZoneCP() { // Don't run defaults in Zone connected to global (it's done in Global)
		defaultsComponent := NewDefaultsComponent(
			runtime.Config().Defaults,
			runtime.ResourceManager(),
			runtime.ResourceStore(),
			runtime.Extensions(),
		)

		if err := runtime.Add(defaultsComponent); err != nil {
			return err
		}
	}

	return nil
}

func NewDefaultsComponent(
	config *dubbo_cp.Defaults,
	resManager core_manager.ResourceManager,
	resStore store.ResourceStore,
	extensions context.Context,
) component.Component {
	return &defaultsComponent{
		config:     config,
		resManager: resManager,
		resStore:   resStore,
		extensions: extensions,
	}
}

var _ component.Component = &defaultsComponent{}

type defaultsComponent struct {
	config     *dubbo_cp.Defaults
	resManager core_manager.ResourceManager
	resStore   store.ResourceStore
	extensions context.Context
}

func (d *defaultsComponent) NeedLeaderElection() bool {
	// If you spin many instances without default resources at once, many of them would create them, therefore only leader should create default resources.
	return true
}

func (d *defaultsComponent) Start(stop <-chan struct{}) error {
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	wg := &sync.WaitGroup{}
	errChan := make(chan error)

	if d.config.SkipMeshCreation {
		log.V(1).Info("skipping default Mesh creation because KUMA_DEFAULTS_SKIP_MESH_CREATION is set to true")
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// if after this time we cannot create a resource - something is wrong and we should return an error which will restart CP.
			err := retry.Do(ctx, retry.WithMaxDuration(10*time.Minute, retry.NewConstant(5*time.Second)), func(ctx context.Context) error {
				return retry.RetryableError(func() error {
					_, err := CreateMeshIfNotExist(ctx, d.resManager, d.extensions)
					return err
				}()) // retry all errors
			})
			if err != nil {
				// Retry this operation since on Kubernetes Mesh needs to be validated and set default values.
				// This code can execute before the control plane is ready therefore hooks can fail.
				errChan <- errors.Wrap(err, "could not create the default Mesh")
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(errChan)
	}()

	var errs error
	for {
		select {
		case <-stop:
			return errs
		case err := <-errChan:
			errs = multierr.Append(errs, err)
		case <-done:
			return errs
		}
	}
}
