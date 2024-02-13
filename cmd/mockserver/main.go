package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	scenarios "github.com/ShroXd/remilia/internal/mock"
)

func main() {
	http.HandleFunc("/", unifiedHandler)

	log.Println("Mock server is running on http://localhost:6657")
	if err := http.ListenAndServe(":6657", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func unifiedHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/":
		scenarios.GenerateHomePage(w)
	case strings.HasPrefix(r.URL.Path, "/page/"):
		pageHandler(w, r)
	case strings.HasPrefix(r.URL.Path, "/content/"):
		contentHandler(w, r)
	case r.URL.Path == "/largehtml":
		scenarios.GenerateLargeHTML(w)
	default:
		http.NotFound(w, r)
	}
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	pageSize := 50000
	pageNumStr := r.URL.Path[len("/page/"):]
	pageNum, err := strconv.Atoi(pageNumStr)
	if err != nil || pageNum < 1 || pageNum > pageSize {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><body><div>%d<div>", pageSize)
	for i := 1; i <= 10000; i++ {
		fmt.Fprintf(w, `<a href="/content/%d">Link %d</a><br>`, i, i)
	}
	fmt.Fprintf(w, "</body></html>")
}

func contentHandler(w http.ResponseWriter, r *http.Request) {
	scenarios.GenerateRandomContentPage(w)
}
