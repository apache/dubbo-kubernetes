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
	"errors"
	"fmt"
	"time"
)

var (
	// ErrTemplateRepoDoesNotExist is a sentinel error if a template repository responds with 404 status code
	ErrTemplateRepoDoesNotExist  = errors.New("template repo does not exist")
	ErrEnvironmentNotFound       = errors.New("environment not found")
	ErrMismatchedName            = errors.New("name passed the function source")
	ErrNameRequired              = errors.New("name required")
	ErrNotBuilt                  = errors.New("not built")
	ErrNotRunning                = errors.New("function not running")
	ErrRepositoriesNotDefined    = errors.New("custom template repositories location not specified")
	ErrRepositoryNotFound        = errors.New("repository not found")
	ErrRootRequired              = errors.New("function root path is required")
	ErrRuntimeNotFound           = errors.New("language runtime not found")
	ErrRuntimeRequired           = errors.New("language runtime required")
	ErrTemplateMissingRepository = errors.New("template name missing repository prefix")
	ErrTemplateNotFound          = errors.New("template not found")
	ErrTemplatesNotFound         = errors.New("templates path (runtimes) not found")
	ErrContextCanceled           = errors.New("the operation was canceled")
)

// ErrNotInitialized indicates that a function is uninitialized
type ErrNotInitialized struct {
	Path string
}

func NewErrNotInitialized(path string) error {
	return &ErrNotInitialized{Path: path}
}

func (e ErrNotInitialized) Error() string {
	if e.Path == "" {
		return "appilcation is not initialized"
	}
	return fmt.Sprintf("'%s' does not contain an initialized appilcation", e.Path)
}

// ErrRuntimeNotRecognized indicates a runtime which is not in the set of
// those known.
type ErrRuntimeNotRecognized struct {
	Runtime string
}

func (e ErrRuntimeNotRecognized) Error() string {
	return fmt.Sprintf("the %q runtime is not recognized", e.Runtime)
}

// ErrRunnerNotImplemented indicates the feature is not available for the
// requested runtime.
type ErrRunnerNotImplemented struct {
	Runtime string
}

func (e ErrRunnerNotImplemented) Error() string {
	return fmt.Sprintf("the %q runtime may only be run containerized.", e.Runtime)
}

type ErrRunTimeout struct {
	Timeout time.Duration
}

func (e ErrRunTimeout) Error() string {
	return fmt.Sprintf("timed out waiting for function to be ready for %s", e.Timeout)
}
