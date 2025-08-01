package krt

func NewSingleton[O any](hf TransformationEmpty[O], opts ...CollectionOption) Singleton[O] {
	return nil
}

var (
	_ Singleton[any] = &collectionAdapter[any]{}
)

type collectionAdapter[T any] struct {
	c Collection[T]
}

func (c collectionAdapter[T]) Get() *T {
	// Guaranteed to be 0 or 1 len
	res := c.c.List()
	if len(res) == 0 {
		return nil
	}
	return &res[0]
}

func (c collectionAdapter[T]) Metadata() Metadata {
	// The metadata is passed to the internal dummy collection so just return that
	return c.c.Metadata()
}

func (c collectionAdapter[T]) Register(f func(o Event[T])) HandlerRegistration {
	return c.c.Register(f)
}

func (c collectionAdapter[T]) AsCollection() Collection[T] {
	return c.c
}
