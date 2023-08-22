package utils

func ArrayToChannel[T any](array []T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for _, element := range array {
			out <- element
		}
	}()

	return out
}
