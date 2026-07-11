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
	"fmt"
	"net/url"

	"google.golang.org/protobuf/types/known/durationpb"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	"github.com/hashicorp/go-multierror"
	typev1alpha3 "github.com/kdubbo/api/type/v1alpha3"
)

// Validation holds errors and warnings collected while validating a config.
type Validation struct {
	Err     error
	Warning Warning
}

// Unwrap returns the validation result as the (Warning, error) pair used by ValidateFunc.
func (v Validation) Unwrap() (Warning, error) {
	return v.Warning, v.Err
}

func appendValidation(v Validation, vs ...error) Validation {
	for _, nv := range vs {
		if nv != nil {
			v.Err = multierror.Append(v.Err, nv)
		}
	}
	return v
}

// AppendErrors joins non-nil errors into a multierror.
func AppendErrors(err error, errs ...error) error {
	appendError := func(err, err2 error) error {
		if err == nil {
			return err2
		} else if err2 == nil {
			return err
		}
		return multierror.Append(err, err2)
	}
	for _, err2 := range errs {
		err = appendError(err, err2)
	}
	return err
}

// ValidateMetadata checks that the config carries a valid name and namespace.
func ValidateMetadata(cfg config.Config) error {
	var errs error
	if cfg.Name == "" {
		errs = AppendErrors(errs, fmt.Errorf("name is required"))
	} else if msgs := validation.IsDNS1123Subdomain(cfg.Name); len(msgs) > 0 {
		errs = AppendErrors(errs, fmt.Errorf("invalid name %q: %v", cfg.Name, msgs))
	}
	if cfg.Namespace != "" {
		if msgs := validation.IsDNS1123Label(cfg.Namespace); len(msgs) > 0 {
			errs = AppendErrors(errs, fmt.Errorf("invalid namespace %q: %v", cfg.Namespace, msgs))
		}
	}
	return errs
}

func validateWorkloadSelector(selector *typev1alpha3.WorkloadSelector) error {
	if selector == nil {
		return nil
	}
	var errs error
	if len(selector.GetMatchLabels()) == 0 {
		errs = AppendErrors(errs, fmt.Errorf("selector is set but matchLabels is empty; omit the selector to apply to all workloads"))
	}
	l := labels.Instance(selector.GetMatchLabels())
	if err := l.Validate(); err != nil {
		errs = AppendErrors(errs, fmt.Errorf("invalid selector matchLabels: %v", err))
	}
	for k, v := range selector.GetMatchLabels() {
		if k == "" {
			errs = AppendErrors(errs, fmt.Errorf("selector matchLabels key must not be empty (value: %q)", v))
		}
	}
	return errs
}

func validateJwksURI(uri string) error {
	u, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("invalid jwksUri %q: %v", uri, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid jwksUri %q: scheme must be http or https", uri)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid jwksUri %q: host is required", uri)
	}
	return nil
}

func validatePositiveDuration(name string, d *durationpb.Duration) error {
	if d == nil {
		return nil
	}
	if err := d.CheckValid(); err != nil {
		return fmt.Errorf("invalid %s: %v", name, err)
	}
	if dur := d.AsDuration(); dur <= 0 {
		return fmt.Errorf("%s must be greater than 0, got %v", name, dur)
	}
	return nil
}

func validatePercent(name string, val int32) error {
	if val < 0 || val > 100 {
		return fmt.Errorf("%s must be in range [0, 100], got %d", name, val)
	}
	return nil
}

func validateNonNegativeInt32(name string, val int32) error {
	if val < 0 {
		return fmt.Errorf("%s must not be negative, got %d", name, val)
	}
	return nil
}
