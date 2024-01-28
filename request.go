package remilia

import (
	"fmt"

	"github.com/valyala/fasthttp"
)

type Request struct {
	Method      string
	URL         string
	Headers     map[string]string
	Body        []byte
	QueryParams map[string]string

	logger Logger
}

type RequestOption func(*Request) error

func WithMethod(method string) RequestOption {
	return func(req *Request) error {
		if method == "GET" || method == "POST" || method == "PUT" || method == "DELETE" {
			req.Method = method
			return nil
		} else {
			return fmt.Errorf("Invalid method: %s", method)
		}
	}
}

func WithURL(url string) RequestOption {
	return func(req *Request) error {
		req.URL = url
		return nil
	}
}

func WithHeader(key, value string) RequestOption {
	return func(req *Request) error {
		req.Headers[key] = value
		return nil
	}
}

func WithBody(body []byte) RequestOption {
	return func(req *Request) error {
		req.Body = body
		return nil
	}
}

func WithQueryParam(key, value string) RequestOption {
	return func(req *Request) error {
		req.QueryParams[key] = value
		return nil
	}
}

func NewRequest(opts ...RequestOption) (*Request, error) {
	req := &Request{
		Headers:     make(map[string]string),
		QueryParams: make(map[string]string),
	}

	for _, opt := range opts {
		err := opt(req)
		if err != nil {
			return nil, err
		}
	}

	return req, nil
}

func EmptyRequest() *Request {
	// TODO: re-use the existing request insteace
	return &Request{}
}

func (req *Request) Build() *fasthttp.Request {
	fasthttpReq := fasthttp.AcquireRequest()

	fasthttpReq.Header.SetMethod(req.Method)
	fasthttpReq.SetRequestURI(req.URL)

	for k, v := range req.Headers {
		fasthttpReq.Header.Set(k, v)
	}

	// TODO: set up all the fields

	return fasthttpReq
}
