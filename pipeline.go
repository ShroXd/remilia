package remilia

import "golang.org/x/sync/errgroup"

type pipeline[T any] struct {
	producer *processor[T]
	stages   []*processor[T]
}

func newPipeline[T any](producerDef ProcessorDef[T], stageDefs ...ProcessorDef[T]) (*pipeline[T], error) {
	p := &pipeline[T]{}
	var err error

	// Build producer
	p.producer, err = producerDef()
	if err != nil {
		return nil, err
	}

	p.stages = make([]*processor[T], len(stageDefs))
	for idx, stageDef := range stageDefs {
		stage, err := stageDef()
		if err != nil {
			return nil, err
		}
		p.stages[idx] = stage
	}

	lastStage := p.stages[0]
	p.producer.outCh = lastStage.inCh

	for _, stage := range p.stages[1:] {
		lastStage.outCh = stage.inCh
		lastStage = stage
	}

	// TODO: support recycling pipeline
	lastStage.outCh = p.producer.inCh

	return p, nil
}

func (p *pipeline[T]) execute() error {
	var eg errgroup.Group

	execute(&eg, p.producer)
	for _, stage := range p.stages {
		execute(&eg, stage)
	}

	return eg.Wait()
}

func Execute(producerDef ProcessorDef[any], stageDef ...ProcessorDef[any]) error {
	pipeline, err := newPipeline[any](producerDef, stageDef...)
	if err != nil {
		return err
	}

	return pipeline.execute()
}

type executor interface {
	execute() error
	outputChannelCloser() func()
	exhaustInputChannel()
}

func execute(eg *errgroup.Group, executor executor) {
	outputChannelCloser := executor.outputChannelCloser()

	eg.Go(func() error {
		err := executor.execute()
		outputChannelCloser()
		executor.exhaustInputChannel()

		return err
	})
}
