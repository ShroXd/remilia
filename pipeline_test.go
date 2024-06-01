package remilia

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sync/errgroup"
)

func mockProcessorDef() (*provider[any], error) {
	return &provider[any]{}, nil
}

func mockStageDef() (*actionLayer[any], error) {
	return &actionLayer[any]{}, nil
}

func TestNewPipeline(t *testing.T) {
	t.Run("NewPipeline with no stages", func(t *testing.T) {
		_, err := newPipeline[any](mockProcessorDef, mockStageDef, mockStageDef)
		assert.NoError(t, err, "newPipeline should not return error")
	})

	t.Run("NewPipeline with processor which returns error", func(t *testing.T) {
		errorProducerDef := func() (*provider[any], error) {
			return nil, errors.New("producer error")
		}
		_, err := newPipeline[any](errorProducerDef, mockStageDef)
		assert.Error(t, err, "newPipeline should have failed with producer error")
	})

	t.Run("NewPipeline with stage which returns error", func(t *testing.T) {
		errorStageDef := func() (*actionLayer[any], error) {
			return nil, errors.New("stage error")
		}
		normalStageDef := func() (*actionLayer[any], error) {
			return &actionLayer[any]{}, nil
		}

		_, err := newPipeline[any](mockProcessorDef, errorStageDef, normalStageDef)
		assert.Error(t, err, "newPipeline should have failed with stage error")
	})
}

func TestPipelineExecute(t *testing.T) {
	t.Run("Successful execute", func(t *testing.T) {
		generator := newProvider[int](func(get Get[int], put, chew Put[int]) error {
			put(1)
			return nil
		})

		processor := newActionLayer[int](func(get Get[int], put Put[int], inCh chan int) error {
			arr, _ := get()
			put(arr * 2)
			// TODO: chew has bug with closed channel
			// chew(item * 3)
			return nil
		})

		pipeline, _ := newPipeline[int](generator, processor)
		err := pipeline.execute()

		assert.NoError(t, err, "execute should not return error")
	})

	t.Run("Failed execute when any stage returns error", func(t *testing.T) {
		generator := newProvider[int](func(get Get[int], put, chew Put[int]) error {
			put(1)
			return nil
		})

		errProcessor := newActionLayer[int](func(get Get[int], put Put[int], inCh chan int) error {
			return errors.New("test error")
		})

		pipeline, _ := newPipeline[int](generator, errProcessor)
		err := pipeline.execute()

		assert.Error(t, err, "execute should return error")
		assert.Equal(t, "test error", err.Error(), "execute should return correct error")
	})
}

type mockExecutor struct {
	executeErr error
	mock.Mock
}

func (m *mockExecutor) execute() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockExecutor) outputChannelCloser() func() {
	args := m.Called()
	return args.Get(0).(func())
}

func (m *mockExecutor) exhaustInputChannel() {
	m.Called()
}

func (m *mockExecutor) concurrency() uint {
	args := m.Called()
	return args.Get(0).(uint)
}

func TestExecutor(t *testing.T) {
	t.Run("Successful execute", func(t *testing.T) {
		mockExec := new(mockExecutor)
		eg := new(errgroup.Group)

		mockExec.On("execute").Return(nil)
		mockExec.On("outputChannelCloser").Return(func() {})
		mockExec.On("exhaustInputChannel").Return()
		mockExec.On("concurrency").Return(uint(1))

		execute(eg, mockExec)
		err := eg.Wait()

		assert.NoError(t, err, "execute should not return error")
		mockExec.AssertExpectations(t)
	})

	t.Run("Failed execute", func(t *testing.T) {
		mockExec := new(mockExecutor)
		eg := new(errgroup.Group)

		mockExec.On("execute").Return(errors.New("test execute error"))
		mockExec.On("outputChannelCloser").Return(func() {})
		mockExec.On("exhaustInputChannel").Return()
		mockExec.On("concurrency").Return(uint(1))

		execute(eg, mockExec)
		err := eg.Wait()

		assert.Error(t, err, "execute should return error")
		assert.Equal(t, "test execute error", err.Error(), "execute should return correct error")
		mockExec.AssertExpectations(t)
	})
}

func TestStageOptions(t *testing.T) {
	t.Run("Successful build with valid options", func(t *testing.T) {
		so, err := buildCommonStageOptions([]commonStageOptionFunc{
			withName("test"),
			WithInputBufferSize(1),
		})

		assert.NoError(t, err, "buildStageOptions should not return error")
		assert.Equal(t, "test", so.name, "Name should be test")
		assert.Equal(t, uint(1), so.inputBufferSize, "InputBufferSize should be 1")
	})

	t.Run("Failed build with invalid options", func(t *testing.T) {
		so, err := buildCommonStageOptions([]commonStageOptionFunc{
			withName("test"),
			WithInputBufferSize(0),
		})

		assert.Error(t, err, "buildStageOptions should return error")
		assert.Nil(t, so, "stageOptions should be nil")
	})
}

func TestCommonStage(t *testing.T) {
	t.Run("outputChannelCloser should close output channel", func(t *testing.T) {
		// the output channel of current stage is the input channel of next stage
		inCh := make(chan int)
		stage := &commonStage[int]{
			opts: &commonStageOptions{
				concurrency: uint(1),
			},
			outCh: inCh,
		}
		closer := stage.outputChannelCloser()
		closer()

		_, ok := <-inCh
		assert.False(t, ok, "Output channel should be closed")
	})

	t.Run("exhaustInputChannel should exhaust input channel", func(t *testing.T) {
		inCh := make(chan int)
		stage := &commonStage[int]{
			opts: &commonStageOptions{
				concurrency: uint(1),
			},
			inCh: inCh,
		}

		go func() {
			inCh <- 1
			inCh <- 2
			close(inCh)
		}()
		stage.exhaustInputChannel()

		_, ok := <-inCh
		assert.False(t, ok, "Input channel should be exhausted")
	})

	t.Run("concurrency should return concurrency", func(t *testing.T) {
		stage := &commonStage[int]{
			opts: &commonStageOptions{
				concurrency: 2,
			},
		}
		assert.Equal(t, uint(2), stage.concurrency(), "Concurrency should be 2")
	})
}

func TestProcessor(t *testing.T) {
	t.Run("Successful build with valid options", func(t *testing.T) {
		workFn := func(get Get[int], put Put[int], chew Put[int]) error {
			return nil
		}

		processor, err := newProvider[int](workFn, withName("test"), WithConcurrency(10), WithInputBufferSize(1))()
		assert.NoError(t, err, "NewProcessor should not return error")
		assert.Equal(t, "test", processor.opts.name, "Name should be test")
		assert.Equal(t, uint(10), processor.opts.concurrency, "Concurrency should be 10")
		assert.Equal(t, uint(1), processor.opts.inputBufferSize, "InputBufferSize should be 1")
	})

	t.Run("Failed build with invalid options", func(t *testing.T) {
		workFn := func(get Get[int], put Put[int], chew Put[int]) error {
			return nil
		}

		t.Run("Invalid concurrency", func(t *testing.T) {
			processor, err := newProvider[int](workFn, withName("test"), WithConcurrency(0))()

			assert.Error(t, err, "NewProcessor should return error")
			assert.Equal(t, errInvalidConcurrency, err, "Error should be ErrInvalidConcurrency")
			assert.Nil(t, processor, "Processor should be nil")
		})

		t.Run("Invalid input buffer size", func(t *testing.T) {
			processor, err := newProvider[int](workFn, withName("test"), WithInputBufferSize(0))()

			assert.Error(t, err, "NewProcessor should return error")
			assert.Equal(t, errInvalidInputBufferSize, err, "Error should be ErrInvalidInputBufferSize")
			assert.Nil(t, processor, "Processor should be nil")
		})
	})

	t.Run("Successful execute once", func(t *testing.T) {
		workFn := func(get Get[int], put Put[int], chew Put[int]) error {
			item, ok := get()
			if !ok {
				return nil
			}
			put(item * 2)
			chew(item * 3)
			return nil
		}

		processor, _ := newProvider[int](workFn, WithInputBufferSize(1))()
		receiver := &commonStage[int]{
			opts: &commonStageOptions{
				concurrency: uint(1),
			},
			inCh: make(chan int, 1),
		}
		processor.outCh = receiver.inCh

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer func() {
				close(processor.inCh)
				close(receiver.inCh)
				wg.Done()
			}()
			err := processor.execute()
			assert.NoError(t, err, "Processor should not return error")
		}()

		processor.inCh <- 1

		wg.Wait()

		value, ok := <-receiver.inCh
		assert.True(t, ok, "Receiver should have received a value")
		assert.Equal(t, 2, value, "Receiver should have received 2")

		value, ok = <-processor.inCh
		assert.True(t, ok, "Processor should have received a value")
		assert.Equal(t, 3, value, "Processor should have received 3")
	})

	t.Run("Successful execution for all received value", func(t *testing.T) {
		hasChewed := false

		workFn := func(get Get[int], put Put[int], chew Put[int]) error {
			for i := 0; i < 2; i++ {
				item, ok := get()
				if !ok {
					return nil
				}
				put(item * 2)
				if !hasChewed {
					chew(item * 3)
					hasChewed = true
				}
			}

			return nil
		}

		processor, _ := newProvider[int](workFn, WithInputBufferSize(2))()
		receiver := &commonStage[int]{
			opts: &commonStageOptions{
				concurrency: uint(1),
			},
			inCh: make(chan int, 2),
		}
		processor.outCh = receiver.inCh

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer func() {
				close(receiver.inCh)
				close(processor.inCh)
				wg.Done()
			}()
			err := processor.execute()
			assert.NoError(t, err, "Processor should not return error")
		}()

		processor.inCh <- 1

		wg.Wait()

		value, ok := <-receiver.inCh
		assert.True(t, ok, "Receiver should have received a value")
		assert.Equal(t, 2, value, "Receiver should have received 2")

		value, ok = <-receiver.inCh
		assert.True(t, ok, "Processor should have received a value")
		assert.Equal(t, 6, value, "Processor should have received 3")

		value, ok = <-processor.inCh
		assert.False(t, ok, "Processor should not have received a value")
		assert.Equal(t, 0, value, "Processor should have received 0")
	})
}

func TestFlow(t *testing.T) {
	t.Run("Successful build with valid options", func(t *testing.T) {
		flowFn := func(in int) (out int, err error) {
			return in * 2, nil
		}

		flow, err := newFlow(flowFn, withName("test"), WithInputBufferSize(1))()
		assert.NoError(t, err, "NewFlow should not return error")
		assert.Equal(t, "test", flow.opts.name, "Name should be test")
		assert.Equal(t, uint(1), flow.opts.inputBufferSize, "InputBufferSize should be 1")
	})

	t.Run("Failed build with invalid options", func(t *testing.T) {
		flowFn := func(in int) (out int, err error) {
			return in * 2, nil
		}

		flow, err := newFlow(flowFn, withName("test"), WithInputBufferSize(0))()
		assert.Error(t, err, "NewFlow should return error")
		assert.Nil(t, flow, "Flow should be nil")
	})

	t.Run("executeOnce", func(t *testing.T) {
		t.Run("Successful execute", func(t *testing.T) {
			fn := func(in int) (out int, err error) {
				return in * 2, nil
			}

			flow, _ := newFlow[int](fn, WithInputBufferSize(1))()
			receiver := &commonStage[int]{
				opts: &commonStageOptions{
					concurrency: uint(1),
				},
				inCh: make(chan int, 1),
			}
			flow.outCh = receiver.inCh

			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer close(receiver.inCh)
				_, err := flow.executeOnce()
				assert.NoError(t, err, "Flow should not return error")

				ok, _ := flow.executeOnce()
				assert.False(t, ok, "Flow should not return ok")
			}()

			flow.inCh <- 1
			close(flow.inCh)

			wg.Wait()

			result, ok := <-receiver.inCh
			assert.True(t, ok, "Receiver should have received a value")
			assert.Equal(t, 2, result, "Receiver should have received 2")
		})

		t.Run("Failed execute when fn returns error", func(t *testing.T) {
			fn := func(in int) (out int, err error) {
				return 0, errors.New("test")
			}

			flow, _ := newFlow[int](fn, WithInputBufferSize(1))()
			receiver := &commonStage[int]{
				opts: &commonStageOptions{
					concurrency: uint(1),
				},
				inCh: make(chan int, 1),
			}
			flow.outCh = receiver.inCh

			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer close(receiver.inCh)
				ok, err := flow.executeOnce()

				assert.False(t, ok, "Flow should not return ok")
				assert.Error(t, err, "Flow should return error")
			}()

			flow.inCh <- 1
			close(flow.inCh)

			wg.Wait()

			_, ok := <-receiver.inCh
			assert.False(t, ok, "Receiver should not have received a value")
		})
	})

	t.Run("execute", func(t *testing.T) {
		t.Run("Successful execute", func(t *testing.T) {
			fn := func(in int) (out int, err error) {
				return in * 2, nil
			}

			flow, _ := newFlow[int](fn, WithInputBufferSize(1))()
			receiver := &commonStage[int]{
				opts: &commonStageOptions{
					concurrency: uint(1),
				},
				inCh: make(chan int, 3),
			}
			flow.outCh = receiver.inCh

			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer close(receiver.inCh)
				err := flow.execute()
				assert.NoError(t, err, "Flow should not return error")
			}()

			flow.inCh <- 1
			flow.inCh <- 2
			flow.inCh <- 3
			close(flow.inCh)

			wg.Wait()

			results := make([]int, 0, 3)
			for out := range receiver.inCh {
				results = append(results, out)
			}

			assert.Equal(t, []int{2, 4, 6}, results, "Receiver should have received 2, 4, 6")
		})
	})
}
