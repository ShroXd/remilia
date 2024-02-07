package remilia

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStageOptions(t *testing.T) {
	t.Run("Successful build with valid options", func(t *testing.T) {
		so, err := buildStageOptions([]StageOptionFn{
			Name("test"),
			InputBufferSize(1),
		})

		assert.NoError(t, err, "buildStageOptions should not return error")
		assert.Equal(t, "test", so.name, "Name should be test")
		assert.Equal(t, uint(1), so.inputBufferSize, "InputBufferSize should be 1")
	})

	t.Run("Failed build with invalid options", func(t *testing.T) {
		so, err := buildStageOptions([]StageOptionFn{
			Name("test"),
			InputBufferSize(0),
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
			opts: &stageOptions{
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
			opts: &stageOptions{
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
			opts: &stageOptions{
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

		processor, err := NewProcessor[int](workFn, Name("test"), Concurrency(10), InputBufferSize(1))()
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
			processor, err := NewProcessor[int](workFn, Name("test"), Concurrency(0))()

			assert.Error(t, err, "NewProcessor should return error")
			assert.Equal(t, ErrInvalidConcurrency, err, "Error should be ErrInvalidConcurrency")
			assert.Nil(t, processor, "Processor should be nil")
		})

		t.Run("Invalid input buffer size", func(t *testing.T) {
			processor, err := NewProcessor[int](workFn, Name("test"), InputBufferSize(0))()

			assert.Error(t, err, "NewProcessor should return error")
			assert.Equal(t, ErrInvalidInputBufferSize, err, "Error should be ErrInvalidInputBufferSize")
			assert.Nil(t, processor, "Processor should be nil")
		})
	})

	t.Run("Successful execute", func(t *testing.T) {
		workFn := func(get Get[int], put Put[int], chew Put[int]) error {
			item, ok := get()
			if !ok {
				return nil
			}
			put(item * 2)
			chew(item * 3)
			return nil
		}

		processor, _ := NewProcessor[int](workFn, InputBufferSize(1))()
		receiver := &commonStage[int]{
			opts: &stageOptions{
				concurrency: uint(1),
			},
			inCh: make(chan int, 1),
		}
		defer close(receiver.inCh)
		processor.outCh = receiver.inCh

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			err := processor.execute()
			assert.NoError(t, err, "Processor should not return error")
		}()

		processor.inCh <- 1
		defer close(processor.inCh)

		wg.Wait()

		result, ok := <-receiver.inCh
		assert.True(t, ok, "Receiver should have received a value")
		assert.Equal(t, 2, result, "Receiver should have received 2")

		value, ok := <-processor.inCh
		assert.True(t, ok, "Processor should have received a value")
		assert.Equal(t, 3, value, "Processor should have received 3")
	})
}

func TestFlow(t *testing.T) {
	t.Run("Successful build with valid options", func(t *testing.T) {
		flowFn := func(in int) (out int, err error) {
			return in * 2, nil
		}

		flow, err := NewFlow(flowFn, Name("test"), InputBufferSize(1))()
		assert.NoError(t, err, "NewFlow should not return error")
		assert.Equal(t, "test", flow.opts.name, "Name should be test")
		assert.Equal(t, uint(1), flow.opts.inputBufferSize, "InputBufferSize should be 1")
	})

	t.Run("Failed build with invalid options", func(t *testing.T) {
		flowFn := func(in int) (out int, err error) {
			return in * 2, nil
		}

		flow, err := NewFlow(flowFn, Name("test"), InputBufferSize(0))()
		assert.Error(t, err, "NewFlow should return error")
		assert.Nil(t, flow, "Flow should be nil")
	})

	t.Run("executeOnce", func(t *testing.T) {
		t.Run("Successful execute", func(t *testing.T) {
			fn := func(in int) (out int, err error) {
				return in * 2, nil
			}

			flow, _ := NewFlow[int](fn, InputBufferSize(1))()
			receiver := &commonStage[int]{
				opts: &stageOptions{
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

			flow, _ := NewFlow[int](fn, InputBufferSize(1))()
			receiver := &commonStage[int]{
				opts: &stageOptions{
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

			flow, _ := NewFlow[int](fn, InputBufferSize(1))()
			receiver := &commonStage[int]{
				opts: &stageOptions{
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
