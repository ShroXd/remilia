package remilia

import "sync"

type Factory[T any] interface {
	New() T
	Reset(T)
}

type Pool[T any] struct {
	factory Factory[T]
	pool    *sync.Pool
}

func NewPool[T any](factory Factory[T]) *Pool[T] {
	return &Pool[T]{
		factory: factory,
		pool: &sync.Pool{
			New: func() any {
				return factory.New()
			},
		},
	}
}

func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

func (p *Pool[T]) Put(item T) {
	p.factory.Reset(item)
	p.pool.Put(item)
}
