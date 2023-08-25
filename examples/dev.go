package main

import (
	"fmt"
	"log"
	"net/url"
	"remilia"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	scraper := remilia.New("https://www.23qb.net/lightnovel/", remilia.ConcurrentNumber(1))

	scraper.UseURL(".pagelink a", func(s *goquery.Selection) *url.URL {
		href, _ := s.Attr("href")
		url, _ := url.Parse(href)

		return url
	}).AddToChain()

	scraper.UseURL(".pagelink a", func(s *goquery.Selection) *url.URL {
		href, _ := s.Attr("href")
		url, _ := url.Parse(href)

		return url
	}).UseHTML("h3 a", func(s *goquery.Selection) interface{} {
		return s.Text()
	}, func(data <-chan interface{}) {
		for v := range data {
			fmt.Println("Get data in data consumer: ", v)
		}
	}).AddToChain()

	err := scraper.Start()
	if err != nil {
		log.Print(err)
	}
}
