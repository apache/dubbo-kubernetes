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

	"github.com/apache/dubbo-kubernetes/pkg/core/model"
)

// The Pagination Store is handling only the pagination functionality in the List.
// This is an in-memory operation and offloads this from the persistent stores (k8s, postgres etc.)
// Two reasons why this is needed:
// * There is no filtering + pagination on the native K8S database
// * On Postgres, we keep the object in a column as a string. We would have to use JSON column type and convert it to native SQL queries.
//
// The in-memory filtering has been tested with 10,000 Dataplanes and proved to be fast enough, although not that efficient.
func NewPaginationStore(delegate ResourceStore) ResourceStore {
	return &paginationStore{
		delegate: delegate,
	}
}

// TODO implement pagination in store
type paginationStore struct {
	delegate ResourceStore
}

func (p *paginationStore) Create(ctx context.Context, resource model.Resource, optionsFunc ...CreateOptionsFunc) error {
	return p.delegate.Create(ctx, resource, optionsFunc...)
}

func (p *paginationStore) Update(ctx context.Context, resource model.Resource, optionsFunc ...UpdateOptionsFunc) error {
	return p.delegate.Update(ctx, resource, optionsFunc...)
}

func (p *paginationStore) Delete(ctx context.Context, resource model.Resource, optionsFunc ...DeleteOptionsFunc) error {
	return p.delegate.Delete(ctx, resource, optionsFunc...)
}

func (p *paginationStore) Get(ctx context.Context, resource model.Resource, optionsFunc ...GetOptionsFunc) error {
	return p.delegate.Get(ctx, resource, optionsFunc...)
}

func (p *paginationStore) List(ctx context.Context, list model.ResourceList, optionsFunc ...ListOptionsFunc) error {
	return p.delegate.List(ctx, list, optionsFunc...)
}

var _ ResourceStore = &paginationStore{}
