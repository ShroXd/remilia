package main

import (
	"fmt"
	"net/http"
	"remilia/pkg/concurrency"
	"remilia/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	done := make(chan interface{})
	defer close(done)

	ch1 := make(chan interface{})
	ch2 := make(chan interface{})
	ch3 := make(chan interface{})

	go fetchURL("https://go.dev", ch1, done)
	go fetchURL("https://go.dev", ch2, done)
	go fetchURL("https://go.dev", ch3, done)

	result := concurrency.FanIn(done, ch1, ch2, ch3)

	for i := 0; i < 3; i++ {
		htmlContent := <-result
		fmt.Println("Received html content: ", htmlContent)
	}
}

func fetchURL(url string, out chan<- interface{}, done <-chan interface{}) {
	resp, err := http.Get(url)
	if err != nil {
		logger.Error("Request error", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	// bodyBytes, err := io.ReadAll(resp.Body)
	//if err != nil {
	//	logger.Error("Failed to read response body", zap.Error(err))
	//}
	// out <- string(bodyBytes)
	out <- resp.StatusCode

	select {
	case <-done:
		return
	default:
	}
}
