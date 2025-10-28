package inject

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
)

type annotationValidationFunc func(value string) error

var (
	AnnotationValidation = map[string]annotationValidationFunc{}
)

func validateAnnotations(annotations map[string]string) (err error) {
	for name, value := range annotations {
		if v, ok := AnnotationValidation[name]; ok {
			if e := v(value); e != nil {
				err = multierror.Append(err, fmt.Errorf("invalid value '%s' for annotation '%s': %v", value, name, e))
			}
		}
	}
	return
}
