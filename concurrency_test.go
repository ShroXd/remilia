package remilia

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestFanIn(t *testing.T) {
	t.Run("no channel", func(t *testing.T) {
		done := make(chan struct{})
		defer close(done)

		out := FanIn[any](done)
		_, open := <-out
		assert.False(t, open, "expected output channel to be closed")
	})

	t.Run("done channel", func(t *testing.T) {
		ch1 := make(chan string, 1)
		ch1 <- "test"
		close(ch1)

		ch2 := make(chan string)
		done := make(chan struct{})

		out := FanIn(done, ch1, ch2)

		assert.Equal(t, "test", <-out, "expected 'test'")

		close(done)
		close(ch2)

		_, open := <-out
		assert.False(t, open, "expected output channel to be closed after done was closed")
	})

	t.Run("single channel", func(t *testing.T) {
		ch := make(chan string, 1)
		ch <- "test"
		close(ch)

		done := make(chan struct{})
		defer close(done)

		out := FanIn[string](done, ch)
		assert.Equal(t, "test", <-out, "expected 'test'")
	})

	t.Run("multiple channels", func(t *testing.T) {
		num := 10
		channels := make([]<-chan string, num)
		for i := 0; i < num; i++ {
			ch := make(chan string, 1)
			ch <- "test"
			close(ch)
			channels[i] = ch
		}

		done := make(chan struct{})
		defer close(done)

		out := FanIn(done, channels...)
		assert.Equal(t, "test", <-out, "expected 'test'")
	})
}

func TestTee(t *testing.T) {
	t.Run("normal operation", func(t *testing.T) {
		var wg sync.WaitGroup
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		in := make(chan int)
		out1, out2 := Tee(ctx, in, &wg)

		go func() {
			in <- 1
			close(in)
		}()

		select {
		case v1 := <-out1:
			assert.Equal(t, 1, v1, "expected 42")
			select {
			case v2 := <-out2:
				assert.Equal(t, 1, v2, "expected 42")
			case <-ctx.Done():
				t.Error("Timed out waiting for value from second output channel")
			}
		case <-ctx.Done():
			t.Error("Timed out waiting for value from first output channel")
		}
	})
}
