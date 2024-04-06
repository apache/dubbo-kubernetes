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

package config

import (
	"fmt"
	"io"
	"os"
)

type Deprecation struct {
	Env             string
	EnvMsg          string
	ConfigValuePath func(cfg Config) (string, bool)
	ConfigValueMsg  string
}

func PrintDeprecations(deprecations []Deprecation, cfg Config, out io.Writer) {
	for _, d := range deprecations {
		if _, ok := os.LookupEnv(d.Env); ok {
			_, _ = fmt.Fprintf(out, "Deprecated: %v. %v\n", d.Env, d.EnvMsg)
		}
		if path, exist := d.ConfigValuePath(cfg); exist {
			_, _ = fmt.Fprintf(out, "Deprecated: %v. %v\n", path, d.ConfigValueMsg)
		}
	}
}
