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
	"github.com/apache/dubbo-kubernetes/operator/pkg/filesystem"
	"path"
)

// Template is a function project template.
// It can be used to instantiate new function project.
type Template interface {
	// Name of this template.
	Name() string
	// Runtime for which this template applies.
	Runtime() string
	// Repository within which this template is contained.  Value is set to the
	// currently effective name of the repository, which may vary. It is user-
	// defined when the repository is added, and can be set to "default" when
	// the client is loaded in single repo mode. I.e. not canonical.
	Repository() string
	// Fullname is a calculated field of [repo]/[name] used
	// to uniquely reference a template which may share a name
	// with one in another repository.
	Fullname() string
	// Write updates fields of function f and writes project files to path pointed by f.Root.
	Write(ctx context.Context, f *Dubbo) error
}

// template default implementation
type template struct {
	name       string
	runtime    string
	repository string
	fs         filesystem.Filesystem
}

func (t template) Write(ctx context.Context, f *Dubbo) error {
	mask := func(p string) bool {
		_, f := path.Split(p)
		return f == templateManifest
	}

	return filesystem.CopyFromFS(".", f.Root, filesystem.NewMaskingFS(mask, t.fs)) // copy everything but manifest.yaml
}

func (t template) Name() string {
	return t.name
}

func (t template) Runtime() string {
	return t.runtime
}

func (t template) Repository() string {
	return t.repository
}

func (t template) Fullname() string {
	return t.repository + "/" + t.name
}
