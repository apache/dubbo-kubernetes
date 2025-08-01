package krt

func FetchOne[T any](ctx HandlerContext, c Collection[T], opts ...FetchOption) *T {
	res := Fetch[T](ctx, c, opts...)
	switch len(res) {
	case 0:
		return nil
	case 1:
		return &res[0]
	default:
		panic("FetchOne found for more than 1 item")
	}
}

func Fetch[T any](ctx HandlerContext, cc Collection[T], opts ...FetchOption) []T {
	return fetch[T](ctx, cc, false, opts...)
}

func fetch[T any](ctx HandlerContext, cc Collection[T], allowMissingContext bool, opts ...FetchOption) []T {
	return nil
}
