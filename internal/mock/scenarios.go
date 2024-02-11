package scenarios

import (
	"fmt"
	"math/rand"
	"net/http"
)

func GenerateHomePage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><body>Welcome to the Mock Server</body></html>")
}

func GenerateLargeHTML(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><head><title>Large HTML</title></head><body>\n")

	for i := 1; i <= 40000; i++ {
		switch rand.Intn(5) {
		case 0:
			fmt.Fprintf(w, "<h1>Header %d</h1>\n", i)
		case 1:
			fmt.Fprintf(w, "<p>Paragraph %d</p>\n", i)
		case 2:
			fmt.Fprintf(w, "<a href='http://example.com/%d'>Link %d</a>\n", i, i)
		case 3:
			fmt.Fprintf(w, "<ul><li>List Item %d</li></ul>\n", i)
		case 4:
			fmt.Fprintf(w, "<div>Div %d</div>\n", i)
		}
	}

	fmt.Fprintf(w, "</body></html>\n")
}

func GenerateRandomContentPage(w http.ResponseWriter) {
	tags := []string{"p", "h1", "h2", "h3", "div"}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><body>")
	for i := 1; i <= 10000; i++ {
		tag := tags[rand.Intn(len(tags))]
		fmt.Fprintf(w, "<%s>%s</%s>", tag, randomString(), tag)
	}
	fmt.Fprintf(w, "</body></html>")
}

func randomString() string {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 10)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
