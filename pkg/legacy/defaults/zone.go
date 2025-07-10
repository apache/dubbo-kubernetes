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
)

import (
	"github.com/go-logr/logr"

	"github.com/pkg/errors"

	"github.com/sethvargo/go-retry"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
)

type ZoneDefaultComponent struct {
	ResManager manager.ResourceManager
	Extensions context.Context
	ZoneName   string
}

var _ component.Component = &ZoneDefaultComponent{}

func (e *ZoneDefaultComponent) Start(stop <-chan struct{}) error {
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	errChan := make(chan error)
	go func() {
		errChan <- retry.Do(ctx, retry.WithMaxDuration(10*time.Minute, retry.NewConstant(5*time.Second)), func(ctx context.Context) error {
			if err := EnsureOnlyOneZoneExists(ctx, e.ResManager, e.ZoneName, log); err != nil {
				log.V(1).Info("could not ensure that Zone exists. Retrying.", "err", err)
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

func (e ZoneDefaultComponent) NeedLeaderElection() bool {
	return true
}

func EnsureOnlyOneZoneExists(
	ctx context.Context,
	resManager manager.ResourceManager,
	zoneName string,
	logger logr.Logger,
) error {
	logger.Info("ensuring Zone resource exists", "name", zoneName)
	zones := &system.ZoneResourceList{}
	if err := resManager.List(ctx, zones); err != nil {
		return errors.Wrap(err, "cannot list zones")
	}
	exists := false
	for _, zone := range zones.Items {
		if zone.GetMeta().GetName() == zoneName {
			exists = true
		} else {
			logger.Info("detected Zone resource with different name than Zone CP name. Deleting. This might happen if you change the name of the Zone CP", "name", zoneName)
			if err := resManager.Delete(ctx, zone, store.DeleteByKey(zone.GetMeta().GetName(), model.NoMesh)); err != nil {
				return errors.Wrap(err, "cannot delete old zone")
			}
		}
	}
	if !exists {
		logger.Info("creating Zone resource", "name", zoneName)
		zone := system.NewZoneResource()
		if err := resManager.Create(ctx, zone, store.CreateByKey(zoneName, model.NoMesh)); err != nil {
			return err
		}
	}
	return nil
}
