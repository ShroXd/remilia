package remilia

import "sync"

// 1. recyling put
// 2. stage network request fn

type stageOptions struct {
	name            string
	concurrency     uint
	inputBufferSize uint
}

type StageOptionFn OptionFn[*stageOptions]

func buildStageOptions(optFns []StageOptionFn) (*stageOptions, error) {
	so := &stageOptions{
		concurrency: uint(1),
	}
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

func Concurrency(concurrency uint) StageOptionFn {
	return func(so *stageOptions) error {
		if concurrency == 0 {
			return ErrInvalidConcurrency
		}
		so.concurrency = concurrency
		return nil
	}
}

func InputBufferSize(size uint) StageOptionFn {
	return func(so *stageOptions) error {
		if size == 0 {
			return ErrInvalidInputBufferSize
		}
		so.inputBufferSize = size
		return nil
	}
}

type commonStage[T any] struct {
	opts        *stageOptions
	emitToOutCh bool
	inCh        chan T
	outCh       chan<- T
}

func (cs commonStage[T]) outputChannelCloser() func() {
	instances := cs.concurrency()
	var mu sync.Mutex
	return func() {
		mu.Lock()
		defer mu.Unlock()
		instances--
		if instances == 0 {
			close(cs.outCh)
		}
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
type Get[T any] func() (T, bool)

// get - get data from upstream
// put - put data to downstream
// chew - put data back to upstream
type WorkFn[T any] func(get Get[T], put Put[T], chew Put[T]) error

type ProcessorDef[T any] func() (*processor[T], error)

type processor[T any] struct {
	commonStage[T]
	fn     WorkFn[T]
	getter func() (T, bool)
}

func buildProcessor[T any](fn WorkFn[T], opts *stageOptions) *processor[T] {
	p := &processor[T]{
		commonStage: commonStage[T]{
			opts:        opts,
			emitToOutCh: true,
			inCh:        make(chan T, opts.inputBufferSize),
		},
		fn: fn,
	}

	p.getter = func() (out T, ok bool) {
		select {
		// case out = <-p.inCh:
		// 	return out, true
		// default:
		// 	return out, false
		case out, ok = <-p.inCh:
			return out, ok
		}
	}
	return p
}

func NewProcessor[T any](fn WorkFn[T], optFns ...StageOptionFn) ProcessorDef[T] {
	return func() (*processor[T], error) {
		opts, err := buildStageOptions(optFns)
		if err != nil {
			return nil, err
		}

		return buildProcessor[T](fn, opts), nil
	}
}

func (p *processor[T]) execute() error {
	put := func(v T) {
		if p.emitToOutCh {
			p.outCh <- v
		}
	}
	chew := func(v T) {
		p.inCh <- v
	}

	return p.fn(p.getter, put, chew)
}

type FlowFn[T any] func(in T) (out T, err error)
type FlowDef[T any] func() (*flow[T], error)

type flow[T any] struct {
	commonStage[T]
	fn FlowFn[T]
}

func buildFlow[T any](fn FlowFn[T], opts *stageOptions) *flow[T] {
	return &flow[T]{
		commonStage: commonStage[T]{
			opts:        opts,
			emitToOutCh: true,
			inCh:        make(chan T, opts.inputBufferSize),
		},
		fn: fn,
	}
}

func NewFlow[T any](fn FlowFn[T], optFns ...StageOptionFn) FlowDef[T] {
	return func() (*flow[T], error) {
		opts, err := buildStageOptions(optFns)
		if err != nil {
			return nil, err
		}

		return buildFlow[T](fn, opts), nil
	}
}

func (s *flow[T]) executeOnce() (ok bool, err error) {
	var in, out T
	in, ok = <-s.inCh
	if !ok {
		return false, nil
	}
	out, err = s.fn(in)
	if err != nil {
		return false, err
	}

	if err == nil && s.emitToOutCh {
		s.outCh <- out
	}
	return ok, err
}

func (s *flow[T]) execute() error {
	ok, err := s.executeOnce()
	for ok && err == nil {
		ok, err = s.executeOnce()
	}

	return err
}

type StageFunc[T any] func(in T, put Put[T], chew Put[T]) error
type StageDef[T any] func() (*stage[T], error)

type stage[T any] struct {
	commonStage[T]
	fn   StageFunc[T]
	put  Put[T]
	chew Put[T]
}

func NewStage[T any](fn StageFunc[T], optFns ...StageOptionFn) StageDef[T] {
	return func() (*stage[T], error) {
		opts, err := buildStageOptions(optFns)
		if err != nil {
			return nil, err
		}

		stage := &stage[T]{
			commonStage: commonStage[T]{
				opts:        opts,
				emitToOutCh: true,
				inCh:        make(chan T, opts.inputBufferSize),
			},
			fn: fn,
		}

		stage.put = func(v T) {
			if stage.emitToOutCh {
				stage.outCh <- v
			}
		}

		stage.chew = func(v T) {
			stage.inCh <- v
		}

		return stage, nil
	}
}

func (s *stage[T]) executeOnce() (ok bool, err error) {
	// TODO: stage can not close itself
	in, ok := <-s.inCh
	if !ok {
		return false, nil
	}
	err = s.fn(in, s.put, s.chew)
	return ok, err
}

func (s *stage[T]) execute() error {
	ok, err := s.executeOnce()
	for ok && err == nil {
		ok, err = s.executeOnce()
	}

	return err
}
