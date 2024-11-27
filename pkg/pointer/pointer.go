package pointer

func Of[T any](t T) *T {
	return &t
}

func Empty[T any]() T {
	var empty T
	return empty
}
