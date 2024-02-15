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
	"golang.org/x/text/encoding/simplifiedchinese"
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
	requestHook  func(*backendClient, *Request) error
	responseHook func(*backendClient, *Response) error
)

// TODO: is this a good preactice to mixin otps for network request and custom functionality?
type clientOptionFunc optionFunc[*backendClient]

type backendClient struct {
	baseURL                 string
	header                  http.Header
	timeout                 time.Duration
	logger                  Logger
	preRequestHooks         []requestHook
	udPreRequestHooks       []requestHook
	udPreRequestHooksLock   sync.RWMutex
	postResponseHooks       []responseHook
	udPostResponseHooks     []responseHook
	udPostResponseHooksLock sync.RWMutex
	transformer             transform.Transformer

	internal    internalClient
	docCreator  documentCreator
	backoffPool *abstractPool[*exponentialBackoff]
	readerPool  *abstractPool[*bytes.Reader]
}

func newClient(opts ...clientOptionFunc) (*backendClient, error) {
	c := &backendClient{
		backoffPool: newPool[*exponentialBackoff](newExponentialBackoffFactory(
			withMinDelay(1*time.Second),
			withMaxDelay(10*time.Second),
			withMultiplier(2.0),
		)),
		readerPool: newPool[*bytes.Reader](readerFactory{}),
	}

	c.header = http.Header{}
	// TODO: pass from root level
	c.transformer = simplifiedchinese.GBK.NewDecoder()
	for _, optFn := range opts {
		if err := optFn(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func withBaseURL(url string) clientOptionFunc {
	return func(c *backendClient) error {
		c.baseURL = url
		return nil
	}
}

func withHeaders(headers map[string]string) clientOptionFunc {
	return func(c *backendClient) error {
		for h, v := range headers {
			c.header.Set(h, v)
		}
		return nil
	}
}

func withTimeout(timeout time.Duration) clientOptionFunc {
	return func(c *backendClient) error {
		if timeout < 0 {
			return errInvalidTimeout
		}
		c.timeout = timeout
		return nil
	}
}

func withClientLogger(logger Logger) clientOptionFunc {
	return func(c *backendClient) error {
		c.logger = logger
		return nil
	}
}

func withPreRequestHooks(hooks ...requestHook) clientOptionFunc {
	return func(c *backendClient) error {
		c.udPreRequestHooksLock.Lock()
		defer c.udPreRequestHooksLock.Unlock()

		c.udPreRequestHooks = append(c.udPreRequestHooks, hooks...)
		return nil
	}
}

func withPostResponseHooks(hooks ...responseHook) clientOptionFunc {
	return func(c *backendClient) error {
		c.udPostResponseHooksLock.Lock()
		defer c.udPostResponseHooksLock.Unlock()

		c.udPostResponseHooks = append(c.udPostResponseHooks, hooks...)
		return nil
	}
}

func withInternalPreRequestHooks(hooks ...requestHook) clientOptionFunc {
	return func(c *backendClient) error {
		c.preRequestHooks = append(c.preRequestHooks, hooks...)
		return nil
	}
}

func withInternalPostResponseHooks(hooks ...responseHook) clientOptionFunc {
	return func(c *backendClient) error {
		c.postResponseHooks = append(c.postResponseHooks, hooks...)
		return nil
	}
}

func withInternalClient(client internalClient) clientOptionFunc {
	return func(c *backendClient) error {
		c.internal = client
		return nil
	}
}

func withDocumentCreator(creator documentCreator) clientOptionFunc {
	return func(c *backendClient) error {
		c.docCreator = creator
		return nil
	}
}

func withBackoffPool(backoffPool *abstractPool[*exponentialBackoff]) clientOptionFunc {
	return func(c *backendClient) error {
		c.backoffPool = backoffPool
		return nil
	}
}

func withReaderPool(readerPool *abstractPool[*bytes.Reader]) clientOptionFunc {
	return func(c *backendClient) error {
		c.readerPool = readerPool
		return nil
	}
}

func withTransformer(transformer transform.Transformer) clientOptionFunc {
	return func(c *backendClient) error {
		c.transformer = transformer
		return nil
	}
}

func (c *backendClient) execute(request *Request) (*Response, error) {
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

	req := request.build()

	// TODO: delay build response
	resp := fasthttp.AcquireResponse()

	backoff := c.backoffPool.get()
	op := func() error {
		return c.internal.Do(req, resp)
	}
	err := retry(context.TODO(), op, backoff)
	c.backoffPool.put(backoff)

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
	transformer := transform.NewReader(reader, c.transformer)
	doc, err := c.docCreator.NewDocumentFromReader(transformer)
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
