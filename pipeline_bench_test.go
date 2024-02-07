package remilia

import "testing"

func BenchmarkPipelineExecution(b *testing.B) {
	tests := []struct {
		name        string
		concurrency int
		bufferSize  int
		dataSize    int
	}{
		// Base case
		{"Concurrency=10, BufferSize=10, DataSize=1000", 10, 10, 1000},

		// Change Concurrency
		{"Concurrency=100, BufferSize=10, DataSize=1000", 100, 10, 1000},
		{"Concurrency=1000, BufferSize=10, DataSize=1000", 1000, 10, 1000},

		// Change Buffer Size
		{"Concurrency=10, BufferSize=100, DataSize=1000", 10, 100, 1000},
		{"Concurrency=10, BufferSize=1000, DataSize=1000", 10, 1000, 1000},

		// Change Data Size
		{"Concurrency=10, BufferSize=10, DataSize=10000", 10, 10, 10000},
		{"Concurrency=10, BufferSize=10, DataSize=100000", 10, 10, 100000},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				b.StopTimer()
				generator := NewProcessor[int](func(get Get[int], put, chew Put[int]) error {
					for i := 0; i < tc.dataSize; i++ {
						put(i)
					}
					return nil
				})

				processor := NewProcessor[int](func(get Get[int], put, chew Put[int]) error {
					for {
						item, ok := get()
						if !ok {
							return nil
						}
						put(item * 2)
						// TODO: chew has bug with closed channel
						// chew(item * 3)
					}
				}, Concurrency(uint(tc.concurrency)))

				pipeline, _ := newPipeline[int](generator, processor)
				b.StartTimer()

				pipeline.execute()
			}
		})
	}
}
