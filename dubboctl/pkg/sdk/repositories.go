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

package sdk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Repositories struct {
	client *Client
	path   string
}

func newRepositories(client *Client) *Repositories {
	return &Repositories{
		client: client,
		path:   client.repositoriesPath,
	}
}

func (r *Repositories) All() (repos []Repository, err error) {
	var repo Repository

	if repo, err = NewRepository("", ""); err != nil {
		return
	}
	repos = append(repos, repo)

	if r.path == "" {
		return
	}

	if _, err = os.Stat(r.path); os.IsNotExist(err) {
		return repos, nil
	}

	ff, err := os.ReadDir(r.path)
	if err != nil {
		return
	}
	for _, f := range ff {
		if !f.IsDir() || strings.HasPrefix(f.Name(), ".") {
			continue
		}
		var abspath string
		abspath, err = filepath.Abs(r.path)
		if err != nil {
			return
		}
		if repo, err = NewRepository("", "file://"+filepath.ToSlash(abspath)+"/"+f.Name()); err != nil {
			return
		}
		repos = append(repos, repo)
	}
	return
}

func (r *Repositories) Get(name string) (repo Repository, err error) {
	all, err := r.All()
	if err != nil {
		return
	}
	if len(all) == 0 { // should not be possible because embedded always exists.
		err = errors.New("internal error: no repositories loaded")
		return
	}

	if name == DefaultRepositoryName {
		repo = all[0]
		return
	}

	for _, v := range all {
		if v.Name == name {
			repo = v
			return
		}
	}
	return repo, fmt.Errorf("repository not found")
}
