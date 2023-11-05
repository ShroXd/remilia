package remilia

import (
	"net/http"
	"time"
)

type Client struct {
	BaseURL  string
	Header   http.Header
	Timeout  time.Duration
	internal *http.Client
}

func NewClient() *Client {
	return &Client{
		internal: &http.Client{},
	}
}

func (c *Client) Visit(req *http.Request) (*http.Response, error) {
	return c.internal.Do(req)
}
