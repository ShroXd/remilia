package main

import (
	"fmt"
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
		logger.Error("Error")
	}

	htmlContent := req.Visit()
	fmt.Print(htmlContent)
	logger.Info("Good")
}
