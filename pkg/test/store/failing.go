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
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

type FailingStore struct {
	Err error
}

var _ core_store.ResourceStore = &FailingStore{}

func (f *FailingStore) Create(context.Context, model.Resource, ...core_store.CreateOptionsFunc) error {
	return f.Err
}

func (f *FailingStore) Update(context.Context, model.Resource, ...core_store.UpdateOptionsFunc) error {
	return f.Err
}

func (f *FailingStore) Delete(context.Context, model.Resource, ...core_store.DeleteOptionsFunc) error {
	return f.Err
}

func (f *FailingStore) Get(context.Context, model.Resource, ...core_store.GetOptionsFunc) error {
	return f.Err
}

func (f *FailingStore) List(context.Context, model.ResourceList, ...core_store.ListOptionsFunc) error {
	return f.Err
}
