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

package datasource

import (
	"context"
)

import (
	"github.com/pkg/errors"
)

import (
	system_proto "github.com/apache/dubbo-kubernetes/api/system/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

type staticLoader struct {
	secrets map[model.ResourceKey]*system.SecretResource
}

var _ Loader = &staticLoader{}

// NewStaticLoader returns a loader that supports predefined list of secrets
// This implementation is more performant if than dynamic if we already have the list of all secrets
// because we can avoid I/O operations.
func NewStaticLoader(secrets []*system.SecretResource) Loader {
	loader := staticLoader{
		secrets: map[model.ResourceKey]*system.SecretResource{},
	}

	for _, secret := range secrets {
		loader.secrets[model.MetaToResourceKey(secret.GetMeta())] = secret
	}

	return &loader
}

func (s *staticLoader) Load(_ context.Context, mesh string, source *system_proto.DataSource) ([]byte, error) {
	var data []byte
	var err error
	switch source.GetType().(type) {
	case *system_proto.DataSource_Secret:
		data, err = s.loadSecret(mesh, source.GetSecret())
	case *system_proto.DataSource_Inline:
		data, err = source.GetInline().GetValue(), nil
	case *system_proto.DataSource_InlineString:
		data, err = []byte(source.GetInlineString()), nil
	default:
		return nil, errors.New("unsupported type of the DataSource")
	}
	if err != nil {
		return nil, errors.Wrap(err, "could not load data")
	}
	return data, nil
}

func (s *staticLoader) loadSecret(mesh string, name string) ([]byte, error) {
	key := model.ResourceKey{
		Mesh: mesh,
		Name: name,
	}

	secret := s.secrets[key]
	if secret == nil {
		return nil, core_store.ErrorResourceNotFound(system.SecretType, name, mesh)
	}
	return secret.Spec.GetData().GetValue(), nil
}
