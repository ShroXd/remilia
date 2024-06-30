package remilia

type ExecutableFunc func() error
type optionFunc[T any] func(ins T) error
