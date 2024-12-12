package util

const (
	defaultSeparator = ", "
)

type Errors []error

func (e Errors) Error() string {
	return ToString(e, defaultSeparator)
}

func (e Errors) ToString() {

}
