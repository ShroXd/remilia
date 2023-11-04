package remilia

import "sync"

func FanIn[T any](
	done <-chan struct{},
	channels ...<-chan T,
) <-chan T {
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
