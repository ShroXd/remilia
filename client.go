package remilia

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/valyala/fasthttp"
)

type DocumentCreator interface {
	NewDocumentFromReader(io.Reader) (*goquery.Document, error)
}

type DefaultDocumentCreator struct{}

func (d DefaultDocumentCreator) NewDocumentFromReader(r io.Reader) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(r)
}

type InternalClient interface {
	Do(req *fasthttp.Request, resp *fasthttp.Response) error
}

type ReaderFactory struct{}

func (f ReaderFactory) New() *bytes.Reader {
	return new(bytes.Reader)
}

func (f ReaderFactory) Reset(r *bytes.Reader) {
	r.Reset(nil)
}

type (
	RequestHook  func(*Client, *Request) error
	ResponseHook func(*Client, *Response) error
)

// TODO: is this a good preactice to mixin otps for network request and custom functionality?
type ClientOptionFunc OptionFunc[*Client]

type Client struct {
	baseURL                 string
	header                  http.Header
	timeout                 time.Duration
	logger                  Logger
	preRequestHooks         []RequestHook
	udPreRequestHooks       []RequestHook
	udPreRequestHooksLock   sync.RWMutex
	postResponseHooks       []ResponseHook
	udPostResponseHooks     []ResponseHook
	udPostResponseHooksLock sync.RWMutex

	internal    InternalClient
	docCreator  DocumentCreator
	backoffPool *Pool[*ExponentialBackoff]
	readerPool  *Pool[*bytes.Reader]
}

func NewClient(opts ...ClientOptionFunc) (*Client, error) {
	c := &Client{
		backoffPool: NewPool[*ExponentialBackoff](NewExponentialBackoffFactory(
			WithMinDelay(1*time.Second),
			WithMaxDelay(10*time.Second),
			WithMultiplier(2.0),
		)),
		readerPool: NewPool[*bytes.Reader](ReaderFactory{}),
	}

	c.header = http.Header{}
	for _, optFn := range opts {
		if err := optFn(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func WithBaseURL(url string) ClientOptionFunc {
	return func(c *Client) error {
		c.baseURL = url
		return nil
	}
}

func WithHeaders(headers map[string]string) ClientOptionFunc {
	return func(c *Client) error {
		for h, v := range headers {
			c.header.Set(h, v)
		}
		return nil
	}
}

func WithTimeout(timeout time.Duration) ClientOptionFunc {
	return func(c *Client) error {
		if timeout < 0 {
			return ErrInvalidTimeout
		}
		c.timeout = timeout
		return nil
	}
}

func WithClientLogger(logger Logger) ClientOptionFunc {
	return func(c *Client) error {
		c.logger = logger
		return nil
	}
}

func WithPreRequestHooks(hooks ...RequestHook) ClientOptionFunc {
	return func(c *Client) error {
		c.udPreRequestHooksLock.Lock()
		defer c.udPreRequestHooksLock.Unlock()

		c.udPreRequestHooks = append(c.udPreRequestHooks, hooks...)
		return nil
	}
}

func WithPostResponseHooks(hooks ...ResponseHook) ClientOptionFunc {
	return func(c *Client) error {
		c.udPostResponseHooksLock.Lock()
		defer c.udPostResponseHooksLock.Unlock()

		c.udPostResponseHooks = append(c.udPostResponseHooks, hooks...)
		return nil
	}
}

func WithInternalPreRequestHooks(hooks ...RequestHook) ClientOptionFunc {
	return func(c *Client) error {
		c.preRequestHooks = append(c.preRequestHooks, hooks...)
		return nil
	}
}

func WithInternalPostResponseHooks(hooks ...ResponseHook) ClientOptionFunc {
	return func(c *Client) error {
		c.postResponseHooks = append(c.postResponseHooks, hooks...)
		return nil
	}
}

func WithInternalClient(client InternalClient) ClientOptionFunc {
	return func(c *Client) error {
		c.internal = client
		return nil
	}
}

func WithDocumentCreator(creator DocumentCreator) ClientOptionFunc {
	return func(c *Client) error {
		c.docCreator = creator
		return nil
	}
}

func WithBackoffPool(backoffPool *Pool[*ExponentialBackoff]) ClientOptionFunc {
	return func(c *Client) error {
		c.backoffPool = backoffPool
		return nil
	}
}

func WithReaderPool(readerPool *Pool[*bytes.Reader]) ClientOptionFunc {
	return func(c *Client) error {
		c.readerPool = readerPool
		return nil
	}
}

func (c *Client) Execute(requestArr []*Request) (*Response, error) {
	request := requestArr[0]

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

	// TODO: delay build response
	resp := fasthttp.AcquireResponse()

	backoff := c.backoffPool.Get()
	op := func() error {
		return c.internal.Do(req, resp)
	}
	err := Retry(context.TODO(), op, backoff)
	c.backoffPool.Put(backoff)

	if err != nil {
		c.logger.Error("Failed to execute request", LogContext{
			"err": err,
		})
		return nil, err
	}
	defer func() {
		fasthttp.ReleaseResponse(resp)
		fasthttp.ReleaseRequest(req)
	}()

	reader := c.readerPool.Get()
	reader.Reset(resp.Body())
	doc, err := c.docCreator.NewDocumentFromReader(reader)
	c.readerPool.Put(reader)
	if err != nil {
		c.logger.Error("Failed to build goquery document", LogContext{
			"err": err,
		})
		return nil, err
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
