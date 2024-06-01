package remilia

import (
	"bytes"
	"context"
	"io"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/valyala/fasthttp"
	"golang.org/x/text/transform"
)

type documentCreator interface {
	NewDocumentFromReader(io.Reader) (*goquery.Document, error)
}

type defaultDocumentCreator struct{}

func (d defaultDocumentCreator) NewDocumentFromReader(r io.Reader) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(r)
}

type internalClient interface {
	Do(req *fasthttp.Request, resp *fasthttp.Response) error
}

type readerFactory struct{}

func (f readerFactory) New() *bytes.Reader {
	return new(bytes.Reader)
}

func (f readerFactory) Reset(r *bytes.Reader) {
	r.Reset(nil)
}

type (
	RequestHook  func(*Request) error
	ResponseHook func(*Response) error
)

// TODO: is this a good preactice to mixin otps for network request and custom functionality?
type ClientOptionFunc optionFunc[*Client]

type Client struct {
	baseURL string

	timeout                 time.Duration
	logger                  Logger
	preRequestHooks         []RequestHook
	udPreRequestHooks       []RequestHook
	udPreRequestHooksLock   sync.RWMutex
	postResponseHooks       []ResponseHook
	udPostResponseHooks     []ResponseHook
	udPostResponseHooksLock sync.RWMutex
	transformer             transform.Transformer

	internal   internalClient
	docCreator documentCreator
	readerPool *abstractPool[*bytes.Reader]

	exponentialBackoff            *exponentialBackoff
	exponentialBackoffOptionFuncs []exponentialBackoffOptionFunc
}

func newClient(opts ...ClientOptionFunc) (*Client, error) {
	c := &Client{
		readerPool: newPool[*bytes.Reader](readerFactory{}),
	}

	for _, optFn := range opts {
		if err := optFn(c); err != nil {
			return nil, err
		}
	}

	c.exponentialBackoff = newExponentialBackoff(c.exponentialBackoffOptionFuncs...)

	return c, nil
}

func (c *Client) execute(request *Request) (*Response, error) {
	c.udPreRequestHooksLock.RLock()
	defer c.udPreRequestHooksLock.RUnlock()

	c.udPostResponseHooksLock.RLock()
	defer c.udPostResponseHooksLock.RUnlock()

	for _, fn := range c.preRequestHooks {
		if err := fn(request); err != nil {
			return nil, err
		}
	}

	for _, fn := range c.udPreRequestHooks {
		if err := fn(request); err != nil {
			return nil, err
		}
	}

	req := request.build()

	// TODO: delay build response
	resp := fasthttp.AcquireResponse()

	op := func() error {
		return c.internal.Do(req, resp)
	}
	err := retry(context.TODO(), op, c.exponentialBackoff)

	if err != nil {
		c.logger.Error("Failed to execute request", logContext{
			"err": err,
		})
		return nil, err
	}
	defer func() {
		fasthttp.ReleaseResponse(resp)
		fasthttp.ReleaseRequest(req)
	}()

	reader := c.readerPool.get()
	reader.Reset(resp.Body())

	var doc *goquery.Document
	if c.transformer != nil {
		transformer := transform.NewReader(reader, c.transformer)
		doc, err = c.docCreator.NewDocumentFromReader(transformer)
	} else {
		doc, err = c.docCreator.NewDocumentFromReader(reader)
	}
	c.readerPool.put(reader)
	if err != nil {
		c.logger.Error("Failed to build goquery document", logContext{
			"err": err,
		})
		return nil, err
	}

	response := &Response{
		document: doc,
	}

	for _, fn := range c.postResponseHooks {
		if err := fn(response); err != nil {
			return nil, err
		}
	}

	for _, fn := range c.udPostResponseHooks {
		if err := fn(response); err != nil {
			return nil, err
		}
	}

	return response, nil
}

func withClientLogger(logger Logger) ClientOptionFunc {
	return func(c *Client) error {
		c.logger = logger
		return nil
	}
}

func withInternalPreRequestHooks(hooks ...RequestHook) ClientOptionFunc {
	return func(c *Client) error {
		c.preRequestHooks = append(c.preRequestHooks, hooks...)
		return nil
	}
}

func withInternalPostResponseHooks(hooks ...ResponseHook) ClientOptionFunc {
	return func(c *Client) error {
		c.postResponseHooks = append(c.postResponseHooks, hooks...)
		return nil
	}
}

func withInternalClient(client internalClient) ClientOptionFunc {
	return func(c *Client) error {
		c.internal = client
		return nil
	}
}

func withDocumentCreator(creator documentCreator) ClientOptionFunc {
	return func(c *Client) error {
		c.docCreator = creator
		return nil
	}
}

func withReaderPool(readerPool *abstractPool[*bytes.Reader]) ClientOptionFunc {
	return func(c *Client) error {
		c.readerPool = readerPool
		return nil
	}
}

func WithTransformer(transformer transform.Transformer) ClientOptionFunc {
	return func(c *Client) error {
		c.transformer = transformer
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

func WithBaseURL(url string) ClientOptionFunc {
	return func(c *Client) error {
		c.baseURL = url
		return nil
	}
}

func WithHeaders(headers map[string]string) ClientOptionFunc {
	return func(c *Client) error {
		c.preRequestHooks = append(c.preRequestHooks, func(r *Request) error {
			for k, v := range headers {
				r.Headers.Add(k, v)
			}
			return nil
		})
		return nil
	}
}

func WithTimeout(timeout time.Duration) ClientOptionFunc {
	return func(c *Client) error {
		if timeout < 0 {
			return errInvalidTimeout
		}
		c.timeout = timeout
		return nil
	}
}

func WithUserAgentGenerator(fn func() string) ClientOptionFunc {
	return func(c *Client) error {
		c.preRequestHooks = append(c.preRequestHooks, func(r *Request) error {
			r.Headers.Add("User-Agent", fn())
			return nil
		})
		return nil
	}
}

// Configuration functions for exponential backoff

func WithMinDelay(d time.Duration) ClientOptionFunc {
	return func(c *Client) error {
		c.exponentialBackoffOptionFuncs = append(c.exponentialBackoffOptionFuncs, WithWorkMinDelay(d))
		return nil
	}
}

func WithMaxDelay(d time.Duration) ClientOptionFunc {
	return func(c *Client) error {
		c.exponentialBackoffOptionFuncs = append(c.exponentialBackoffOptionFuncs, WithWorkMaxDelay(d))
		return nil
	}
}

func WithMultiplier(m float64) ClientOptionFunc {
	return func(c *Client) error {
		c.exponentialBackoffOptionFuncs = append(c.exponentialBackoffOptionFuncs, WithWorkMultiplier(m))
		return nil
	}
}

func WithMaxAttempt(a uint8) ClientOptionFunc {
	return func(c *Client) error {
		c.exponentialBackoffOptionFuncs = append(c.exponentialBackoffOptionFuncs, WithWorkMaxAttempt(a))
		return nil
	}
}

func WithLinearAttempt(a uint8) ClientOptionFunc {
	return func(c *Client) error {
		c.exponentialBackoffOptionFuncs = append(c.exponentialBackoffOptionFuncs, WithWorkLinearAttempt(a))
		return nil
	}
}
