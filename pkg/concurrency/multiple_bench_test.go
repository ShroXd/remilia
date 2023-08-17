package concurrency

import "testing"

func generateDate(n int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)
		for i := 0; i < n; i++ {
			out <- i
		}
	}()

	return out
}

func BenchmarkFanIn(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		ch1 := generateDate(1000)
		ch2 := generateDate(1000)
		ch3 := generateDate(1000)
		done := make(chan struct{})

		b.StartTimer()

		out := FanIn(done, ch1, ch2, ch3)

		for range out {
		}
		close(done)
	}
}

func BenchmarkFanOut(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		input := generateDate(1000)
		done := make(chan struct{})
		fn := func(raw int) int { return raw }

		b.StartTimer()

		out := FanOut(done, input, 1000, fn)

		for range out {
		}
		close(done)
	}
}
