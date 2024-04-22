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
type clientOptionFunc optionFunc[*Client]

type Client struct {
	baseURL string
	// TODO: consider if the header is still needed
	header http.Header

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

func newClient(opts ...clientOptionFunc) (*Client, error) {
	c := &Client{
		readerPool: newPool[*bytes.Reader](readerFactory{}),
	}
	c.header = http.Header{}

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

func withClientLogger(logger Logger) clientOptionFunc {
	return func(c *Client) error {
		c.logger = logger
		return nil
	}
}

func withInternalPreRequestHooks(hooks ...RequestHook) clientOptionFunc {
	return func(c *Client) error {
		c.preRequestHooks = append(c.preRequestHooks, hooks...)
		return nil
	}
}

func withInternalPostResponseHooks(hooks ...ResponseHook) clientOptionFunc {
	return func(c *Client) error {
		c.postResponseHooks = append(c.postResponseHooks, hooks...)
		return nil
	}
}

func withInternalClient(client internalClient) clientOptionFunc {
	return func(c *Client) error {
		c.internal = client
		return nil
	}
}

func withDocumentCreator(creator documentCreator) clientOptionFunc {
	return func(c *Client) error {
		c.docCreator = creator
		return nil
	}
}

func withReaderPool(readerPool *abstractPool[*bytes.Reader]) clientOptionFunc {
	return func(c *Client) error {
		c.readerPool = readerPool
		return nil
	}
}

func WithTransformer(transformer transform.Transformer) clientOptionFunc {
	return func(c *Client) error {
		c.transformer = transformer
		return nil
	}
}

func WithPreRequestHooks(hooks ...RequestHook) clientOptionFunc {
	return func(c *Client) error {
		c.udPreRequestHooksLock.Lock()
		defer c.udPreRequestHooksLock.Unlock()

		c.udPreRequestHooks = append(c.udPreRequestHooks, hooks...)
		return nil
	}
}

func WithPostResponseHooks(hooks ...ResponseHook) clientOptionFunc {
	return func(c *Client) error {
		c.udPostResponseHooksLock.Lock()
		defer c.udPostResponseHooksLock.Unlock()

		c.udPostResponseHooks = append(c.udPostResponseHooks, hooks...)
		return nil
	}
}

func WithBaseURL(url string) clientOptionFunc {
	return func(c *Client) error {
		c.baseURL = url
		return nil
	}
}

func WithHeaders(headers map[string]string) clientOptionFunc {
	return func(c *Client) error {
		for h, v := range headers {
			c.header.Set(h, v)
		}
		return nil
	}
}

func WithTimeout(timeout time.Duration) clientOptionFunc {
	return func(c *Client) error {
		if timeout < 0 {
			return errInvalidTimeout
		}
		c.timeout = timeout
		return nil
	}
}

func WithUserAgentGenerator(fn func() string) clientOptionFunc {
	return func(c *Client) error {
		c.preRequestHooks = append(c.preRequestHooks, func(r *Request) error {
			r.Headers["User-Agent"] = fn()
			return nil
		})
		return nil
	}
}

// Configuration functions for exponential backoff

func WithRequestMinimumDelay(d time.Duration) clientOptionFunc {
	return func(c *Client) error {
		c.exponentialBackoffOptionFuncs = append(c.exponentialBackoffOptionFuncs, WithMinDelay(d))
		return nil
	}
}

func WithRequestMaximumDelay(d time.Duration) clientOptionFunc {
	return func(c *Client) error {
		c.exponentialBackoffOptionFuncs = append(c.exponentialBackoffOptionFuncs, WithMaxDelay(d))
		return nil
	}
}

func WithRequestMultiplier(m float64) clientOptionFunc {
	return func(c *Client) error {
		c.exponentialBackoffOptionFuncs = append(c.exponentialBackoffOptionFuncs, WithMultiplier(m))
		return nil
	}
}

func WithRequestMaximumAttempt(a uint8) clientOptionFunc {
	return func(c *Client) error {
		c.exponentialBackoffOptionFuncs = append(c.exponentialBackoffOptionFuncs, WithMaxAttempt(a))
		return nil
	}
}

func WithRequestLinearAttempt(a uint8) clientOptionFunc {
	return func(c *Client) error {
		c.exponentialBackoffOptionFuncs = append(c.exponentialBackoffOptionFuncs, WithLinearAttempt(a))
		return nil
	}
}
