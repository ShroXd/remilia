package remilia

import (
	"testing"
)

func GenerateData(n int) chan int {
	out := make(chan int)

	go func() {
		defer close(out)
		for i := 0; i < n; i++ {
			out <- i
		}
	}()

	return out
}

var scenarios = []struct {
	data <-chan int
	name string
}{
	{GenerateData(100), "Small Data"},
	{GenerateData(1000), "Medium Data"},
	{GenerateData(10000), "Large Data"},
	{GenerateData(100000), "Huge Data"},
}

func BenchmarkFanIn(b *testing.B) {
	b.ReportAllocs()

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()

				ch1 := s.data
				ch2 := s.data
				ch3 := s.data
				done := make(chan struct{})

				b.StartTimer()

				out := FanIn(done, ch1, ch2, ch3)

				for range out {
				}
				close(done)
			}
		})
	}
}
