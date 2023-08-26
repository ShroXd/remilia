package concurrency

import (
	"testing"

	"github.com/ShroXd/remilia/pkg/utils"
)

var scenarios = []struct {
	data <-chan int
	name string
}{
	{utils.GenerateData(100), "Small Data"},
	{utils.GenerateData(1000), "Medium Data"},
	{utils.GenerateData(10000), "Large Data"},
	{utils.GenerateData(100000), "Huge Data"},
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
