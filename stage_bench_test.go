package remilia

import (
	"sync"
	"testing"
)

func BenchmarkProcessorExecution(b *testing.B) {
	tests := []struct {
		name          string
		concurrency   int
		bufferSize    int
		dataSize      int
		parallelInput int
	}{
		// Change Concurrency
		{"Concurrency=10, BufferSize=100, DataSize=1000", 10, 100, 1000, 10},
		{"Concurrency=100, BufferSize=100, DataSize=1000", 100, 100, 1000, 10},
		{"Concurrency=1000, BufferSize=100, DataSize=1000", 1000, 100, 1000, 10},

		// Change Buffer Size
		{"Concurrency=100, BufferSize=10, DataSize=1000", 100, 10, 1000, 10},
		{"Concurrency=100, BufferSize=100, DataSize=1000", 100, 100, 1000, 10},
		{"Concurrency=100, BufferSize=1000, DataSize=1000", 100, 1000, 1000, 10},

		// Change Data Size
		{"Concurrency=100, BufferSize=100, DataSize=100", 100, 100, 100, 10},
		{"Concurrency=100, BufferSize=101, DataSize=1000", 100, 101, 1000, 10},
		// {"Concurrency=100, BufferSize=100, DataSize=10000", 100, 100, 10000, 10},

		// More concurrency on large data size
		// {"Concurrency=100, BufferSize=10, DataSize=100000", 100, 10, 100000, 10},
		// {"Concurrency=1000, BufferSize=10, DataSize=100000", 1000, 10, 100000, 10},

		// More buffer size on large data size
		// {"Concurrency=100, BufferSize=1000, DataSize=100000", 10, 100, 100000, 10},
		// {"Concurrency=100, BufferSize=10000, DataSize=100000", 10, 1000, 100000, 10},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			workFn := func(get Get[int], put Put[int], chew Put[int]) error {
				item, ok := get()
				if !ok {
					return nil
				}
				put(item * 2)
				return nil
			}
			opts := []stageOptionFn{
				withConcurrency(uint(tc.concurrency)),
				withInputBufferSize(uint(tc.bufferSize)),
			}
			processor, _ := newProcessor[int](workFn, opts...)()

			receiver := &commonStage[int]{
				inCh: make(chan int, tc.dataSize),
			}
			processor.outCh = receiver.inCh

			b.ResetTimer()

			for n := 0; n < b.N; n++ {
				var wg sync.WaitGroup

				for p := 0; p < tc.parallelInput; p++ {
					wg.Add(1)
					go func(p int) {
						defer wg.Done()
						for i := p; i < tc.dataSize; i += tc.parallelInput {
							processor.inCh <- i
						}
					}(p)
				}

				wg.Add(1)
				go func() {
					defer close(receiver.inCh)
					defer wg.Done()
					for range receiver.inCh {
					}
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()
					_ = processor.execute()
				}()
			}
		})
	}
}
