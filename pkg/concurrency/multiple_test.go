package concurrency

import (
	"testing"
)

func TestFanIn(t *testing.T) {
	t.Run("single channel", func(t *testing.T) {
		ch := make(chan interface{}, 1)
		ch <- "test"
		close(ch)

		done := make(chan interface{})
		defer close(done)

		out := FanIn(done, ch)
		if got := <-out; got != "test" {
			t.Fatalf("expected 'test', got %v", got)
		}
	})

	t.Run("multiple channel", func(t *testing.T) {
		num := 10
		channels := make([]<-chan interface{}, num)
		for i := 0; i < num; i++ {
			ch := make(chan interface{}, 1)
			ch <- "test"
			close(ch)
			channels[i] = ch
		}

		done := make(chan interface{})
		defer close(done)

		out := FanIn(done, channels...)
		if got := <-out; got != "test" {
			t.Fatalf("expected 'test', got %v", got)
		}
	})

	t.Run("no channel", func(t *testing.T) {
		done := make(chan interface{})
		defer close(done)

		out := FanIn(done)
		_, open := <-out
		if open {
			t.Fatalf("expected output channel to be closed")
		}
	})

	t.Run("done channel", func(t *testing.T) {
		ch1 := make(chan interface{}, 1)
		ch1 <- "test"
		close(ch1)

		ch2 := make(chan interface{})
		done := make(chan interface{})

		out := FanIn(done, ch1, ch2)

		if got := <-out; got != "test" {
			t.Fatalf("excepted 'test', got %v", got)
		}

		close(done)
		close(ch2)

		_, open := <-out
		if open {
			t.Fatalf("expected output channel to be closed after done was closed")
		}
	})
}

func TestFanOut(t *testing.T) {
	t.Run("single worker", func(t *testing.T) {
		ch := make(chan string, 1)
		ch <- "test"
		close(ch)

		done := make(chan interface{})
		defer close(done)

		processer := func(input string) string { return input }
		out := FanOut(done, ch, 1, processer)
		if got := <-out; got != "test" {
			t.Fatalf("expected 'test', got %v", got)
		}
	})

	t.Run("more worker", func(t *testing.T) {
		ch := make(chan string, 3)
		ch <- "test"
		ch <- "test"
		ch <- "test"
		close(ch)

		done := make(chan interface{})
		defer close(done)

		processer := func(input string) string { return input }
		out := FanOut(done, ch, 5, processer)

		for i := 0; i < 3; i++ {
			if got := <-out; got != "test" {
				t.Fatalf("expected 'test', got %v", got)
			}
		}
	})

	t.Run("less worker", func(t *testing.T) {
		ch := make(chan string, 3)
		ch <- "test"
		ch <- "test"
		ch <- "test"
		close(ch)

		done := make(chan interface{})
		defer close(done)

		processer := func(input string) string { return input }
		out := FanOut(done, ch, 1, processer)

		for i := 0; i < 3; i++ {
			if got := <-out; got != "test" {
				t.Fatalf("expected 'test', got %v", got)
			}
		}
	})

	t.Run("no worker", func(t *testing.T) {
		ch := make(chan string, 3)
		ch <- "test"
		ch <- "test"
		ch <- "test"
		close(ch)

		done := make(chan interface{})
		defer close(done)

		processer := func(input string) string { return input }
		out := FanOut(done, ch, 0, processer)

		_, open := <-out
		if open {
			t.Fatalf("expected output channel to be closed")
		}
	})

	t.Run("custom processer", func(t *testing.T) {
		ch := make(chan int, 1)
		ch <- 1
		close(ch)

		done := make(chan interface{})
		defer close(done)

		processer := func(input int) int { return input * 2 }
		out := FanOut(done, ch, 1, processer)

		if got := <-out; got != 2 {
			t.Fatalf("excepted 2, got %d", got)
		}
	})
}
