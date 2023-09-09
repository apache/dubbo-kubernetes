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

package util

import (
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

// ErrInvalidApplicationName indicates the name did not pass function name validation.
type ErrInvalidApplicationName error

// ValidateApplicationName validates that the input name is a valid function name, ie. valid DNS-1035 label.
// It must consist of lower case alphanumeric characters or '-' and start with an alphabetic character and end with an alphanumeric character.
// (e.g. 'my-name',  or 'abc-1', regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?')
func ValidateApplicationName(name string) error {
	if errs := validation.IsDNS1035Label(name); len(errs) > 0 {
		// In case of invalid name the error is this:
		// "a DNS-1035 label must consist of lower case alphanumeric characters or '-',
		// start with an alphabetic character,
		// and end with an alphanumeric character".
		// Let's reuse it for our purposes, ie. replace "a DNS-1035 label" substring with "Function name" and the actual function name
		return ErrInvalidApplicationName(errors.New(strings.Replace(strings.Join(errs, ""), "a DNS-1035 label", fmt.Sprintf("Function name '%v'", name), 1)))
	}

	return nil
}
