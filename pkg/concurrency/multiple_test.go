package concurrency

import (
	"sync"
	"testing"
)

func TestFanIn(t *testing.T) {
	t.Run("single channel", func(t *testing.T) {
		ch := make(chan interface{}, 1)
		ch <- "test"
		close(ch)

		done := make(chan struct{})
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

		done := make(chan struct{})
		defer close(done)

		out := FanIn(done, channels...)
		if got := <-out; got != "test" {
			t.Fatalf("expected 'test', got %v", got)
		}
	})

	t.Run("no channel", func(t *testing.T) {
		done := make(chan struct{})
		defer close(done)

		out := FanIn[any](done)
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
		done := make(chan struct{})

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

func TestOrDone(t *testing.T) {
	t.Run("closes output when done is closed first", func(t *testing.T) {
		done := make(chan struct{})
		ch := make(chan int)
		out := OrDone(done, ch)

		close(done)
		_, ok := <-out
		if ok {
			t.Fatalf("expected channel to be closed")
		}
	})

	t.Run("sends data from ch to output", func(t *testing.T) {
		done := make(chan struct{})
		ch := make(chan int)
		out := OrDone(done, ch)

		go func() {
			ch <- 1
			close(ch)
		}()

		v, ok := <-out
		if !ok || v != 1 {
			t.Fatalf("expected 1, got %d", v)
		}

		_, ok = <-out
		if ok {
			t.Fatalf("expected channel to be closed after input channel is closed")
		}
	})

	t.Run("closes output when done is closed while waiting", func(t *testing.T) {
		done := make(chan struct{})
		ch := make(chan int)
		out := OrDone(done, ch)

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			close(done)
		}()

		wg.Wait()
		_, ok := <-out
		if ok {
			t.Fatalf("expected channel to be closed due to done being closed")
		}
	})
}

func TestTee(t *testing.T) {
	t.Run("send data to both outputs", func(t *testing.T) {
		done := make(chan struct{})
		in := make(chan int)
		out1, out2 := Tee(done, in)

		go func() {
			in <- 1
			close(in)
		}()

		if v := <-out1; v != 1 {
			t.Fatalf("expected 1 from out1, got %d", v)
		}
		if v := <-out2; v != 1 {
			t.Fatalf("expected 1 from out2, got %d", v)
		}

		// Ensure both channels are closed
		_, ok1 := <-out1
		_, ok2 := <-out2
		if ok1 || ok2 {
			t.Fatalf("expected both channels to be clsoed")
		}
	})

	t.Run("stops sending data when done is closed", func(t *testing.T) {
		done := make(chan struct{})
		in := make(chan int)
		out1, out2 := Tee(done, in)

		go func() {
			in <- 1
			close(done)
			in <- 2
			close(in)
		}()

		if v := <-out1; v != 1 {
			t.Fatalf("expected 1 from out1, got %d", v)
		}
		if v := <-out2; v != 1 {
			t.Fatalf("expected 1 from out2, got %d", v)
		}

		// Test if values after closing done are not sent
		_, ok1 := <-out1
		_, ok2 := <-out2
		if ok1 || ok2 {
			t.Fatalf("expected both channels to be clsoed")
		}
	})
}
