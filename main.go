package main

import (
	"fmt"
	"log"
	"net/http"
	"remilia/pkg/network"
)

func main() {
	url := "https://go.dev"
	req, err := network.New("GET", url, &http.Header{})
	if err != nil {
		// TODO: handle this error
		log.Fatal(err)
	}

	htmlContent := req.Visit()
	fmt.Print(htmlContent)
}
