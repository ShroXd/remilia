package remilia

// 1. recyling put
// 2. stage network request fn

type stageOptions struct {
	name            string
	concurrency     uint
	inputBufferSize uint
}

type StageOptionFn func(so *stageOptions) error

func buildStageOptions(optFns []StageOptionFn) (*stageOptions, error) {
	so := &stageOptions{}
	for _, optFn := range optFns {
		if err := optFn(so); err != nil {
			return nil, err
		}
	}
	return so, nil
}

func Name(name string) StageOptionFn {
	return func(so *stageOptions) error {
		so.name = name
		return nil
	}
}

func InputBufferSize(size uint) StageOptionFn {
	return func(so *stageOptions) error {
		so.inputBufferSize = size
		return nil
	}
}

type commonStage[T any] struct {
	opts  *stageOptions
	inCh  chan T
	outCh chan<- T
}

func (cs commonStage[T]) outputChannelCloser() func() {
	return func() {
		close(cs.outCh)
	}
}

func (cs commonStage[T]) exhaustInputChannel() {
	for range cs.inCh {
	}
}

func (cs commonStage[T]) concurrency() uint {
	return cs.opts.concurrency
}

type Put[T any] func(T)
type Get[T any] func() T

// TODO: add getter for cycling pipeline
type ProducerFn[T any] func(get Get[T], put Put[T], chew Put[T]) error
type ProducerDef[T any] func() (*producer[T], error)

type producer[T any] struct {
	commonStage[T]
	fn     ProducerFn[T]
	getter func() T
}

func buildProducer[T any](fn ProducerFn[T], opts *stageOptions) *producer[T] {
	p := &producer[T]{
		commonStage: commonStage[T]{
			opts: opts,
			inCh: make(chan T, opts.inputBufferSize),
		},
		fn: fn,
	}

	p.getter = func() T {
		return <-p.inCh
	}
	return p
}

func NewProducer[T any](fn ProducerFn[T], optFns ...StageOptionFn) ProducerDef[T] {
	return func() (*producer[T], error) {
		opts, err := buildStageOptions(optFns)
		if err != nil {
			return nil, err
		}

		return buildProducer[T](fn, opts), nil
	}
}

func (p *producer[T]) execute() error {
	put := func(v T) {
		p.outCh <- v
	}
	chew := func(v T) {
		p.inCh <- v
	}

	return p.fn(p.getter, put, chew)
}

type StageFn[T any] func(in T) (out T, err error)
type StageDef[T any] func() (*stage[T], error)

type stage[T any] struct {
	commonStage[T]
	fn StageFn[T]
}

func buildStage[T any](fn StageFn[T], opts *stageOptions) *stage[T] {
	return &stage[T]{
		commonStage: commonStage[T]{
			opts: opts,
			inCh: make(chan T, opts.inputBufferSize),
		},
		fn: fn,
	}
}

func NewStage[T any](fn StageFn[T], optFns ...StageOptionFn) StageDef[T] {
	return func() (*stage[T], error) {
		opts, err := buildStageOptions(optFns)
		if err != nil {
			return nil, err
		}

		return buildStage[T](fn, opts), nil
	}
}

func (s *stage[T]) executeOnce() (ok bool, err error) {
	var in, out T
	in, ok = <-s.inCh
	if !ok {
		return false, nil
	}
	out, err = s.fn(in)
	if err != nil {
		return false, err
	}

	s.outCh <- out
	return ok, err
}

func (s *stage[T]) execute() error {
	ok, err := s.executeOnce()
	for ok && err != nil {
		ok, err = s.executeOnce()
	}

	return err
}
