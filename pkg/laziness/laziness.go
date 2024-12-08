package laziness

import (
	"sync"
	"sync/atomic"
)

type lazinessImpl[T any] struct {
	getter func() (T, error)
	retry  bool
	res    T
	err    error
	done   uint32
	m      sync.Mutex
}

type Laziness[T any] interface {
	Get() (T, error)
}

var _ Laziness[any] = &lazinessImpl[any]{}

func New[T any](f func() (T, error)) Laziness[T] {
	return &lazinessImpl[T]{
		getter: f,
	}
}

func NewWithRetry[T any](f func() (T, error)) Laziness[T] {
	return &lazinessImpl[T]{
		getter: f,
		retry:  true,
	}
}

func (l *lazinessImpl[T]) Get() (T, error) {
	if atomic.LoadUint32(&l.done) == 0 {
	}
}

func (l *lazinessImpl[T]) doSlow() (T, error) {
	l.m.Lock()
	defer l.m.Unlock()
	if l.done == 0 {
		done := uint32(1)
		defer func() {
			atomic.StoreInt32(&l.done, done)
		}()
		res, err := l.getter()
		if l.retry && err != nil {
			done = 0
		} else {
			l.res, l.err = res, err
		}
		return res, err
	}
	return l.res, l.err
}
