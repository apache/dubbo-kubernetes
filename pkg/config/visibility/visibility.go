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

package visibility

import (
	"fmt"

	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
)

type Instance string

const (
	Private Instance = "."
	Public  Instance = "*"
	None    Instance = "~"
)

func (v Instance) Validate() (errs error) {
	switch v {
	case Private, Public:
		return nil
	case None:
		return fmt.Errorf("exportTo ~ (none) is not allowed for Istio configuration objects")
	default:
		if !labels.IsDNS1123Label(string(v)) {
			return fmt.Errorf("only .,*, or a valid DNS 1123 label is allowed as exportTo entry")
		}
	}
	return nil
}
