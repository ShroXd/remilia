package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
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

	firstParser := func(in *goquery.Document, put remilia.Put[string], chew remilia.Put[string]) {
		in.Find("a").Each(func(i int, s *goquery.Selection) {
			href, ok := s.Attr("href")
			if ok {
				put(baseURL + href)
			}
		})

		// chew(baseURL + "/page/2")
	}

	secondParser := func(in *goquery.Document, put remilia.Put[string], chew remilia.Put[string]) {
		title := in.Find("p").First().Text()
		fmt.Println("Article title: ", title)
	}

	producer := rem.Just(initURL)
	first := rem.Unit(firstParser)
	second := rem.Unit(secondParser)

	if err := rem.Do(producer, first, second); err != nil {
		fmt.Println("Error: ", err)
	}
}

func startCPUProfile() {
	f, err := createFileWithDir("out/cpu.pprof")
	if err != nil {
		panic(err)
	}
	pprof.StartCPUProfile(f)
}

func stopCPUProfile() {
	pprof.StopCPUProfile()
}

func writeMemProfile() {
	f, err := createFileWithDir("out/mem.pprof")
	if err != nil {
		panic(err)
	}
	pprof.WriteHeapProfile(f)
	f.Close()
}

func writeBlockProfile() {
	f, err := createFileWithDir("out/block.pprof")
	if err != nil {
		// panic(err)
	}
	pprof.Lookup("block").WriteTo(f, 0)
	f.Close()
}

func writeGoroutineProfile() {
	f, err := createFileWithDir("out/goroutine.pprof")
	if err != nil {
		// panic(err)
	}
	pprof.Lookup("goroutine").WriteTo(f, 0)
	f.Close()
}

func writeThreadcreateProfile() {
	f, err := createFileWithDir("out/threadcreate.pprof")
	if err != nil {
		// panic(err)
	}
	pprof.Lookup("threadcreate").WriteTo(f, 0)
	f.Close()
}

func writeMutexProfile() {
	f, err := createFileWithDir("out/mutex.pprof")
	if err != nil {
		// panic(err)
	}
	pprof.Lookup("mutex").WriteTo(f, 0)
	f.Close()
}

func createFileWithDir(filePath string) (*os.File, error) {
	dirPath := filepath.Dir(filePath)
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return nil, err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	return file, nil
}
