package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Example HTML response
		htmlContent := `
			<!DOCTYPE html>
			<html>
				<head>
					<title>Test Page</title>
				</head>
				<body>
					<h1>Welcome to the Test Page</h1>
					<p>This is a paragraph in your mock server.</p>
				</body>
			</html>
		`

		// Set header for HTML content and write response
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, htmlContent)
	})

	log.Println("Mock server running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
