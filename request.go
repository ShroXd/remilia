package remilia

import (
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
		return nil, err
	}

	parsedURL, _ := url.Parse(urlString)

	return &Request{
		internal: req,
		URL:      parsedURL,
	}, nil
}

func (req *Request) Unpack() (*http.Request, error) {
	return req.internal, nil
}
