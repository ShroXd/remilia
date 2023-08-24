package main

import (
	"fmt"
	"log"
	"remilia"
)

func main() {
	scraper := remilia.New("https://www.23qb.net/lightnovel/", remilia.ConcurrentNumber(1))

	scraper.Parse(".pagelink", func(r string) {
		fmt.Println("Parse result: ", r)
	})

	err := scraper.Start()
	if err != nil {
		log.Print(err)
	}
}
