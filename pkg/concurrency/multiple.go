package concurrency

import (
	"sync"
)

// TODO: add timeout for each function in select statement
func FanIn[T any](
	done <-chan struct{},
	channels ...<-chan T,
) <-chan T {
	var wg sync.WaitGroup
	output := make(chan T)

	multiplex := func(ch <-chan T) {
		defer wg.Done()

		for i := range ch {
			select {
			case <-done:
				return
			case output <- i:
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

func OrDone[T any](done <-chan struct{}, ch <-chan T) <-chan T {
	valStream := make(chan T)

	go func() {
		defer close(valStream)

		for {
			select {
			case <-done:
				return
			case v, ok := <-ch:
				if !ok {
					return
				}
				select {
				case valStream <- v:
				case <-done:
				}
			}
		}
	}()

	return valStream
}

func Tee[T any](done <-chan struct{}, in <-chan T) (_, _ <-chan T) {
	out1 := make(chan T)
	out2 := make(chan T)

	go func() {
		defer close(out1)
		defer close(out2)

		for v := range OrDone(done, in) {
			// Create shadowed versions of out1 and out2 for this iteration to ensure each value is sent only once to each channel.
			// By doing so, even if we set one of the shadowed variables to nil after sending a value,
			// the next loop iteration will reset using the original channels, ensuring efficient value distribution.
			var out1, out2 = out1, out2
			for i := 0; i < 2; i++ {
				select {
				case <-done:
				case out1 <- v:
					out1 = nil
				case out2 <- v:
					out2 = nil
				}
			}
		}
	}()

	return out1, out2
}
