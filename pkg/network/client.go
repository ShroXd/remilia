package network

import "net/http"

type Client struct {
	Limit Limit

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
