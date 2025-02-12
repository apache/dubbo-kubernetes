package pointer

func Of[T any](t T) *T {
	return &t
}

func Empty[T any]() T {
	var empty T
	return empty
}

func NonEmptyOrDefault[T comparable](t T, def T) T {
	var empty T
	if t != empty {
		return t
	}
	return def
}
