package concurrency

import (
	"remilia/pkg/logger"
	"sync"

	"go.uber.org/zap"
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

func FanOut[T any, U any](
	done <-chan struct{},
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
				logger.Debug("Push url to mid channel", zap.String("channel", "middle"), zap.String("function", "FanOut"))
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
