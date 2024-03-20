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

package datasource

import (
	system_proto "github.com/apache/dubbo-kubernetes/api/system/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/validators"
)

func Validate(source *system_proto.DataSource) validators.ValidationError {
	verr := validators.ValidationError{}
	if source == nil || source.Type == nil {
		verr.AddViolation("", "data source has to be chosen. Available sources: secret, file, inline")
	}
	switch source.GetType().(type) {
	case *system_proto.DataSource_Secret:
		if source.GetSecret() == "" {
			verr.AddViolation("secret", "cannot be empty")
		}
	case *system_proto.DataSource_Inline:
		if len(source.GetInline().GetValue()) == 0 {
			verr.AddViolation("inline", "cannot be empty")
		}
	case *system_proto.DataSource_File:
		if source.GetFile() == "" {
			verr.AddViolation("file", "cannot be empty")
		}
	}
	return verr
}
