package network

import (
	"net/http"
	"net/url"
)

type Request struct {
	URL    *url.URL
	Method string

	internal *http.Request
}

func NewRequest(method, startURL string) (*Request, error) {
	u, err := url.Parse(startURL)
	if err != nil {
		return nil, err
	}

	return &Request{
		URL:    u,
		Method: method,
	}, nil
}

// Build returns http request
func (req *Request) Build() (*http.Request, error) {
	request, err := http.NewRequest(req.Method, req.URL.String(), nil)
	if err != nil {
		return nil, err
	}

	return request, err
}

// Host returns host of given url
func (req *Request) Host() string {
	return req.URL.Host
}
