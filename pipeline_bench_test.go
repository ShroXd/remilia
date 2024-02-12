package remilia

import (
	"fmt"
	"os"
	"runtime/trace"
	"testing"
)

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

		// More concurrency on large data size
		{"Concurrency=100, BufferSize=10, DataSize=100000", 100, 10, 100000},

		// More buffer size on large data size
		{"Concurrency=10, BufferSize=100, DataSize=100000", 100, 100, 100000},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			// Ensure the output directory exists
			err := os.MkdirAll("./out", 0755)
			if err != nil {
				b.Fatalf("failed to create out directory: %v", err)
			}

			// Create a new trace file for each test case to avoid conflicts
			traceFileName := fmt.Sprintf("./out/trace_%s.pprof", tc.name)
			traceFile, err := os.Create(traceFileName)
			if err != nil {
				b.Fatalf("failed to create trace file: %v", err)
			}
			defer traceFile.Close()

			// Start tracing
			err = trace.Start(traceFile)
			if err != nil {
				b.Fatalf("failed to start trace: %v", err)
			}
			// Ensure tracing stops at the end of each test case
			defer trace.Stop()

			// Reset timer to exclude setup time
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				generator := NewProcessor[int](func(get Get[int], put, chew Put[int]) error {
					for i := 0; i < tc.dataSize; i++ {
						put(i)
					}
					return nil
				})

				processor := NewStage[int](func(get BatchGetFunc[int], put, chew Put[int]) error {
					vals, _ := get()
					put(vals[0] * 2)
					return nil
				}, Concurrency(uint(tc.concurrency)))

				pipeline, _ := newPipeline[int](generator, processor)
				pipeline.execute()
			}
			b.StopTimer()
			// Explicitly stop tracing to ensure it finishes before closing the file
			trace.Stop()
		})
	}
}
