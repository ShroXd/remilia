package remilia

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/valyala/fasthttp"
)

type (
	RequestHook  func(*Client, *Request) error
	ResponseHook func(*Client, *Response) error
)

// TODO: is this a good preactice to mixin otps for network request and custom functionality?
type clientOptions struct {
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
}

type ClientOptionFn OptionFn[*clientOptions]

func buildClientOptions(optFns []ClientOptionFn) (*clientOptions, error) {
	opts := &clientOptions{
		header: http.Header{},
	}
	for _, optFn := range optFns {
		if err := optFn(opts); err != nil {
			return nil, err
		}
	}
	return opts, nil
}

func BaseURL(url string) ClientOptionFn {
	return func(opts *clientOptions) error {
		opts.baseURL = url
		return nil
	}
}

func Headers(headers map[string]string) ClientOptionFn {
	return func(opts *clientOptions) error {
		for h, v := range headers {
			opts.header.Set(h, v)
		}
		return nil
	}
}

func Timeout(timeout time.Duration) ClientOptionFn {
	return func(opts *clientOptions) error {
		opts.timeout = timeout
		return nil
	}
}

func ClientLogger(logger Logger) ClientOptionFn {
	return func(opts *clientOptions) error {
		opts.logger = logger
		return nil
	}
}

func PreRequestHooks(hooks ...RequestHook) ClientOptionFn {
	return func(opts *clientOptions) error {
		opts.udPreRequestHooksLock.Lock()
		defer opts.udPreRequestHooksLock.Unlock()

		opts.udPreRequestHooks = append(opts.udPreRequestHooks, hooks...)
		return nil
	}
}

func PostResponseHooks(hooks ...ResponseHook) ClientOptionFn {
	return func(opts *clientOptions) error {
		opts.udPostResponseHooksLock.Lock()
		defer opts.udPostResponseHooksLock.Unlock()

		opts.udPostResponseHooks = append(opts.udPostResponseHooks, hooks...)
		return nil
	}
}

type Client struct {
	opts     *clientOptions
	internal InternalClient
}

func NewClient(client InternalClient, optFns ...ClientOptionFn) (*Client, error) {
	opts, err := buildClientOptions(optFns)
	if err != nil {
		return nil, err
	}

	return &Client{
		opts:     opts,
		internal: client,
	}, nil
}

func (c *Client) Execute(request *Request) (*Response, error) {
	c.opts.udPreRequestHooksLock.RLock()
	defer c.opts.udPreRequestHooksLock.RUnlock()

	c.opts.udPostResponseHooksLock.RLock()
	defer c.opts.udPostResponseHooksLock.RUnlock()

	for _, fn := range c.opts.preRequestHooks {
		if err := fn(c, request); err != nil {
			return nil, err
		}
	}

	for _, fn := range c.opts.udPreRequestHooks {
		if err := fn(c, request); err != nil {
			return nil, err
		}
	}

	req := request.Build()

	// TODO: delay build response
	resp := fasthttp.AcquireResponse()

	err := c.internal.Do(req, resp)
	if err != nil {
		c.opts.logger.Error("Failed to execute request", LogContext{
			"err": err,
		})
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

	for _, fn := range c.opts.postResponseHooks {
		if err := fn(c, response); err != nil {
			return nil, err
		}
	}

	for _, fn := range c.opts.udPostResponseHooks {
		if err := fn(c, response); err != nil {
			return nil, err
		}
	}

	return response, nil
}
