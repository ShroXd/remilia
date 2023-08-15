package main

import (
	"fmt"
	"log"
	"net/http"
	"remilia/pkg/logger"
	"remilia/pkg/network"
)

func init() {
	logger.New()
}

func main() {
	url := "https://go.dev"
	req, err := network.New("GET", url, &http.Header{})
	if err != nil {
		// TODO: handle this error
		log.Fatal(err)
	}

	htmlContent := req.Visit()
	fmt.Print(htmlContent)
	logger.Info("Here is the testing log")
}
