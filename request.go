package remilia

import (
	"fmt"
	"net/http"
	"net/url"
)

type Request struct {
	URL      *url.URL
	internal *http.Request
	logger   Logger
}

func NewRequest(urlString string) (*Request, error) {
	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	return &Request{
		internal: req,
		URL:      parsedURL,
	}, nil
}

func (req *Request) Unpack() (*http.Request, error) {
	return req.internal, nil
}
