package remilia

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

type (
	RequestHook  func(*Client, *Request) error
	ResponseHook func(*Client, *Response) error
)

type Client struct {
	BaseURL string
	Header  http.Header
	Timeout time.Duration

	internal                *fasthttp.Client
	logger                  Logger
	preRequestHooks         []RequestHook
	udPreRequestHooks       []RequestHook
	udPreRequestHooksLock   sync.RWMutex
	postResponseHooks       []ResponseHook
	udPostResponseHooks     []ResponseHook
	udPostResponseHooksLock sync.RWMutex
}

func NewClient() *Client {
	return &Client{
		Header: http.Header{},
		internal: &fasthttp.Client{
			ReadTimeout:              500 * time.Millisecond,
			WriteTimeout:             500 * time.Millisecond,
			NoDefaultUserAgentHeader: true,
			Dial:                     fasthttpproxy.FasthttpHTTPDialer("127.0.0.1:8866"),
		},
	}
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
	return nil
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
	c.udPreRequestHooksLock.RLock()
	defer c.udPreRequestHooksLock.RUnlock()

	c.udPostResponseHooksLock.RLock()
	defer c.udPostResponseHooksLock.RUnlock()

	for _, fn := range c.preRequestHooks {
		if err := fn(c, request); err != nil {
			return nil, err
		}
	}

	for _, fn := range c.udPreRequestHooks {
		if err := fn(c, request); err != nil {
			return nil, err
		}
	}

	req := request.Build()
	req.SetRequestURI(request.URL)

	// TODO: delay build response
	resp := fasthttp.AcquireResponse()

	err := c.internal.Do(req, resp)
	if err != nil {
		return nil, err
	}
	defer func() {
		fasthttp.ReleaseResponse(resp)
		fasthttp.ReleaseRequest(req)
	}()

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(resp.Body()))
	if err != nil {

	}

	response := &Response{
		document: doc,
	}

	for _, fn := range c.postResponseHooks {
		if err := fn(c, response); err != nil {
			return nil, err
		}
	}

	for _, fn := range c.udPostResponseHooks {
		if err := fn(c, response); err != nil {
			return nil, err
		}
	}

	return response, nil
}

func (c *Client) transport() fasthttp.DialFunc {
	return c.internal.Dial
}
