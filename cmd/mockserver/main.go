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

// func main() {
// 	http.HandleFunc("/", handler)
// 	http.HandleFunc("/page/", pageHandler)
// 	http.HandleFunc("/content/", contentHandler)

// 	log.Println("Mock server is running on http://localhost:6657")
// 	if err := http.ListenAndServe(":6657", nil); err != nil {
// 		log.Fatalf("Failed to start server: %v", err)
// 	}
// }

// // handler decides which scenario to use based on the request path
// func handler(w http.ResponseWriter, r *http.Request) {
// 	switch r.URL.Path {
// 	case "/largehtml":
// 		generateLargeHTML(w)
// 	default:
// 		http.NotFound(w, r)
// 	}
// }

// func pageHandler(w http.ResponseWriter, r *http.Request) {
// 	pageSize := 50000
// 	pageNumStr := r.URL.Path[len("/page/"):]
// 	pageNum, err := strconv.Atoi(pageNumStr)
// 	if err != nil || pageNum < 1 || pageNum > pageSize {
// 		// pageNum不在1到40000的范围内，返回错误消息或重定向
// 		http.Error(w, "Page not found", http.StatusNotFound)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "text/html")
// 	fmt.Fprintf(w, "<html><body><div>%d<div>", pageSize)
// 	for i := 1; i <= 10000; i++ {
// 		// 假设每个a标签指向的内容页面URL为/content/{i}，其中{i}是a标签的序号
// 		fmt.Fprintf(w, `<a href="/content/%d">Link %d</a><br>`, i, i)
// 	}
// 	fmt.Fprintf(w, "</body></html>")
// }

// func contentHandler(w http.ResponseWriter, r *http.Request) {
// 	generateRandomContentPage(w)
// }

// func generateLargeHTML(w http.ResponseWriter) {
// 	w.Header().Set("Content-Type", "text/html")
// 	fmt.Fprintf(w, "<html><head><title>Large HTML</title></head><body>\n")

// 	for i := 1; i <= 40000; i++ {
// 		switch rand.Intn(5) {
// 		case 0:
// 			fmt.Fprintf(w, "<h1>Header %d</h1>\n", i)
// 		case 1:
// 			fmt.Fprintf(w, "<p>Paragraph %d</p>\n", i)
// 		case 2:
// 			fmt.Fprintf(w, "<a href='http://example.com/%d'>Link %d</a>\n", i, i)
// 		case 3:
// 			fmt.Fprintf(w, "<ul><li>List Item %d</li></ul>\n", i)
// 		case 4:
// 			fmt.Fprintf(w, "<div>Div %d</div>\n", i)
// 		}
// 	}

// 	fmt.Fprintf(w, "</body></html>\n")
// }

// func generateRandomContentPage(w http.ResponseWriter) {
// 	tags := []string{"p", "h1", "h2", "h3", "div"}
// 	w.Header().Set("Content-Type", "text/html")
// 	fmt.Fprintf(w, "<html><body>")
// 	for i := 1; i <= 10000; i++ {
// 		tag := tags[rand.Intn(len(tags))]
// 		fmt.Fprintf(w, "<%s>%s</%s>", tag, randomString(), tag)
// 	}
// 	fmt.Fprintf(w, "</body></html>")
// }

// func randomString() string {
// 	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
// 	b := make([]byte, 10)
// 	for i := range b {
// 		b[i] = letters[rand.Intn(len(letters))]
// 	}
// 	return string(b)
// }
