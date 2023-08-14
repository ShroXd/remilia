package network

import (
	"io"
	"log"
	"net/http"
)

type Request struct {
	Header *http.Header
	Client *http.Client
	URL    string
	Method string
}

func New(method, URL string, header *http.Header) (*Request, error) {
	client := &http.Client{}

	return &Request{
		URL:    URL,
		Header: header,
		Client: client,
		Method: method,
	}, nil
}

func (r *Request) Visit() string {
	req, err := http.NewRequest(r.Method, r.URL, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	req.Header = *r.Header

	resp, err := r.Client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected response status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	return string(bodyBytes)
}
