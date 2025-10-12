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

package errors

import (
	"fmt"
)

// K8sServiceDiscoveryError represents errors related to Kubernetes service discovery
type K8sServiceDiscoveryError struct {
	Operation string
	Resource  string
	Reason    string
	Cause     error
}

func (e *K8sServiceDiscoveryError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("k8s service discovery error during %s of %s: %s (caused by: %v)",
			e.Operation, e.Resource, e.Reason, e.Cause)
	}
	return fmt.Sprintf("k8s service discovery error during %s of %s: %s",
		e.Operation, e.Resource, e.Reason)
}

func (e *K8sServiceDiscoveryError) Unwrap() error {
	return e.Cause
}

// NewServiceParsingError creates an error for service parsing failures
func NewServiceParsingError(serviceName, reason string, cause error) *K8sServiceDiscoveryError {
	return &K8sServiceDiscoveryError{
		Operation: "parsing",
		Resource:  fmt.Sprintf("service %s", serviceName),
		Reason:    reason,
		Cause:     cause,
	}
}
