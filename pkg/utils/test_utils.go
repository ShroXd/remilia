package utils

func GenerateData(n int) chan int {
	out := make(chan int)

	go func() {
		defer close(out)
		for i := 0; i < n; i++ {
			out <- i
		}
	}()

	return out
}
