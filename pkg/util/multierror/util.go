package multierror

import (
	"strings"

	"github.com/hashicorp/go-multierror"
)

func MultiErrorFormat() multierror.ErrorFormatFunc {
	return func(es []error) string {
		if len(es) == 1 {
			return es[0].Error()
		}

		var b strings.Builder

		for i, err := range es {
			if i > 0 {
				b.WriteString("\n\t")
			}
			b.WriteString("* ")
			b.WriteString(err.Error())
		}

		b.WriteByte('\n')
		return b.String()
	}
}

func New() *multierror.Error {
	return &multierror.Error{ErrorFormat: MultiErrorFormat()}
}
