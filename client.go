package remilia

import (
	"errors"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	BaseURL string
	Header  http.Header
	Timeout time.Duration

	internal *http.Client
	logger   Logger
}

func NewClient() *Client {
	transport := &http.Transport{}
	c := &Client{
		internal: &http.Client{Transport: transport},
	}
	return c
}

func (c *Client) SetBaseURL(url string) *Client {
	c.BaseURL = url
	return c
}

func (c *Client) SetHeader(headers map[string]string) *Client {
	for h, v := range headers {
		c.Header.Set(h, v)
	}
	return c
}

func (c *Client) SetTimeout(timeout time.Duration) *Client {
	c.Timeout = timeout
	return c
}

func (c *Client) SetProxy(proxyURL string) *Client {
	pURL, err := url.Parse(proxyURL)
	if err != nil {
		c.logger.Error("Error parsing proxy URL", LogContext{"error": err})
		return c
	}

	t, err := c.transport()
	if err != nil {
		c.logger.Error("Error getting transport", LogContext{"error": err})
		return c
	}

	t.Proxy = http.ProxyURL(pURL)
	return c
}

func (c *Client) SetLogger(logger Logger) *Client {
	c.logger = logger
	return c
}

func (c *Client) Execute(req *http.Request) (*http.Response, error) {
	return c.internal.Do(req)
}

func (c *Client) transport() (*http.Transport, error) {
	t, ok := c.internal.Transport.(*http.Transport)
	if !ok {
		return nil, errors.New("invalid transport instance")
	}

	return t, nil
}
