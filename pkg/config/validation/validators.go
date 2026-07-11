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
	"math"
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	networking "github.com/kdubbo/api/networking/v1alpha3"
	security "github.com/kdubbo/api/security/v1alpha3"
	telemetry "github.com/kdubbo/api/telemetry/v1alpha1"
)

// ValidateAuthorizationPolicy checks that an AuthorizationPolicy is well-formed.
var ValidateAuthorizationPolicy = validateFunc(
	func(cfg config.Config) (Warning, error) {
		spec, ok := cfg.Spec.(*security.AuthorizationPolicy)
		if !ok {
			return nil, fmt.Errorf("cannot cast to AuthorizationPolicy")
		}
		v := Validation{}
		v = appendValidation(v, validateWorkloadSelector(spec.GetSelector()))
		if spec.GetAction() != security.AuthorizationPolicy_ALLOW &&
			spec.GetAction() != security.AuthorizationPolicy_DENY {
			v = appendValidation(v, fmt.Errorf("unsupported action %q", spec.GetAction()))
		}
		if spec.GetAction() == security.AuthorizationPolicy_DENY && len(spec.GetRules()) == 0 {
			v = appendValidation(v, fmt.Errorf("a DENY policy must have at least one rule; an empty DENY policy matches nothing"))
		}
		for i, rule := range spec.GetRules() {
			if rule == nil {
				v = appendValidation(v, fmt.Errorf("rule[%d] must not be null", i))
				continue
			}
			for j, from := range rule.GetFrom() {
				if from == nil {
					v = appendValidation(v, fmt.Errorf("rule[%d].from[%d] must not be null", i, j))
					continue
				}
				if from.GetSource() == nil {
					v = appendValidation(v, fmt.Errorf("rule[%d].from[%d].source must be set", i, j))
					continue
				}
				if len(from.GetSource().GetRequestPrincipals()) == 0 {
					v = appendValidation(v, fmt.Errorf("rule[%d].from[%d].source must specify requestPrincipals", i, j))
				}
				for k, principal := range from.GetSource().GetRequestPrincipals() {
					if principal == "" {
						v = appendValidation(v, fmt.Errorf("rule[%d].from[%d].source.requestPrincipals[%d] must not be empty", i, j, k))
					}
				}
			}
			for j, when := range rule.GetWhen() {
				if when == nil {
					v = appendValidation(v, fmt.Errorf("rule[%d].when[%d] must not be null", i, j))
					continue
				}
				if when.GetKey() == "" {
					v = appendValidation(v, fmt.Errorf("rule[%d].when[%d].key must not be empty", i, j))
				}
				if len(when.GetValues()) == 0 && len(when.GetNotValues()) == 0 {
					v = appendValidation(v, fmt.Errorf("rule[%d].when[%d] must specify values or notValues", i, j))
				}
			}
		}
		return v.Unwrap()
	})

// ValidatePeerAuthentication checks that a PeerAuthentication is well-formed.
var ValidatePeerAuthentication = validateFunc(
	func(cfg config.Config) (Warning, error) {
		spec, ok := cfg.Spec.(*security.PeerAuthentication)
		if !ok {
			return nil, fmt.Errorf("cannot cast to PeerAuthentication")
		}
		v := Validation{}
		v = appendValidation(v, validateWorkloadSelector(spec.GetSelector()))
		v = appendValidation(v, validateMutualTLSMode("mtls.mode", spec.GetMtls().GetMode()))
		for port, mtls := range spec.GetPortLevelMtls() {
			if port == 0 {
				v = appendValidation(v, fmt.Errorf("portLevelMtls port must not be 0"))
			}
			if mtls == nil {
				v = appendValidation(v, fmt.Errorf("portLevelMtls[%d] must not be null", port))
				continue
			}
			v = appendValidation(v, validateMutualTLSMode(fmt.Sprintf("portLevelMtls[%d].mode", port), mtls.GetMode()))
		}
		if len(spec.GetPortLevelMtls()) > 0 && spec.GetSelector() == nil {
			v = appendValidation(v, fmt.Errorf("portLevelMtls requires a workload selector"))
		}
		return v.Unwrap()
	})

func validateMutualTLSMode(field string, mode security.PeerAuthentication_MutualTLS_Mode) error {
	switch mode {
	case security.PeerAuthentication_MutualTLS_UNSET,
		security.PeerAuthentication_MutualTLS_DISABLE,
		security.PeerAuthentication_MutualTLS_PERMISSIVE,
		security.PeerAuthentication_MutualTLS_STRICT:
		return nil
	default:
		return fmt.Errorf("unsupported %s %q", field, mode)
	}
}

// ValidateRequestAuthentication checks that a RequestAuthentication is well-formed.
var ValidateRequestAuthentication = validateFunc(
	func(cfg config.Config) (Warning, error) {
		spec, ok := cfg.Spec.(*security.RequestAuthentication)
		if !ok {
			return nil, fmt.Errorf("cannot cast to RequestAuthentication")
		}
		v := Validation{}
		v = appendValidation(v, validateWorkloadSelector(spec.GetSelector()))
		for i, rule := range spec.GetJwtRules() {
			if rule == nil {
				v = appendValidation(v, fmt.Errorf("jwtRules[%d] must not be null", i))
				continue
			}
			if rule.GetIssuer() == "" {
				v = appendValidation(v, fmt.Errorf("jwtRules[%d].issuer must not be empty", i))
			}
			if rule.GetJwksUri() != "" && rule.GetJwks() != "" {
				v = appendValidation(v, fmt.Errorf("jwtRules[%d]: only one of jwksUri or jwks can be set", i))
			}
			if rule.GetJwksUri() != "" {
				if err := validateJwksURI(rule.GetJwksUri()); err != nil {
					v = appendValidation(v, fmt.Errorf("jwtRules[%d]: %v", i, err))
				}
			}
			for j, header := range rule.GetFromHeaders() {
				if header == nil || header.GetName() == "" {
					v = appendValidation(v, fmt.Errorf("jwtRules[%d].fromHeaders[%d].name must not be empty", i, j))
				}
			}
			for j, param := range rule.GetFromParams() {
				if param == "" {
					v = appendValidation(v, fmt.Errorf("jwtRules[%d].fromParams[%d] must not be empty", i, j))
				}
			}
		}
		return v.Unwrap()
	})

// ValidateCircuitBreakerPolicy checks that a CircuitBreakerPolicy is well-formed.
var ValidateCircuitBreakerPolicy = validateFunc(
	func(cfg config.Config) (Warning, error) {
		spec, ok := cfg.Spec.(*networking.CircuitBreakerPolicy)
		if !ok {
			return nil, fmt.Errorf("cannot cast to CircuitBreakerPolicy")
		}
		v := Validation{}
		if len(spec.GetTargetRefs()) == 0 {
			v = appendValidation(v, fmt.Errorf("targetRefs must not be empty"))
		}
		for i, ref := range spec.GetTargetRefs() {
			if ref == nil {
				v = appendValidation(v, fmt.Errorf("targetRefs[%d] must not be null", i))
				continue
			}
			if ref.GetKind() == "" {
				v = appendValidation(v, fmt.Errorf("targetRefs[%d].kind must not be empty", i))
			} else if ref.GetKind() != "Service" {
				v = appendValidation(v, fmt.Errorf("targetRefs[%d].kind %q is not supported; only Service targets are applied", i, ref.GetKind()))
			}
			if ref.GetName() == "" {
				v = appendValidation(v, fmt.Errorf("targetRefs[%d].name must not be empty", i))
			}
		}
		if spec.GetConnectionPool() == nil && spec.GetOutlierDetection() == nil {
			v = appendValidation(v, fmt.Errorf("at least one of connectionPool or outlierDetection must be set"))
		}
		if cp := spec.GetConnectionPool(); cp != nil {
			v = appendValidation(v,
				validateNonNegativeInt32("connectionPool.maxConnections", cp.GetMaxConnections()),
				validateNonNegativeInt32("connectionPool.http1MaxPendingRequests", cp.GetHttp1MaxPendingRequests()),
				validateNonNegativeInt32("connectionPool.http2MaxRequests", cp.GetHttp2MaxRequests()),
				validateNonNegativeInt32("connectionPool.maxRequestsPerConnection", cp.GetMaxRequestsPerConnection()),
				validateNonNegativeInt32("connectionPool.maxRetries", cp.GetMaxRetries()),
			)
		}
		if od := spec.GetOutlierDetection(); od != nil {
			v = appendValidation(v,
				validatePositiveDuration("outlierDetection.interval", od.GetInterval()),
				validatePositiveDuration("outlierDetection.baseEjectionTime", od.GetBaseEjectionTime()),
				validatePercent("outlierDetection.maxEjectionPercent", od.GetMaxEjectionPercent()),
				validatePercent("outlierDetection.minHealthPercent", od.GetMinHealthPercent()),
			)
		}
		return v.Unwrap()
	})

// ValidateTelemetry checks that a Telemetry resource is well-formed.
var ValidateTelemetry = RegisterValidateFunc("ValidateTelemetry",
	func(cfg config.Config) (Warning, error) {
		spec, ok := cfg.Spec.(*telemetry.Telemetry)
		if !ok {
			return nil, fmt.Errorf("cannot cast to Telemetry")
		}
		v := Validation{}
		v = appendValidation(v, validateWorkloadSelector(spec.GetSelector()))
		if cfg.Namespace == constants.DubboSystemNamespace && spec.GetSelector() != nil {
			v = appendValidation(v, fmt.Errorf("selector is not allowed on meshlevel Telemetry in namespace %q", constants.DubboSystemNamespace))
		}
		for i, t := range spec.GetTracing() {
			if t == nil {
				v = appendValidation(v, fmt.Errorf("tracing[%d] must not be null", i))
				continue
			}
			for j, p := range t.GetProviders() {
				if strings.TrimSpace(p.GetName()) == "" {
					v = appendValidation(v, fmt.Errorf("tracing[%d].providers[%d].name must be set", i, j))
				}
			}
			if s := t.GetRandomSamplingPercentage(); s != nil {
				if math.IsNaN(s.GetValue()) || math.IsInf(s.GetValue(), 0) || s.GetValue() < 0 || s.GetValue() > 100 {
					v = appendValidation(v, fmt.Errorf("tracing[%d].randomSamplingPercentage must be in range [0.0, 100.0], got %v", i, s.GetValue()))
				}
			}
			tagNames := map[string]struct{}{}
			for j, tag := range t.GetTags() {
				name := strings.TrimSpace(tag.GetName())
				if name == "" {
					v = appendValidation(v, fmt.Errorf("tracing[%d].tags[%d].name must be set", i, j))
					continue
				}
				if _, found := tagNames[name]; found {
					v = appendValidation(v, fmt.Errorf("tracing[%d].tags[%d].name %q is duplicated", i, j, name))
				}
				tagNames[name] = struct{}{}
			}
		}
		return v.Unwrap()
	})
