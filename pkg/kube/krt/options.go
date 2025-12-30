//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package krt

type BuilderOption func(opt CollectionOption) OptionsBuilder

type OptionsBuilder struct {
	namePrefix string
	stop       <-chan struct{}
	debugger   *DebugHandler
}

func NewOptionsBuilder(stop <-chan struct{}, namePrefix string, debugger *DebugHandler) OptionsBuilder {
	return OptionsBuilder{
		namePrefix: namePrefix,
		stop:       stop,
		debugger:   debugger,
	}
}

func (k OptionsBuilder) Stop() <-chan struct{} {
	return k.stop
}

func (k OptionsBuilder) With(opts ...CollectionOption) []CollectionOption {
	return append([]CollectionOption{WithDebugging(k.debugger), WithStop(k.stop)}, opts...)
}

func (k OptionsBuilder) Debugger() *DebugHandler {
	return k.debugger
}

func (k OptionsBuilder) WithName(n string) []CollectionOption {
	name := n
	if k.namePrefix != "" {
		name = k.namePrefix + "/" + name
	}
	return []CollectionOption{WithDebugging(k.debugger), WithStop(k.stop), WithName(name)}
}

func WithStop(stop <-chan struct{}) CollectionOption {
	return func(c *collectionOptions) {
		c.stop = stop
	}
}

func WithName(name string) CollectionOption {
	return func(c *collectionOptions) {
		c.name = name
	}
}

func WithObjectAugmentation(fn func(o any) any) CollectionOption {
	return func(c *collectionOptions) {
		c.augmentation = fn
	}
}

func WithDebugging(handler *DebugHandler) CollectionOption {
	return func(c *collectionOptions) {
		c.debugger = handler
	}
}
