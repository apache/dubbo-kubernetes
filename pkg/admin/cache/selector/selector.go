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

package selector

import (
	"k8s.io/apimachinery/pkg/labels"
)

// Selector is an interface for selecting resources from cache
type Selector interface {
	AsLabelsSelector() labels.Selector

	ApplicationOptions() (Options, bool)
	ServiceNameOptions() (Options, bool)
	ServiceGroupOptions() (Options, bool)
	ServiceVersionOptions() (Options, bool)
}

// Options is an interface to represent possible options of a selector at a certain level(e.g. application, service)
type Options interface {
	Len() int
	Exist(str string) bool
}

func newOptions(strs ...string) Options {
	return options(strs)
}

// options is a slice of string, it implements Options interface
type options []string

func (o options) Len() int {
	return len(o)
}

func (o options) Exist(str string) bool {
	for _, s := range o {
		if s == str {
			return true
		}
	}
	return false
}
