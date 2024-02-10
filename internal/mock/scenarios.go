package scenarios

import (
	"fmt"
	"net/http"
)

// generateLargeHTML generates a large HTML document with many h1 tags
func GenerateLargeHTML(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><head><title>Large HTML</title></head><body>\n")
	for i := 1; i <= 40000; i++ {
		fmt.Fprintf(w, "<h1>Header %d</h1>\n", i)
	}
	fmt.Fprintf(w, "</body></html>\n")
}
