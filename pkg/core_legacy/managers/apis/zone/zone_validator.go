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

package zone

import (
	"context"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/validators"
)

type Validator struct {
	Store store.ResourceStore
}

func (v *Validator) ValidateDelete(ctx context.Context, name string) error {
	zi := system.NewZoneInsightResource()
	validationErr := &validators.ValidationError{}
	if err := v.Store.Get(ctx, zi, store.GetByKey(name, model.NoMesh)); err != nil {
		if store.IsResourceNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "unable to get ZoneInsight")
	}
	if zi.Spec.IsOnline() {
		validationErr.AddViolation("zone", "unable to delete Zone, Zone CP is still connected, please shut it down first")
		return validationErr
	}
	return nil
}
