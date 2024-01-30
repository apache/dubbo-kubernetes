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

package store

import (
	"context"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model/rest"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	util_yaml "github.com/apache/dubbo-kubernetes/pkg/util/yaml"
	"github.com/pkg/errors"

	"os"
)

func LoadResourcesFromFile(ctx context.Context, rs store.ResourceStore, fileName string) error {
	d, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}
	return LoadResources(ctx, rs, string(d))
}

func LoadResources(ctx context.Context, rs store.ResourceStore, inputs string) error {
	rawResources := util_yaml.SplitYAML(inputs)
	for i, rawResource := range rawResources {
		resource, err := rest.YAML.UnmarshalCore([]byte(rawResource))
		if err != nil {
			return errors.Wrapf(err, "failed to parse yaml %d", i)
		}
		curResource := resource.Descriptor().NewObject()
		create := false
		if err := rs.Get(ctx, curResource, store.GetByKey(resource.GetMeta().GetName(), resource.GetMeta().GetMesh())); err != nil {
			if !store.IsResourceNotFound(err) {
				return err
			}
			create = true
		}

		if create {
			err = rs.Create(ctx, resource, store.CreateByKey(resource.GetMeta().GetName(), resource.GetMeta().GetMesh()))
		} else {
			_ = curResource.SetSpec(resource.GetSpec())
			err = rs.Update(ctx, curResource)
		}
		if err != nil {
			return errors.Wrapf(err, "failed with entry %d meta: %s", i, resource.GetMeta())
		}
	}
	return nil
}
