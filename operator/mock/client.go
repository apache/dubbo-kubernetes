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

package mock

//import (
//	"github.com/apache/dubbo-kubernetes/operator/dubbo"
//)
//
//type Client struct {
//	// Members used to confirm certain configuration was used for instantiation
//	// (roughly map to the real clients WithX functions)
//	Confirm          bool
//	RepositoriesPath string
//
//	// repositories manager accessor
//	repositories *Repositories
//}
//
//func NewClient() *Client {
//	return &Client{repositories: NewRepositories()}
//}
//
//func (c *Client) Repositories() *Repositories {
//	return c.repositories
//}
//
//type Repositories struct {
//	// Members which record whether or not the various methods were invoked.
//	ListInvoked bool
//
//	all []dubbo.Repository
//}
//
//func NewRepositories() *Repositories {
//	return &Repositories{all: []dubbo.Repository{{Name: "default"}}}
//}
//
//func (r *Repositories) All() ([]dubbo.Repository, error) {
//	return r.all, nil
//}
//
//func (r *Repositories) List() ([]string, error) {
//	r.ListInvoked = true
//	var names []string
//	for _, v := range r.all {
//		names = append(names, v.Name)
//	}
//	return names, nil
//}
//
//func (r *Repositories) Add(name, url string) (string, error) {
//	r.all = append(r.all, dubbo.Repository{Name: name})
//	return "", nil
//}
//
//func (r *Repositories) Rename(old, new string) error {
//	for i, v := range r.all {
//		if v.Name == old {
//			v.Name = new
//			r.all[i] = v
//		}
//	}
//	return nil
//}
//
//func (r *Repositories) Remove(name string) error {
//	var repos []dubbo.Repository
//	for _, v := range r.all {
//		if v.Name == name {
//			continue
//		}
//		repos = append(repos, v)
//	}
//	r.all = repos
//	return nil
//}
