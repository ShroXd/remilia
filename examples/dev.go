package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"

	"github.com/PuerkitoBio/goquery"
	"github.com/ShroXd/remilia"
)

func main() {
	f, err := os.Create("out/cpu.pprof")
	if err != nil {
		panic(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	work()

	pprof.StopCPUProfile()
	f.Close()

	f, err = os.Create("out/mem.pprof")
	if err != nil {
		panic(err)
	}
	pprof.WriteHeapProfile(f)
	f.Close()
}

func work() {

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
