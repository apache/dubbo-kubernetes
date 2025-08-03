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

func Equal[T comparable](a, b *T) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	return *a == *b
}
