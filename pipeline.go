package remilia

import (
	"sync"

	"golang.org/x/sync/errgroup"
)

type pipeline[T any] struct {
	provider *provider[T]
	layers   []*actionLayer[T]
}

func newPipeline[T any](pd providerDef[T], stageDefs ...actionLayerDef[T]) (*pipeline[T], error) {
	p := &pipeline[T]{}
	var err error

	// Build producer
	p.provider, err = pd()
	if err != nil {
		return nil, err
	}

	p.layers = make([]*actionLayer[T], len(stageDefs))
	for idx, stageDef := range stageDefs {
		stage, err := stageDef()
		if err != nil {
			return nil, err
		}
		p.layers[idx] = stage
	}

	lastStage := p.layers[0]
	p.provider.outCh = lastStage.inCh

	for _, stage := range p.layers[1:] {
		lastStage.outCh = stage.inCh
		lastStage = stage
	}

	// INS: insert stage into pipeline, between producers
	// the stage accept multiple values from producer and use FanOut to next producer

	// TODO: support recycling pipeline
	lastStage.outCh = p.provider.inCh
	lastStage.emitToOutCh = false

	return p, nil
}

func (p *pipeline[T]) execute() error {
	var eg errgroup.Group

	// TODO: currently it's only horizontal concurrency, we need to support vertical to improve the
	execute(&eg, p.provider)
	for _, stage := range p.layers {
		execute(&eg, stage)
	}

	return eg.Wait()
}

type executor interface {
	execute() error
	outputChannelCloser() func()
	exhaustInputChannel()
	concurrency() uint
}

func execute(eg *errgroup.Group, executor executor) {
	outputChannelCloser := executor.outputChannelCloser()

	for i := uint(0); i < executor.concurrency(); i++ {
		eg.Go(func() error {
			err := executor.execute()
			outputChannelCloser()
			executor.exhaustInputChannel()

			return err
		})
	}
}

// 1. recyling put
// 2. stage network request fn

// TODO: utilize the interface to reduce the code duplication
type commonStageOptions struct {
	name            string
	concurrency     uint
	inputBufferSize uint
}

type commonStageOptionFunc optionFunc[*commonStageOptions]

func buildCommonStageOptions(optFns []commonStageOptionFunc) (*commonStageOptions, error) {
	cso := &commonStageOptions{
		concurrency: uint(1),
	}
	for _, optFn := range optFns {
		if err := optFn(cso); err != nil {
			return nil, err
		}
	}
	return cso, nil
}

func withName(name string) commonStageOptionFunc {
	return func(cso *commonStageOptions) error {
		cso.name = name
		return nil
	}
}

func WithConcurrency(concurrency uint) commonStageOptionFunc {
	return func(cso *commonStageOptions) error {
		if concurrency == 0 {
			return errInvalidConcurrency
		}
		cso.concurrency = concurrency
		return nil
	}
}

func WithInputBufferSize(size uint) commonStageOptionFunc {
	return func(cso *commonStageOptions) error {
		if size == 0 {
			return errInvalidInputBufferSize
		}
		cso.inputBufferSize = size
		return nil
	}
}

type commonStage[T any] struct {
	opts        *commonStageOptions
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

type providerDef[T any] func() (*provider[T], error)

type provider[T any] struct {
	commonStage[T]
	fn     workFn[T]
	getter func() (T, bool)
}

func buildProvider[T any](fn workFn[T], opts *commonStageOptions) *provider[T] {
	p := &provider[T]{
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

func newProvider[T any](fn workFn[T], optFns ...commonStageOptionFunc) providerDef[T] {
	return func() (*provider[T], error) {
		opts, err := buildCommonStageOptions(optFns)
		if err != nil {
			return nil, err
		}

		return buildProvider[T](fn, opts), nil
	}
}

func (p *provider[T]) execute() error {
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

func buildFlow[T any](fn flowFn[T], opts *commonStageOptions) *flow[T] {
	return &flow[T]{
		commonStage: commonStage[T]{
			opts:        opts,
			emitToOutCh: true,
			inCh:        make(chan T, opts.inputBufferSize),
		},
		fn: fn,
	}
}

func newFlow[T any](fn flowFn[T], optFns ...commonStageOptionFunc) flowDef[T] {
	return func() (*flow[T], error) {
		opts, err := buildCommonStageOptions(optFns)
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

type actionLayerFunc[T any] func(get Get[T], put Put[T], inCh chan T) error
type actionLayerDef[T any] func() (*actionLayer[T], error)

type actionLayer[T any] struct {
	commonStage[T]
	fn  actionLayerFunc[T]
	put Put[T]
	get func() (T, bool)
}

func newActionLayer[T any](fn actionLayerFunc[T], optFns ...commonStageOptionFunc) actionLayerDef[T] {
	return func() (*actionLayer[T], error) {
		opts, err := buildCommonStageOptions(optFns)
		if err != nil {
			return nil, err
		}

		stage := &actionLayer[T]{
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

func (s *actionLayer[T]) executeOnce() (ok bool, err error) {
	var batchOk bool

	err = s.fn(s.get, s.put, s.inCh)
	return batchOk, err
}

func (s *actionLayer[T]) execute() error {
	ok, err := s.executeOnce()
	for ok && err == nil {
		ok, err = s.executeOnce()
	}

	return err
}
