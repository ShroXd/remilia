package main

import (
	"fmt"
	"log"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/ShroXd/remilia"
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

	scraper := remilia.New("https://go.dev/doc/", remilia.ConcurrentNumber(10))
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
