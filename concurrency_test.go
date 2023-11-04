package remilia

import "testing"

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
