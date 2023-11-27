package remilia

import (
	"context"
	"sync"
)

func FanIn[T any](done <-chan struct{}, channels ...<-chan T) <-chan T {
	var wg sync.WaitGroup
	output := make(chan T)

	multiplex := func(ch <-chan T) {
		defer wg.Done()

		for val := range ch {
			select {
			case <-done:
				return
			case output <- val:
			}
		}
	}

	wg.Add(len(channels))
	for _, ch := range channels {
		go multiplex(ch)
	}

	go func() {
		wg.Wait()
		close(output)
	}()

	return output
}

func orDone[T any](c <-chan T) <-chan T {
	valStream := make(chan T)
	go func() {
		defer close(valStream)
		for {
			select {
			case v, ok := <-c:
				if ok == false {
					return
				}
				select {
				case valStream <- v:
				}
			}
		}
	}()
	return valStream
}

func Tee[T any](ctx context.Context, in <-chan T) (<-chan T, <-chan T) {
	out1 := make(chan T)
	out2 := make(chan T)
	go func() {
		defer close(out1)
		defer close(out2)

		for val := range orDone[T](in) {
			var out1, out2 = out1, out2
			for i := 0; i < 2; i++ {
				select {
				case <-ctx.Done():
					return
				case out1 <- val:
					out1 = nil
				case out2 <- val:
					out2 = nil
				}
			}
		}
	}()

	return out1, out2
}
