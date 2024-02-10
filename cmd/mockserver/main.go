package main

import (
	"log"
	"net/http"

	scenarios "github.com/ShroXd/remilia/internal/mock"
)

func main() {
	http.HandleFunc("/", handler)

	log.Println("Mock server is running on http://localhost:6657")
	if err := http.ListenAndServe(":6657", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// handler decides which scenario to use based on the request path
func handler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/largehtml":
		scenarios.GenerateLargeHTML(w)
	default:
		http.NotFound(w, r)
	}
}
