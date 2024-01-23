package v1alpha1

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/validators"
)

func (r *ServiceNameMappingResource) validate() error {
	var verr validators.ValidationError

	return verr.OrNil()
}
