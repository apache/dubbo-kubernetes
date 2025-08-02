package ptr

import "fmt"

func Of[T any](t T) *T {
	return &t
}

func Empty[T any]() T {
	var empty T
	return empty
}

func Flatten[T any](t **T) *T {
	if t == nil {
		return nil
	}
	return *t
}

func TypeName[T any]() string {
	var empty T
	return fmt.Sprintf("%T", empty)
}
