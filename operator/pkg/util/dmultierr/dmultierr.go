package dmultierr

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"strings"
)

func MultiErrorFormat() multierror.ErrorFormatFunc {
	return func(errors []error) string {
		if len(errors) == 1 {
			return errors[0].Error()
		}
		points := make([]string, len(errors))
		for index, err := range errors {
			points[index] = fmt.Sprintf("* %s", err)
		}
		return fmt.Sprintf("%d errors occurred:\n\t%s\n",
			len(errors), strings.Join(points, "\n\t"))
	}
}

func New() *multierror.Error {
	return &multierror.Error{
		ErrorFormat: MultiErrorFormat(),
	}
}
