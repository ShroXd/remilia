package main

import (
	"fmt"
	"log"

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

	scraper := remilia.New("https://go.dev/", remilia.ConcurrentNumber(10))
	scraper.UseHTML(".WhyGo-reasonText p", htmlParser, contentConsumer).AddToChain()

	err := scraper.Start()
	if err != nil {
		log.Print(err)
	}
}
