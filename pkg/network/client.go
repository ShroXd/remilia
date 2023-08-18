package network

import "net/http"

type Client struct {
	internal *http.Client
	Limit    Limit
}

func NewClient() *Client {
	return &Client{
		internal: &http.Client{},
	}
}
