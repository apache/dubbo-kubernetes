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
	system_proto "github.com/apache/dubbo-kubernetes/api/system/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"

	"github.com/pkg/errors"
	"os"
)

type dynamicLoader struct {
	secretManager manager.ReadOnlyResourceManager
}

var _ Loader = &dynamicLoader{}

func NewDataSourceLoader(secretManager manager.ReadOnlyResourceManager) Loader {
	return &dynamicLoader{
		secretManager: secretManager,
	}
}

func (l *dynamicLoader) Load(ctx context.Context, mesh string, source *system_proto.DataSource) ([]byte, error) {
	var data []byte
	var err error
	switch source.GetType().(type) {
	case *system_proto.DataSource_Secret:
		data, err = l.loadSecret(ctx, mesh, source.GetSecret())
	case *system_proto.DataSource_Inline:
		data, err = source.GetInline().GetValue(), nil
	case *system_proto.DataSource_InlineString:
		data, err = []byte(source.GetInlineString()), nil
	case *system_proto.DataSource_File:
		data, err = os.ReadFile(source.GetFile())
	default:
		return nil, errors.New("unsupported type of the DataSource")
	}
	if err != nil {
		return nil, errors.Wrap(err, "could not load data")
	}
	return data, nil
}

func (l *dynamicLoader) loadSecret(ctx context.Context, mesh string, secret string) ([]byte, error) {
	if l.secretManager == nil {
		return nil, errors.New("no resource manager")
	}
	resource := system.NewSecretResource()
	if err := l.secretManager.Get(ctx, resource, core_store.GetByKey(secret, mesh)); err != nil {
		return nil, err
	}
	return resource.Spec.GetData().GetValue(), nil
}
