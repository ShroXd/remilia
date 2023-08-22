package concurrency

import (
	"remilia/pkg/utils"
	"testing"
)

var scenarios = []struct {
	data <-chan int
	name string
}{
	{utils.GenerateData(100), "Small Data"},
	{utils.GenerateData(1000), "Medium Data"},
	{utils.GenerateData(10000), "Large Data"},
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

func BenchmarkFanOut(b *testing.B) {
	b.ReportAllocs()

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()

				input := s.data
				done := make(chan struct{})
				fn := func(raw int) []int { return []int{raw} }

				b.StartTimer()

				out := FanOut(done, input, 1000, fn)

				for range out {
				}
				close(done)
			}
		})
	}
}
