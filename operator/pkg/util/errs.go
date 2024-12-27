package util

import "fmt"

const (
	defaultSeparator = ", "
)

type Errors []error

func (e Errors) Error() string {
	return ToString(e, defaultSeparator)
}

func (e Errors) ToErrors() error {
	if len(e) == 0 {
		return nil
	}
	return fmt.Errorf("%s", e)
}

func ToString(errors []error, separator string) string {
	var out string
	for i, e := range errors {
		if e == nil {
			continue
		}
		if i != 0 {
			out += separator
		}
		out += e.Error()
	}
	return out
}

func NewErrs(err error) Errors {
	if err == nil {
		return nil
	}
	return []error{err}
}

func AppendErr(errors []error, err error) Errors {
	if err == nil {
		if len(errors) == 0 {
			return nil
		}
		return errors
	}
	return append(errors, err)
}

func AppendErrs(errors []error, newErrs []error) Errors {
	if len(newErrs) == 0 {
		return errors
	}
	for _, e := range newErrs {
		errors = AppendErr(errors, e)
	}
	if len(errors) == 0 {
		return nil
	}
	return errors
}
