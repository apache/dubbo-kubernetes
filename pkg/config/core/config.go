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

package core

import (
	"github.com/pkg/errors"
)

type EnvironmentType = string

const (
	KubernetesEnvironment EnvironmentType = "kubernetes"
	UniversalEnvironment  EnvironmentType = "universal"
)

// Control Plane mode

type CpMode = string

const (
	// Deprecated: use zone
	Standalone CpMode = "standalone"
	Zone       CpMode = "zone"
	Global     CpMode = "global"
)

// ValidateCpMode to check modes of dubbo-cp
func ValidateCpMode(mode CpMode) error {
	if mode != Standalone && mode != Zone && mode != Global {
		return errors.Errorf("invalid mode. Available modes: %s, %s, %s", Standalone, Zone, Global)
	}
	return nil
}
