package remilia

import (
	"net/http"
)

type Request struct {
	internal *http.Request
	logger   Logger
}

func NewRequest(url string) (*Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return &Request{
		internal: req,
	}, nil
}

func (req *Request) Unpack() (*http.Request, error) {
	return req.internal, nil
}
