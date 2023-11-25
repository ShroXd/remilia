package remilia

import (
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type (
	RequestHook  func(*Client, *Request) error
	ResponseHook func(*Client, *Response) error
)

type Client struct {
	BaseURL string
	Header  http.Header
	Timeout time.Duration

	internal                *http.Client
	logger                  Logger
	preRequestHooks         []RequestHook
	udPreRequestHooks       []RequestHook
	udPreRequestHooksLock   sync.RWMutex
	postResponseHooks       []ResponseHook
	udPostResponseHooks     []ResponseHook
	udPostResponseHooksLock sync.RWMutex
}

func NewClient() *Client {
	transport := &http.Transport{}
	c := &Client{
		Header: http.Header{},
		internal: &http.Client{
			Transport: transport,
		},
	}
	return c
}

func (c *Client) SetBaseURL(url string) *Client {
	c.BaseURL = url
	return c
}

func (c *Client) SetHeaders(headers map[string]string) *Client {
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

func (c *Client) PreRequestHooks(hooks ...RequestHook) *Client {
	c.udPreRequestHooksLock.Lock()
	defer c.udPreRequestHooksLock.Unlock()

	c.udPreRequestHooks = append(c.udPreRequestHooks, hooks...)
	return c
}

func (c *Client) PostResponseHooks(hooks ...ResponseHook) *Client {
	c.udPostResponseHooksLock.Lock()
	defer c.udPostResponseHooksLock.Unlock()

	c.udPostResponseHooks = append(c.udPostResponseHooks, hooks...)
	return c
}

func (c *Client) Execute(request *Request) (*Response, error) {
	req, err := request.Unpack()
	if err != nil {
		return nil, err
	}

	resp, err := c.internal.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	response := &Response{
		internal: resp,
	}

	return response, nil
}

func (c *Client) transport() (*http.Transport, error) {
	t, ok := c.internal.Transport.(*http.Transport)
	if !ok {
		return nil, errors.New("invalid transport instance")
	}

	return t, nil
}
