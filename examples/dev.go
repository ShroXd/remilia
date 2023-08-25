package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/url"
	"remilia"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func main() {
	scraper := remilia.New("https://www.23qb.net/lightnovel/", remilia.ConcurrentNumber(1))

	scraper.UseURL(".pagelink a", func(s *goquery.Selection) *url.URL {
		href, _ := s.Attr("href")
		url, _ := url.Parse(href)

		return url
	}).UseHTML("h3 a", func(s *goquery.Selection) interface{} {
		return s.Text()
	}, func(data <-chan interface{}) {
		for v := range data {
			utf8Str, err := GB2312ToUTF8(v.(string))
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("Get data in data consumer: ", utf8Str)
		}
	}).AddToChain()

	err := scraper.Start()
	if err != nil {
		log.Print(err)
	}
}

func GB2312ToUTF8(s string) (string, error) {
	reader := transform.NewReader(bytes.NewReader([]byte(s)), simplifiedchinese.GB18030.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
}
