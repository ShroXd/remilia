package remilia

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessorExecute(t *testing.T) {
	workFn := func(get Get[int], put Put[int], chew Put[int]) error {
		item, ok := get()
		if !ok {
			return nil
		}
		put(item * 2)
		return nil
	}

	processor, _ := NewProcessor[int](workFn, InputBufferSize(1))()
	receiver := &commonStage[int]{
		inCh: make(chan int, 1),
	}
	processor.outCh = receiver.inCh

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		err := processor.execute()
		assert.NoError(t, err, "Processor should not return error")
	}()

	processor.inCh <- 1
	close(processor.inCh)

	wg.Wait()
	close(receiver.inCh)

	result, ok := <-receiver.inCh
	assert.True(t, ok, "Receiver should have received a value")
	assert.Equal(t, 2, result, "Receiver should have received 2")
}

func TestFlowExecute(t *testing.T) {
	fn := func(in int) (out int, err error) {
		return in * 2, nil
	}

	flow, _ := NewFlow[int](fn, InputBufferSize(1))()
	receiver := &commonStage[int]{
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
}
