package model

type typedXdsCache[K comparable] interface {
}

type lruCache[K comparable] struct {
}

var _ typedXdsCache[uint64] = &lruCache[uint64]{}

func newTypedXdsCache[K comparable]() typedXdsCache[K] {
	cache := &lruCache[K]{}
	return cache
}

type disabledCache[K comparable] struct{}

var _ typedXdsCache[uint64] = &disabledCache[uint64]{}
