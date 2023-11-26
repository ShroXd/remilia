package main

import (
	"fmt"
	"log"

	"github.com/PuerkitoBio/goquery"
	"github.com/ShroXd/remilia"
)

func main() {
	// htmlParser := func(s *goquery.Selection) interface{} {
	// 	return s.Text()
	// }

	// contentConsumer := func(data <-chan interface{}) {
	// 	for v := range data {
	// 		fmt.Println("Receive data: ", v)
	// 	}
	// }

	// TODO:
	// 2. return fn controling concurreny to user in End or other API
	// 3. support middleware for request
	// 4. support retry
	// 5. support basic client configurations
	// 6. expand or shrink the number of goroutines according to the tasks in the queue
	// client := remilia.New(
	// 	"https://go.dev/doc/",
	// 	remilia.ConcurrentNumber(10),
	// 	remilia.ConsoleLog(remilia.ErrorLevel),
	// )
	// client.R().UseURL("h3 a", func(s *goquery.Selection) *url.URL {
	// 	path, _ := s.Attr("href")
	// 	url, _ := url.Parse("https://go.dev" + path)

	// 	return url
	// }).UseHTML("h3", htmlParser, contentConsumer).Visit("https://www.google.com/")

	// scraper.R().UseHTML("h2", htmlParser, contentConsumer).Visit("https://go.dev/doc/")

	// err := scraper.Start()
	// if err != nil {
	// 	log.Print(err)
	// }

	// r := remilia.New(
	// 	"https://go.dev/doc/",
	// 	remilia.ConcurrentNumber(10),
	// 	remilia.ConsoleLog(remilia.ErrorLevel),
	// )

	// client := r.C().SetProxy("http://127.0.0.1:8866")

	// req, err := http.NewRequest("GET", "", nil)
	// if err != nil {
	// 	panic(err)
	// }

	// resp, err := client.Process("http://localhost:8080")
	// if err != nil {
	// 	panic(err)
	// }
	// defer resp.Body.Close()

	// // TODO: learn how to use buffer
	// body, _ := io.ReadAll(resp.Body)

	// fmt.Println(string(body))

	htmlParser := func(doc *goquery.Document) interface{} {
		h1Text := doc.Find("h1").First().Text()
		log.Printf("H1 Tag Content: %s\n", h1Text)

		return h1Text
	}

	contentConsumer := func(data <-chan interface{}) {
		fmt.Println("In the consumer")
		for v := range data {
			fmt.Println("Receive data: ", v)
		}
	}

	c := remilia.C().SetProxy("http://127.0.0.1:8866")

	step1 := remilia.NewStage(nil, htmlParser, contentConsumer)

	scrapy := remilia.New(c, step1)
	result, err := scrapy.Process("http://localhost:8080")
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	fmt.Println("Scraping result: ", result)
}
