package remilia

import "sync"

type abstractFactory[T any] interface {
	New() T
	Reset(T)
}

type abstractPool[T any] struct {
	factory abstractFactory[T]
	pool    *sync.Pool
}

func newPool[T any](factory abstractFactory[T]) *abstractPool[T] {
	return &abstractPool[T]{
		factory: factory,
		pool: &sync.Pool{
			New: func() any {
				return factory.New()
			},
		},
	}
}

func (p *abstractPool[T]) get() T {
	return p.pool.Get().(T)
}

func (p *abstractPool[T]) put(item T) {
	p.factory.Reset(item)
	p.pool.Put(item)
}
