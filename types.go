package remilia

type OptionFn[T any] func(ins T) error
