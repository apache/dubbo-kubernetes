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

package collection

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/config"
	"github.com/apache/dubbo-kubernetes/operator/pkg/schema"
)

type Schemas struct {
	byCollection map[config.GroupVersionKind]schema.Schema
	byAddOrder   []schema.Schema
}

func (s Schemas) FindByGroupVersionAliasesKind(gvk config.GroupVersionKind) (schema.Schema, bool) {
	for _, rs := range s.byAddOrder {
		for _, va := range rs.GroupVersionAliasKinds() {
			if va == gvk {
				return rs, true
			}
		}
	}
	return nil, false
}

type SchemasBuilder struct {
	schemas Schemas
}

func NewSchemasBuilder() *SchemasBuilder {
	s := Schemas{
		byCollection: make(map[config.GroupVersionKind]schema.Schema),
	}
	return &SchemasBuilder{schemas: s}
}

func (b *SchemasBuilder) Add(s schema.Schema) error {
	if _, found := b.schemas.byCollection[s.GroupVersionKind()]; found {
		return fmt.Errorf("collection already exists: %v", s.GroupVersionKind())
	}
	b.schemas.byCollection[s.GroupVersionKind()] = s
	b.schemas.byAddOrder = append(b.schemas.byAddOrder, s)
	return nil
}

func (b *SchemasBuilder) MustAdd(s schema.Schema) *SchemasBuilder {
	if err := b.Add(s); err != nil {
		panic(fmt.Sprintf("SchemasBuilder.MustAdd: %v", err))
	}
	return b
}

func (b *SchemasBuilder) Build() Schemas {
	s := b.schemas
	b.schemas = Schemas{}
	return s
}
