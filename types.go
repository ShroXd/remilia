package remilia

type OptionFunc[T any] func(ins T) error
