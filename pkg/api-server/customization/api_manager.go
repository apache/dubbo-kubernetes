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

package customization

import (
	"github.com/emicklei/go-restful/v3"
)

type APIInstaller interface {
	Install(container *restful.Container)
}

type APIManager interface {
	APIInstaller
	Add(ws *restful.WebService)
	AddFilter(filter restful.FilterFunction)
}

type APIList struct {
	list    []*restful.WebService
	filters []restful.FilterFunction
}

func NewAPIList() *APIList {
	return &APIList{
		list:    []*restful.WebService{},
		filters: []restful.FilterFunction{},
	}
}

func (c *APIList) Add(ws *restful.WebService) {
	c.list = append(c.list, ws)
}

func (c *APIList) AddFilter(f restful.FilterFunction) {
	c.filters = append(c.filters, f)
}

func (c *APIList) Install(container *restful.Container) {
	for _, ws := range c.list {
		container.Add(ws)
	}

	for _, f := range c.filters {
		container.Filter(f)
	}
}
