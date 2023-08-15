package network

import (
	"io"
	"net/http"
	"remilia/pkg/logger"

	"go.uber.org/zap"
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
		logger.Error("Error creating request", zap.Error(err))
	}

	req.Header = *r.Header

	resp, err := r.Client.Do(req)
	if err != nil {
		logger.Error("Error sending request", zap.Error(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("Unexpected response status code", zap.Error(err))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body", zap.Error(err))
	}

	return string(bodyBytes)
}
