package main

import (
	"fmt"
	"log"
	"net/http"
	"remilia"
	"remilia/pkg/concurrency"
	"remilia/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	scraper := remilia.New("https://go.dev", remilia.ConcurrentNumber(5))

	err := scraper.Start()
	if err != nil {
		log.Print(err)
	}
}

func test() {
	done := make(chan struct{})
	defer close(done)

	url := "https://go.dev"
	num := 50

	channels := make([]<-chan interface{}, num)

	for i := 0; i < num; i++ {
		ch := make(chan interface{})
		channels[i] = ch

		go fetchURL(url, ch, done)
	}

	result := concurrency.FanIn(done, channels...)

	for i := 0; i < num; i++ {
		htmlContent := <-result
		fmt.Println("Received html content: ", htmlContent)
	}
}

func fetchURL(url string, out chan<- interface{}, done <-chan struct{}) {
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
