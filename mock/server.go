package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Example response
		response := "Hello, this is your mock server!"

		// Set header and write response
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, response)
	})

	log.Println("Mock server running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
