package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/PuerkitoBio/goquery"
	"github.com/ShroXd/remilia"
)

func main() {
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)
	startCPUProfile()
	defer stopCPUProfile()

	work()

	writeMemProfile()
	writeBlockProfile()
	writeGoroutineProfile()
	writeThreadcreateProfile()
	writeMutexProfile()
}

func work() {
	rem, _ := remilia.New()

	initURL := "http://localhost:6657/page/1"
	baseURL := "http://localhost:6657"

	firstParser := func(in *goquery.Document, put remilia.Put[string]) {
		in.Find("a").Each(func(i int, s *goquery.Selection) {
			href, ok := s.Attr("href")
			if ok {
				put(baseURL + href)
			}
		})
	}

	secondParser := func(in *goquery.Document, put remilia.Put[string]) {
		title := in.Find("p").First().Text()
		fmt.Println("Article title: ", title)
	}

	producer := rem.Just(initURL)
	first := rem.Relay(firstParser)
	second := rem.Relay(secondParser)

	if err := rem.Do(producer, first, second); err != nil {
		fmt.Println("Error: ", err)
	}
}

func startCPUProfile() {
	f, err := os.Create("out/cpu.pprof")
	if err != nil {
		panic(err)
	}
	pprof.StartCPUProfile(f)
}

func stopCPUProfile() {
	pprof.StopCPUProfile()
}

func writeMemProfile() {
	f, err := os.Create("out/mem.pprof")
	if err != nil {
		panic(err)
	}
	pprof.WriteHeapProfile(f)
	f.Close()
}

func writeBlockProfile() {
	f, err := os.Create("out/block.pprof")
	if err != nil {
		panic(err)
	}
	pprof.Lookup("block").WriteTo(f, 0)
	f.Close()
}

func writeGoroutineProfile() {
	f, err := os.Create("out/goroutine.pprof")
	if err != nil {
		panic(err)
	}
	pprof.Lookup("goroutine").WriteTo(f, 0)
	f.Close()
}

func writeThreadcreateProfile() {
	f, err := os.Create("out/threadcreate.pprof")
	if err != nil {
		panic(err)
	}
	pprof.Lookup("threadcreate").WriteTo(f, 0)
	f.Close()
}

func writeMutexProfile() {
	f, err := os.Create("out/mutex.pprof")
	if err != nil {
		panic(err)
	}
	pprof.Lookup("mutex").WriteTo(f, 0)
	f.Close()
}
