package concurrency

import "sync"

func FanIn(
	done <-chan interface{},
	channels ...<-chan interface{},
) <-chan interface{} {
	var wg sync.WaitGroup
	output := make(chan interface{})

	multiplex := func(ch <-chan interface{}) {
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
