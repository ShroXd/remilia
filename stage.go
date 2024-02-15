package remilia

import "sync"

// 1. recyling put
// 2. stage network request fn

type stageOptions struct {
	name            string
	concurrency     uint
	inputBufferSize uint
}

type stageOptionFn optionFunc[*stageOptions]

func buildStageOptions(optFns []stageOptionFn) (*stageOptions, error) {
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

func withName(name string) stageOptionFn {
	return func(so *stageOptions) error {
		so.name = name
		return nil
	}
}

func withConcurrency(concurrency uint) stageOptionFn {
	return func(so *stageOptions) error {
		if concurrency == 0 {
			return errInvalidConcurrency
		}
		so.concurrency = concurrency
		return nil
	}
}

func withInputBufferSize(size uint) stageOptionFn {
	return func(so *stageOptions) error {
		if size == 0 {
			return errInvalidInputBufferSize
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
type workFn[T any] func(get Get[T], put Put[T], chew Put[T]) error

type processorDef[T any] func() (*processor[T], error)

type processor[T any] struct {
	commonStage[T]
	fn     workFn[T]
	getter func() (T, bool)
}

func buildProcessor[T any](fn workFn[T], opts *stageOptions) *processor[T] {
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

func newProcessor[T any](fn workFn[T], optFns ...stageOptionFn) processorDef[T] {
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

type flowFn[T any] func(in T) (out T, err error)
type flowDef[T any] func() (*flow[T], error)

type flow[T any] struct {
	commonStage[T]
	fn flowFn[T]
}

func buildFlow[T any](fn flowFn[T], opts *stageOptions) *flow[T] {
	return &flow[T]{
		commonStage: commonStage[T]{
			opts:        opts,
			emitToOutCh: true,
			inCh:        make(chan T, opts.inputBufferSize),
		},
		fn: fn,
	}
}

func newFlow[T any](fn flowFn[T], optFns ...stageOptionFn) flowDef[T] {
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

type stageFunc[T any] func(get Get[T], put Put[T], inCh chan T) error
type stageDef[T any] func() (*stage[T], error)

type stage[T any] struct {
	commonStage[T]
	fn  stageFunc[T]
	put Put[T]
	get func() (T, bool)
}

func newStage[T any](fn stageFunc[T], optFns ...stageOptionFn) stageDef[T] {
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

		stage.get = func() (out T, ok bool) {
			select {
			case out, ok = <-stage.inCh:
				return out, ok
			}
		}

		return stage, nil
	}
}

func (s *stage[T]) executeOnce() (ok bool, err error) {
	var batchOk bool

	err = s.fn(s.get, s.put, s.inCh)
	return batchOk, err
}

func (s *stage[T]) execute() error {
	ok, err := s.executeOnce()
	for ok && err == nil {
		ok, err = s.executeOnce()
	}

	return err
}
