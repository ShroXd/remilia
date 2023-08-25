package main

import (
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
	}).AddToChain()

	// scraper.Use(func(s *goquery.Selection) *url.URL {
	// 	href, _ := s.Attr("href")
	// 	url, _ := url.Parse(href)

	// 	return url
	// })

	// scraper.Use(func(s *goquery.Selection) *url.URL {
	// 	href, _ := s.Attr("href")
	// 	url, _ := url.Parse(href)

	// 	return url
	// })

	err := scraper.Start()
	if err != nil {
		log.Print(err)
	}
}
