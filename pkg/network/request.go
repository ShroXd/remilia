package network

import (
	"io"
	"net/http"
)

type Request struct {
	internal *http.Request
	Host     string
}

func NewRequest(method, url string, body io.Reader) (*Request, error) {
	r, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	return &Request{
		internal: r,
		Host:     r.Host,
	}, nil
}
