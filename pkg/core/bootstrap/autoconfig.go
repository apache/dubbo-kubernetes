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

package bootstrap

import (
	"os"
	"path"
)

import (
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-kubernetes/pkg/core"
)

var autoconfigureLog = core.Log.WithName("bootstrap").WithName("auto-configure")

func autoconfigure(cfg *dubbo_cp.Config) error {
	return nil
}

type workDir string

func (w workDir) Open(name string) (*os.File, error) {
	if err := os.MkdirAll(string(w), 0o700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	return os.OpenFile(path.Join(string(w), name), os.O_RDWR|os.O_CREATE, 0o600)
}
