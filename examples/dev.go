package main

import (
	"fmt"
	_ "net/http/pprof"

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

	// f, err := os.Create("trace.out")
	// if err != nil {
	// 	panic(err)
	// }
	// defer f.Close()

	// err = trace.Start(f)
	// if err != nil {
	// 	panic(err)
	// }
	// defer trace.Stop()

	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	// urlGenerator := func(doc *goquery.Document) string {
	// 	return "http://localhost:8080"
	// 	// return ""
	// }

	// htmlParser := func(doc *goquery.Document) interface{} {
	// 	h1Text := doc.Find("h1").First().Text()
	// 	log.Printf("H1 Tag Content: %s\n", h1Text)

	// 	return h1Text
	// }

	// contentConsumer := func(data <-chan interface{}) {
	// 	fmt.Println("In the consumer")
	// 	for v := range data {
	// 		fmt.Println("Receive data: ", v)
	// 	}
	// }

	// c := remilia.C()

	// step1 := remilia.NewStage(urlGenerator, htmlParser, contentConsumer)
	// step2 := remilia.NewStage(urlGenerator, htmlParser, contentConsumer)
	// step3 := remilia.NewFinalStage(htmlParser, contentConsumer)

	// scrapy := remilia.New(c, step1, step2, step3)
	// scrapy.Process("http://localhost:8080", context.TODO())

	// // TODO: only need to wait for urlGenerator
	// // the scrapy need the new url to run the pipeline
	// // but the data consumer and html parser should be controled by user
	// scrapy.Wait()

	rem, _ := remilia.New()

	initURL := "https://go.dev/doc/"

	firstParser := func(in *goquery.Document, put remilia.Put[string]) {
		in.Find("#developing-modules h3 a").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if exists {
				// fmt.Println(href)
			}

			generated := "https://go.dev" + href
			put(generated)
		})
	}

	secondParser := func(in *goquery.Document, put remilia.Put[string]) {
		title := in.Find("h1").First().Text()
		fmt.Println("Article title: ", title)
	}

	producer := rem.Just(initURL)
	first := rem.Relay(firstParser)
	second := rem.Relay(secondParser)

	if err := rem.Do(producer, first, second); err != nil {
		fmt.Println("Error: ", err)
	}
}
