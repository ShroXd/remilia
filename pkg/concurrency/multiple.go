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

func FanOut[T any, U any](
	done <-chan interface{},
	input <-chan T,
	workerCount int,
	processFn func(T) U,
) <-chan U {
	var wg sync.WaitGroup
	output := make(chan U)

	worker := func() {
		defer wg.Done()

		for {
			select {
			case value, ok := <-input:
				if !ok {
					return
				}
				processedValue := processFn(value)
				select {
				case output <- processedValue:
				case <-done:
					return
				}
			case <-done:
				return
			}
		}
	}

	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker()
	}

	go func() {
		wg.Wait()
		close(output)
	}()

	return output
}
