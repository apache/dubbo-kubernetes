package pointer

func Of[T any](t T) *T {
	return &t
}
