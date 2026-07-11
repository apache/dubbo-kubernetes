//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/validation/agent"
)

var (
	// EmptyValidate is a Validate that does nothing and returns no error.
	EmptyValidate = func(config.Config) (Warning, error) {
		return nil, nil
	}
	validateFuncs = make(map[string]ValidateFunc)
)

type Warning = agent.Warning

// ValidateFunc defines a validation func for an API proto.
type ValidateFunc func(config config.Config) (Warning, error)

func IsValidateFunc(name string) bool {
	return GetValidateFunc(name) != nil
}

// RegisterValidateFunc registers a validation function by name so that codegen
// and the validation webhook can look it up. It returns the function to allow
// assignment-style registration:
//
//	var ValidateFoo = RegisterValidateFunc("ValidateFoo", func(cfg config.Config) (Warning, error) { ... })
func RegisterValidateFunc(name string, f ValidateFunc) ValidateFunc {
	// Wrap the original validate function with an extra validate function if applicable.
	validate := validateFunc(f)
	validateFuncs[name] = validate
	return validate
}

func validateFunc(f ValidateFunc) ValidateFunc {
	return func(cfg config.Config) (Warning, error) {
		warn, err := f(cfg)
		// Validate name and namespace regardless of the schema.
		if err2 := ValidateMetadata(cfg); err2 != nil {
			err = AppendErrors(err, err2)
		}
		return warn, err
	}
}

// GetValidateFunc returns the validation function with the given name, or null if it does not exist.
func GetValidateFunc(name string) ValidateFunc {
	return validateFuncs[name]
}
