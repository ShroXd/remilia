package main

import (
	"fmt"
	"log"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/ShroXd/remilia"
	"github.com/ShroXd/remilia/pkg/logger"
)

func main() {
	htmlParser := func(s *goquery.Selection) interface{} {
		return s.Text()
	}

	contentConsumer := func(data <-chan interface{}) {
		for v := range data {
			fmt.Println("Receive data: ", v)
		}
	}

	// TODO:
	// 1. use chain calling to setup the remilia instance instead of using such stupid way
	// 2. return fn controling concurreny to user in AddToChain or other API
	// 3. support middleware for request
	// 4. support retry
	// 5. support basic client configurations
	scraper := remilia.New(
		"https://go.dev/doc/",
		remilia.ConcurrentNumber(10),
		remilia.ConsoleLog(logger.ErrorLevel),
	)
	scraper.UseURL("h3 a", func(s *goquery.Selection) *url.URL {
		path, _ := s.Attr("href")
		url, _ := url.Parse("https://go.dev" + path)

		return url
	}).UseHTML("h3", htmlParser, contentConsumer).AddToChain()

	scraper.UseHTML("h2", htmlParser, contentConsumer).AddToChain()

	err := scraper.Start()
	if err != nil {
		log.Print(err)
	}
}
