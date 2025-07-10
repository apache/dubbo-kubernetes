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

package mesh

import (
	"regexp"

	"github.com/apache/dubbo-kubernetes/pkg/common/validators"
	"github.com/apache/dubbo-kubernetes/pkg/core/resource/model"
)

var (
	backwardCompatRegexp = regexp.MustCompile(`^[0-9a-z-_.]*$`)
	backwardCompatErrMsg = "invalid characters. Valid characters are numbers, lowercase latin letters and '-', '_', '.' symbols."
)

// ValidateMesh checks that resource's mesh matches the old regex (with '_'). Even if user creates entirely new resource,
// we can't check resource's mesh against the new regex, because Mesh resource itself can be old and contain '_' in its name.
// All new Mesh resources will have their name validated against new regex.
func ValidateMesh(mesh string, scope model.ResourceScope) validators.ValidationError {
	var err validators.ValidationError
	if scope == model.ScopeMesh {
		err.AddError("mesh", validateIdentifier(mesh, backwardCompatRegexp, backwardCompatErrMsg))
	}
	return err
}

func validateIdentifier(identifier string, r *regexp.Regexp, errMsg string) validators.ValidationError {
	var err validators.ValidationError
	switch {
	case identifier == "":
		err.AddViolation("", "cannot be empty")
	case len(identifier) > 253:
		err.AddViolation("", "value length must less or equal 253")
	case !r.MatchString(identifier):
		err.AddViolation("", errMsg)
	}
	return err
}
