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

package model

import (
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/core/api"
)

type Model struct {
	provider    string
	internalKey string // internalKey is the internal model representation of different providers.
	supports    *ai.ModelSupports
}

func NewModel(provider string, internalKey string, supports *ai.ModelSupports) *Model {
	return &Model{
		provider:    provider,
		internalKey: internalKey,
		supports:    supports,
	}
}

// Key is the model query string of genkit registry.
func (m *Model) Key() string {
	return api.NewName(m.provider, m.internalKey)
}

func (m *Model) InternalKey() string {
	return m.internalKey
}

func (m *Model) Options() ai.ModelOptions {
	return ai.ModelOptions{
		Label:    m.internalKey,
		Supports: m.supports,
		Versions: []string{m.internalKey},
	}
}

type Embedder struct {
	config      map[string]any
	provider    string
	internalKey string
	dimensions  int
	supports    *ai.EmbedderSupports
}

func NewEmbedder(provider string, internalKey string, dimensions int, supports *ai.EmbedderSupports, config any) *Embedder {
	return &Embedder{
		config:      core.InferSchemaMap(config),
		provider:    provider,
		internalKey: internalKey,
		dimensions:  dimensions,
		supports:    supports,
	}
}

func (m *Embedder) Key() string {
	return api.NewName(m.provider, m.internalKey)
}

func (m *Embedder) InternalKey() string {
	return m.internalKey
}

func (m *Embedder) Options() *ai.EmbedderOptions {
	return &ai.EmbedderOptions{
		ConfigSchema: m.config,
		Label:        m.internalKey,
		Supports:     m.supports,
		Dimensions:   m.dimensions,
	}
}
