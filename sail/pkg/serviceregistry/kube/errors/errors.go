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
