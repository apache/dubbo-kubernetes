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

package dubbo

import (
	"context"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util"
	"strings"
)

// Templates Manager
type Templates struct {
	client *Client
}

// newTemplates manager
// Includes a back-reference to client (logic tree root) such
// that the templates manager has full access to the API for
// use in its implementations.
func newTemplates(client *Client) *Templates {
	return &Templates{client: client}
}

// List the full name of templates available for the runtime.
// Full name is the optional repository prefix plus the template's repository
// local name.  Default templates grouped first sans prefix.
func (t *Templates) List(runtime string) ([]string, error) {
	var names []string
	extended := util.NewSortedSet()

	rr, err := t.client.Repositories().All()
	if err != nil {
		return []string{}, err
	}

	for _, r := range rr {
		tt, err := r.Templates(runtime)
		if err != nil {
			return []string{}, err
		}
		for _, t := range tt {
			if r.Name == DefaultRepositoryName {
				names = append(names, t.Name())
			} else {
				extended.Add(t.Fullname())
			}
		}
	}
	return append(names, extended.Items()...), nil
}

func (t *Templates) Get(runtime, fullname string) (Template, error) {
	var (
		template Template
		repo     Repository
		err      error
	)
	repoName, tplName := splitTemplateFullname(fullname)

	// Get specified repository
	repo, err = t.client.Repositories().Get(repoName)
	if err != nil {
		return template, err
	}

	return repo.Template(runtime, tplName)
}

// splits a template reference into its constituent parts: repository name
// and template name.
// The form '[repo]/[name]'.  The reposititory name and slash prefix are
// optional, in which case DefaultRepositoryName is returned.
func splitTemplateFullname(name string) (repoName, tplName string) {
	// Split into repo and template names.
	// Defaults when unprefixed to DefaultRepositoryName
	cc := strings.Split(name, "/")
	if len(cc) == 1 {
		repoName = DefaultRepositoryName
		tplName = name
	} else {
		repoName = cc[0]
		tplName = cc[1]
	}
	return
}

// Write a function's template to disk.
// Returns a function which may have been modified dependent on the content
// of the template (which can define default function fields, builders,
// build packs, etc)
func (t *Templates) Write(f *Dubbo) error {
	// Ensure the function itself is syntactically valid
	if err := f.Validate(); err != nil {
		return err
	}

	// The function's Template
	template, err := t.Get(f.Runtime, f.Template)
	if err != nil {
		return err
	}

	return template.Write(context.TODO(), f)
}
